package scwaudittrail

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	audit_trail "github.com/scaleway/scaleway-sdk-go/api/audit_trail/v1alpha1"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
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
	events, err := r.fetchEvents(until)
	if err != nil {
		return err
	}

	for _, event := range events {
		r.handleEvent(ctx, event)
	}

	r.lastFetchedAt = until

	return nil
}

// fetchEvents uses scw sdk to fetch events.
func (r *auditTrailReceiver) fetchEvents(until time.Time) ([]*audit_trail.Event, error) {
	events := make([]*audit_trail.Event, 0)
	var nextPageToken *string

	for {
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
			r.settings.Logger.Error("Failed to fetch events", zap.Error(err))
			return nil, err
		}

		r.settings.Logger.Debug("Events fetched", zap.Int("count", len(response.Events)))
		events = append(events, response.Events...)

		if response.NextPageToken == nil {
			break
		}

		nextPageToken = response.NextPageToken
	}

	return events, nil
}

// handleEvent converts audit trail event to logs and forward it to the consumer
func (r *auditTrailReceiver) handleEvent(ctx context.Context, event *audit_trail.Event) {
	ctx = r.obsrecv.StartLogsOp(ctx)
	logs := auditTrailEventToLogs(r.settings.Logger, event)
	consumerErr := r.consumer.ConsumeLogs(ctx, logs)
	r.obsrecv.EndLogsOp(ctx, "audit_trail_events", 1, consumerErr)
}

// Shutdown stops the receiver.
func (r *auditTrailReceiver) Shutdown(_ context.Context) error {
	r.settings.Logger.Debug("shutting down logs receiver")
	close(r.stopChan)
	r.wg.Wait()
	return nil
}
