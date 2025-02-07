# Scaleway components for OpenTelemetry Collector

This is a repository for OpenTelemetry Collector components for Scaleway.

The official distributions, core and contrib, are available as part of the [opentelemetry-collector-releases](https://github.com/open-telemetry/opentelemetry-collector-releases) repository.

Users of the OpenTelemetry Collector must build their own custom distributions with the [OpenTelemetry Collector Builder](https://github.com/open-telemetry/opentelemetry-collector/tree/main/cmd/builder), using the components they need from the core repository, the contrib repository, and this repository.

## Components

* [![Go Report Card](https://goreportcard.com/badge/github.com/scaleway/opentelemetry-collector-scaleway/receiver/scwaudittrail)](https://goreportcard.com/report/github.com/scaleway/opentelemetry-collector-scaleway/receiver/scwaudittrail) [receiver/scwaudittrail](./receiver/scwaudittrail/)
