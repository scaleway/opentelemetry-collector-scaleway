# Scaleway Audit Trail receiver

The Scaleway Audit Trail receiver collects audit trail events from your scaleway organization.

## Configuration

The following config is required:

- `access_key` Scaleway access key (from API key)
- `secret_key` Scaleway secret key (from api key)
- `organization_id` Scaleway organization ID to monitor
- `region` Scaleway region to monitor

The following config is optional:

- `interval` Polling frequency (defaults to 1 minute, must be at least 1 minute)
- `max_events_per_request` Number of events to process per poll (defaults to 100)

Examples:

```yaml
  scwaudittrail:
    access_key: ${env:SCW_ACCESS_KEY}
    secret_key: ${env:SCW_ACCESS_KEY}
    organization_id: ${env:SCW_DEFAULT_ORGANIZATION_ID}
    region: ${env:SCW_DEFAULT_REGION}
```

The full list of settings exposed for this receiver are documented in [config.go](./config.go).

## Examples

Here is an example configuration for the collector using this receiver and the [OTLP Exporter](https://github.com/open-telemetry/opentelemetry-collector/blob/main/exporter/otlpexporter/README.md).

```yaml
receivers:
  scwaudittrail:
    access_key: ${env:SCW_ACCESS_KEY}
    secret_key: ${env:SCW_ACCESS_KEY}
    organization_id: ${env:SCW_DEFAULT_ORGANIZATION_ID}
    region: ${env:SCW_DEFAULT_REGION}

exporters:
  otlp:
    endpoint: <OTLP_ENDPOINT>
    tls:
      insecure: true

service:
  pipelines:
    logs:
      receivers: [scwaudittrail]
      exporters: [otlp]
```
