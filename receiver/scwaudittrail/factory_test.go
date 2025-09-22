package scwaudittrail

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
	gomock "go.uber.org/mock/gomock"
)

func TestCreateReceiver(t *testing.T) {
	rCfg := createDefaultConfig().(*Config)

	// Fails with bad SCW Config.
	_, err := createLogsReceiver(
		context.Background(), receivertest.NewNopSettings(receivertest.NopType),
		rCfg, consumertest.NewNop(),
	)
	assert.Error(t, err)

	// Override for test.
	rCfg.makeClient = func(_ *Config) (Client, error) {
		ctrl := gomock.NewController(t)
		client := NewMockClient(ctrl)
		return client, nil
	}

	r, err := createLogsReceiver(
		context.Background(),
		receivertest.NewNopSettings(receivertest.NopType),
		rCfg, consumertest.NewNop(),
	)

	require.NoError(t, err)
	err = r.Start(context.Background(), componenttest.NewNopHost())
	assert.NoError(t, err)
	require.NoError(t, r.Shutdown(context.Background()))
}
