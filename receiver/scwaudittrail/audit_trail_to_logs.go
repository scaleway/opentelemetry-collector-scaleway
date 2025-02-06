package scwaudittrail

import (
	"encoding/json"
	"net/http"

	audit_trail "github.com/scaleway/scaleway-sdk-go/api/audit_trail/v1alpha1"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	semconv "go.opentelemetry.io/collector/semconv/v1.27.0"
	"go.uber.org/zap"
)

func auditTrailEventToLogs(logger *zap.Logger, event *audit_trail.Event) plog.Logs {
	ld := plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	lr := sl.LogRecords().AppendEmpty()

	if event.Resource != nil {
		resourceAttrs := rl.Resource().Attributes()
		resourceAttrs.PutStr("audit_trail.resource.id", event.Resource.ID)
		resourceAttrs.PutStr("audit_trail.resource.type", event.Resource.Type.String())

		if event.Resource.Name != nil {
			resourceAttrs.PutStr("audit_trail.resource.name", *event.Resource.Name)
		}
	}

	lr.SetTimestamp(pcommon.NewTimestampFromTime(*event.RecordedAt))
	lr.SetEventName(event.MethodName)

	switch event.StatusCode {
	case http.StatusOK:
		lr.SetSeverityText("success")
		lr.SetSeverityNumber(plog.SeverityNumberInfo)
	default:
		lr.SetSeverityText("failed")
		lr.SetSeverityNumber(plog.SeverityNumberError)
	}

	body, err := eventToString(event)
	if err != nil {
		logger.Warn("unable to decode event")
	} else {
		lr.Body().SetStr(body)
	}

	attrs := lr.Attributes()
	attrs.PutStr("audit_trail.event.id", event.ID)
	attrs.PutStr("audit_trail.event.locality", event.Locality)
	attrs.PutStr("audit_trail.event.source_ip", event.SourceIP.String())
	attrs.PutInt("audit_trail.event.status_code", int64(event.StatusCode))
	attrs.PutStr("audit_trail.event.request_id", event.RequestID)
	attrs.PutStr(semconv.AttributeServiceName, event.ServiceName)

	if event.UserAgent != nil {
		attrs.PutStr("audit_trail.event.user_agent", *event.UserAgent)
	}

	return ld
}

func eventToString(event *audit_trail.Event) (string, error) {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return "", err
	}

	return string(eventJSON), nil
}
