package scwaudittrail

import (
	"context"
	"testing"
	"time"

	audit_trail "github.com/scaleway/scaleway-sdk-go/api/audit_trail/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
	gomock "go.uber.org/mock/gomock"
)

func TestNewLogsReceiver(t *testing.T) {
	rCfg := createDefaultConfig().(*Config)
	rCfg.makeClient = func(c *Config) (Client, error) {
		ctrl := gomock.NewController(t)
		client := NewMockClient(ctrl)
		return client, nil
	}

	r, err := newLogsReceiver(
		rCfg,
		receivertest.NewNopSettings(),
		consumertest.NewNop(),
	)

	require.NoError(t, err)
	require.NotNil(t, r)

	require.NoError(t, r.Start(context.Background(), componenttest.NewNopHost()))
	assert.NoError(t, r.Shutdown(context.Background()))
}

func TestHandleEvent(t *testing.T) {
	rCfg := createDefaultConfig().(*Config)
	rCfg.makeClient = func(c *Config) (Client, error) {
		ctrl := gomock.NewController(t)
		client := NewMockClient(ctrl)
		return client, nil
	}

	sink := new(consumertest.LogsSink)
	r, err := newLogsReceiver(
		rCfg,
		receivertest.NewNopSettings(),
		sink,
	)

	require.NoError(t, err)
	require.NotNil(t, r)

	recv := r.(*auditTrailReceiver)
	auditTrailEvent := getEvent()
	recv.handleEvent(context.Background(), auditTrailEvent)

	assert.Equal(t, 1, sink.LogRecordCount())
}

func TestFetchEvents(t *testing.T) {
	rCfg := createDefaultConfig().(*Config)

	resp := &audit_trail.ListEventsResponse{
		Events: []*audit_trail.Event{getEvent()},
	}

	rCfg.makeClient = func(c *Config) (Client, error) {
		ctrl := gomock.NewController(t)
		client := NewMockClient(ctrl)

		client.
			EXPECT().
			ListEvents(gomock.Any()).
			Return(resp, nil).
			AnyTimes()

		return client, nil
	}

	r, err := newLogsReceiver(
		rCfg,
		receivertest.NewNopSettings(),
		consumertest.NewNop(),
	)

	require.NoError(t, err)
	require.NotNil(t, r)

	recv := r.(*auditTrailReceiver)
	events, err := recv.fetchEvents(time.Now())

	assert.NoError(t, err)
	assert.Equal(t, resp.Events, events)
}
