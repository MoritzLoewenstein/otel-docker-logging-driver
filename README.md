A Docker Logging Driver plugin that forwards container logs to an OpenTelemetry (OTLP) Logs endpoint.

## Install and send to gRPC logs endpoint

```bash
docker plugin install --disable moritzloewenstein/otel-docker-logging-driver
docker plugin set moritzloewenstein/otel-docker-logging-driver \
  OTEL_EXPORTER_OTLP_LOGS_ENDPOINT=http://localhost:4317 \
  OTEL_EXPORTER_OTLP_LOGS_PROTOCOL=grpc \
  OTEL_EXPORTER_OTLP_LOGS_INSECURE=true
docker plugin enable moritzloewenstein/otel-docker-logging-driver
```

## Install and send to http logs endpoint

```bash
docker plugin install --disable moritzloewenstein/otel-docker-logging-driver
docker plugin set moritzloewenstein/otel-docker-logging-driver \
  OTEL_EXPORTER_OTLP_LOGS_ENDPOINT=http://localhost:4318 \
  OTEL_EXPORTER_OTLP_LOGS_PROTOCOL=http \
  OTEL_EXPORTER_OTLP_LOGS_INSECURE=true
docker plugin enable moritzloewenstein/otel-docker-logging-driver
```

## Configuration

Plugin-level options (set via `docker plugin set`), defined in [plugin/config.json](plugin/config.json):

- `OTEL_EXPORTER_OTLP_LOGS_ENDPOINT` – OTLP logs endpoint (e.g., `http://collector:4317` or `http://collector:4318`).
- `OTEL_EXPORTER_OTLP_LOGS_PROTOCOL` – `grpc` or `http/protobuf`. Default is `grpc` for backward compatibility.
- `OTEL_EXPORTER_OTLP_LOGS_INSECURE` – set `true` to disable TLS when using `http`.
- `OTEL_EXPORTER_OTLP_LOGS_HEADERS` – comma-separated headers, `k=v,k2=v2`.
- `OTEL_EXPORTER_OTLP_LOGS_COMPRESSION` – compression setting for gRPC (e.g., `gzip`).
- TLS (file-based certificates, gRPC only):

  - `OTEL_EXPORTER_OTLP_LOGS_CERTIFICATE` – path to CA certificate PEM file enabling TLS. If unset, the generic `OTEL_EXPORTER_OTLP_CERTIFICATE` is used as a fallback. TLS creds are applied only when a CA certificate is provided (see the implementation in [internal/otelx/otel.go](internal/otelx/otel.go#L95-L114)).
  - `OTEL_EXPORTER_OTLP_LOGS_CLIENT_CERTIFICATE` – optional path to client certificate PEM for mTLS.
  - `OTEL_EXPORTER_OTLP_LOGS_CLIENT_KEY` – optional path to client private key PEM for mTLS.

Per-container options (set via `--log-opt` or compose `logging.options`), implemented in [internal/driver/driver.go](internal/driver/driver.go#L172-L187):

- `include-labels` – `true|1|yes` to include container labels as `docker.label.<key>` attributes.
- Note: endpoint/headers overrides per container are not yet supported; the driver logs a warning if provided.

The plugin server exposes a Unix socket named `otel-logs` when started by Docker Plugin runtime.

## Integration test

Collector config: [test/integration/collector-config.yaml](test/integration/collector-config.yaml).

```bash
# grpc
make plugin-test-grpc
# http
make plugin-test-http
```

Logs should appear in `test/integration/data/otel-logs.json`.
Cleanup after testing:

```bash
make plugin-remove
```
