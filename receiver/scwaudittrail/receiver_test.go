package scwaudittrail

import (
	"context"
	"net"
	"testing"
	"time"

	audit_trail "github.com/scaleway/scaleway-sdk-go/api/audit_trail/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
	gomock "go.uber.org/mock/gomock"
)

func TestNewLogsReceiver(t *testing.T) {
	rCfg := createDefaultConfig().(*Config)
	rCfg.makeClient = func(_ *Config) (Client, error) {
		ctrl := gomock.NewController(t)
		client := NewMockClient(ctrl)
		return client, nil
	}

	r, err := newLogsReceiver(
		rCfg,
		receivertest.NewNopSettings(receivertest.NopType),
		consumertest.NewNop(),
	)

	require.NoError(t, err)
	require.NotNil(t, r)

	require.NoError(t, r.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, r.Shutdown(context.Background()))
}

func TestFetchEvents(t *testing.T) {
	rCfg := createDefaultConfig().(*Config)

	resp := &audit_trail.ListEventsResponse{
		Events: []*audit_trail.Event{getEvent()},
	}

	rCfg.makeClient = func(_ *Config) (Client, error) {
		ctrl := gomock.NewController(t)
		client := NewMockClient(ctrl)

		client.
			EXPECT().
			ListEvents(gomock.Any()).
			Return(resp, nil).
			AnyTimes()

		return client, nil
	}

	sink := new(consumertest.LogsSink)
	r, err := newLogsReceiver(
		rCfg,
		receivertest.NewNopSettings(receivertest.NopType),
		sink,
	)

	require.NoError(t, err)
	require.NotNil(t, r)

	recv := r.(*auditTrailReceiver)
	err = recv.fetchEvents(t.Context(), time.Now())
	assert.NoError(t, err)

	assert.Equal(t, 1, sink.LogRecordCount())
}

func getEvent() *audit_trail.Event {
	return &audit_trail.Event{
		ID:         "dd0575b4-0f9e-4398-bbfb-2dec46cd11a2",
		RecordedAt: toPtr(time.Now()),
		Locality:   "fr-par",
		Principal: &audit_trail.EventPrincipal{
			ID: "bbeeb01a-9145-46fe-8638-6ca169e64b2a",
		},
		OrganizationID: "78a3f2fa-e53b-45c0-9d71-80ea1a349a62",
		ProjectID:      toPtr("75e2a3e9-4ecd-4e33-b0e0-fc01a6e946ce"),
		SourceIP:       net.IPv4(1, 2, 3, 4),
		UserAgent:      toPtr("curl/8.11.1"),
		ProductName:    "secret-manager",
		ServiceName:    "scaleway.secret_manager.v1beta1.Api",
		MethodName:     "CreateSecret",
		Resources: []*audit_trail.Resource{
			{
				ID:        "74472a00-98e9-42b1-8195-5a40a4e1d674",
				Type:      audit_trail.ResourceTypeSecmSecret,
				CreatedAt: toPtr(time.Now()),
				UpdatedAt: toPtr(time.Now()),
				Name:      toPtr("secret-name"),
				SecmSecretInfo: &audit_trail.SecretManagerSecretInfo{
					Path: "/",
				},
			},
		},
		RequestID: "45062835-e0b2-4b48-bb22-52227542bc79",
		RequestBody: &scw.JSONObject{
			"name":       "secret-name",
			"project_id": "75e2a3e9-4ecd-4e33-b0e0-fc01a6e946ce",
		},
		StatusCode: 200,
	}
}

func toPtr[T any](v T) *T {
	return &v
}
