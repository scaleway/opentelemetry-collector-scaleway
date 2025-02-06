package scwaudittrail

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

var (
	typeStr = component.MustNewType("scwaudittrail")
)

const (
	defaultInterval   = 1 * time.Minute
	defaultEventLimit = 100
)

func createDefaultConfig() component.Config {
	return &Config{
		Interval:            defaultInterval.String(),
		MaxEventsPerRequest: defaultEventLimit,
		APIURL:              "https://api.scaleway.com",
	}
}

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithLogs(createLogsReceiver, component.StabilityLevelDevelopment),
	)
}

func createLogsReceiver(
	_ context.Context,
	params receiver.Settings,
	rConf component.Config,
	consumer consumer.Logs,
) (receiver.Logs, error) {
	cfg := rConf.(*Config)
	return newLogsReceiver(cfg, params, consumer)
}
