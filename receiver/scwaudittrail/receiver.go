package scwaudittrail

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	audit_trail "github.com/scaleway/scaleway-sdk-go/api/audit_trail/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.uber.org/zap"
)

// auditTrailReceiver implements a custom OpenTelemetry log receiver.
type auditTrailReceiver struct {
	settings      receiver.Settings
	config        *Config
	consumer      consumer.Logs
	stopChan      chan struct{}
	wg            *sync.WaitGroup
	pollInterval  time.Duration
	obsrecv       *receiverhelper.ObsReport
	lastFetchedAt time.Time
	client        Client
}

// newLogsReceiver creates a new instance of the Audit Trail receiver.
func newLogsReceiver(config *Config, set receiver.Settings, consumer consumer.Logs) (receiver.Logs, error) {
	interval, _ := time.ParseDuration(config.Interval)

	obsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		ReceiverID:             set.ID,
		Transport:              "http",
		ReceiverCreateSettings: set,
	})
	if err != nil {
		return nil, err
	}

	client, err := config.getScwClient()
	if err != nil {
		set.Logger.Error("Failed to create scaleway client", zap.Error(err))
		return nil, err
	}

	return &auditTrailReceiver{
		settings:      set,
		config:        config,
		consumer:      consumer,
		stopChan:      make(chan struct{}),
		wg:            &sync.WaitGroup{},
		pollInterval:  interval,
		obsrecv:       obsrecv,
		lastFetchedAt: time.Now(),
		client:        client,
	}, nil
}

// Start begins polling the Audit Trail API for events.
func (r *auditTrailReceiver) Start(ctx context.Context, _ component.Host) error {
	r.settings.Logger.Info("Starting Audit Trail receiver")
	r.wg.Add(1)
	go r.startPolling(ctx)
	return nil
}

func (r *auditTrailReceiver) startPolling(ctx context.Context) {
	defer r.wg.Done()

	t := time.NewTicker(r.pollInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopChan:
			return
		case until := <-t.C:
			r.settings.Logger.Info("Polling Audit Trail logs")
			err := r.poll(ctx, until)
			if err != nil {
				r.settings.Logger.Error("there was an error during the poll", zap.Error(err))
			}
		}
	}
}

// poll fetches events from the Audit Trail API and sends them to the consumer.
func (r *auditTrailReceiver) poll(ctx context.Context, until time.Time) error {
	err := r.fetchEvents(ctx, until)
	if err != nil {
		return err
	}

	r.lastFetchedAt = until

	return nil
}

// fetchEvents uses scw sdk to fetch events.
func (r *auditTrailReceiver) fetchEvents(ctx context.Context, until time.Time) error {
	// init first token to empty string so that first loop can proceed
	nextPageToken := scw.StringPtr("")

	for nextPageToken != nil {
		select {
		case _, ok := <-r.stopChan:
			if !ok {
				return nil
			}
		default:
			r.settings.Logger.Debug(
				"List events",
				zap.Time("after", r.lastFetchedAt),
				zap.Time("before", until),
				zap.Stringp("page_token", nextPageToken),
			)

			response, err := r.client.ListEvents(&audit_trail.ListEventsRequest{
				PageSize:       &r.config.MaxEventsPerRequest,
				OrderBy:        audit_trail.ListEventsRequestOrderByRecordedAtAsc,
				RecordedAfter:  &r.lastFetchedAt,
				RecordedBefore: &until,
				PageToken:      nextPageToken,
			})
			if err != nil {
				return err
			}

			r.settings.Logger.Debug("Events fetched", zap.Int("count", len(response.Events)))

			logs := r.processEvents(response)
			err = r.consumeLogs(ctx, logs)
			if err != nil {
				return err
			}

			nextPageToken = response.NextPageToken
		}
	}

	return nil
}

// processEvents transforms a list of events into a plog.Logs object.
func (r *auditTrailReceiver) processEvents(resp *audit_trail.ListEventsResponse) plog.Logs {
	ld := plog.NewLogs()

	resourceMap := map[string]*plog.ResourceLogs{}

	for _, event := range resp.Events {
		resourceLogs, ok := resourceMap[event.ServiceName]
		if !ok {
			rl := ld.ResourceLogs().AppendEmpty()
			resourceLogs = &rl
			resourceAttrs := resourceLogs.Resource().Attributes()
			resourceAttrs.PutStr(string(semconv.ServiceNameKey), event.ServiceName)

			_ = resourceLogs.ScopeLogs().AppendEmpty()
			resourceMap[event.ServiceName] = resourceLogs
		}

		lr := resourceLogs.ScopeLogs().At(0).LogRecords().AppendEmpty()

		lr.SetTimestamp(pcommon.NewTimestampFromTime(*event.RecordedAt))
		lr.SetEventName(event.MethodName)

		switch event.StatusCode {
		case http.StatusOK, http.StatusCreated, http.StatusNoContent:
			lr.SetSeverityText("success")
			lr.SetSeverityNumber(plog.SeverityNumberInfo)
		default:
			lr.SetSeverityText("failed")
			lr.SetSeverityNumber(plog.SeverityNumberError)
		}

		body, err := json.Marshal(event)
		if err != nil {
			r.settings.Logger.Warn("unable to decode event")
		} else {
			lr.Body().SetStr(string(body))
		}

		attrs := lr.Attributes()
		attrs.PutStr("id", event.ID)
		attrs.PutStr("audit_trail.event.locality", event.Locality)
		attrs.PutStr("audit_trail.event.source_ip", event.SourceIP.String())
		attrs.PutInt("audit_trail.event.status_code", int64(event.StatusCode))
		attrs.PutStr("audit_trail.event.request_id", event.RequestID)

		if event.UserAgent != nil {
			attrs.PutStr("audit_trail.event.user_agent", *event.UserAgent)
		}
	}

	return ld
}

// consumeLogs forward logs to the consumer
func (r *auditTrailReceiver) consumeLogs(ctx context.Context, logs plog.Logs) error {
	ctx = r.obsrecv.StartLogsOp(ctx)
	err := r.consumer.ConsumeLogs(ctx, logs)
	r.obsrecv.EndLogsOp(ctx, "audit_trail_events", logs.LogRecordCount(), err)
	return err
}

// Shutdown stops the receiver.
func (r *auditTrailReceiver) Shutdown(_ context.Context) error {
	r.settings.Logger.Debug("shutting down logs receiver")
	close(r.stopChan)
	r.wg.Wait()
	return nil
}
