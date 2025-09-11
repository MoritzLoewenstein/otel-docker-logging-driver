# otel-docker-logging-driver

A Docker Logging Driver plugin that forwards container logs to an OpenTelemetry (OTLP) Logs endpoint.

## Build & Run (dev)

- Build:

```bash
go build ./...
```

- Configure endpoint via environment variables (OTEL standard):

  - `OTEL_EXPORTER_OTLP_LOGS_ENDPOINT` (e.g., `http://localhost:4317`)
  - `OTEL_EXPORTER_OTLP_LOGS_INSECURE=true` (if needed)
  - `OTEL_EXPORTER_OTLP_LOGS_HEADERS=key=value`

- The plugin server exposes a Unix socket named `otel-logs` when started by Docker Plugin runtime.
