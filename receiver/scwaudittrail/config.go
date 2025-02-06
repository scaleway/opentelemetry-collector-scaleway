package scwaudittrail

import (
	"errors"
	"time"
)

type Config struct {
	// Polling frequency
	Interval string `mapstructure:"interval"`
	// Number of events to fetch per API call
	MaxEventsPerRequest uint32 `mapstructure:"max_events_per_request"`
	// Scaleway API access key
	AccessKey string `mapstructure:"access_key"`
	// Scaleway API secret key
	SecretKey string `mapstructure:"secret_key"`
	// Scaleway organization ID to monitor
	OrganizationID string `mapstructure:"organization_id"`
	// Scaleway region to monitor
	Region string `mapstructure:"region"`
	// Scaleway API URL
	APIURL string `mapstructure:"api_url"`

	// for mock
	makeClient func(*Config) (Client, error)
}

var (
	errBadInterval         = errors.New("when defined, the interval has to be set to at least 1 minute (1m)")
	errBadEventsPerRequest = errors.New("max_events_per_request must be greater or equal to 1")
)

func (cfg *Config) Validate() error {
	interval, _ := time.ParseDuration(cfg.Interval)
	if interval.Minutes() < 1 {
		return errBadInterval
	}

	if cfg.MaxEventsPerRequest < 1 {
		return errBadEventsPerRequest
	}
	return nil
}

func (cfg *Config) getScwClient() (Client, error) {
	if cfg.makeClient == nil {
		cfg.makeClient = newScalewayClient
	}

	return cfg.makeClient(cfg)
}
