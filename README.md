# otel-docker-logging-driver

A Docker Logging Driver plugin that forwards container logs to an OpenTelemetry (OTLP) Logs endpoint.

## Build & Run (dev)

```bash
make plugin-up
```

- Configure endpoint via environment variables (OTEL standard):

  - `OTEL_EXPORTER_OTLP_LOGS_ENDPOINT` (e.g., `http://localhost:4317` for gRPC or `http://localhost:4318` for HTTP)
  - `OTEL_EXPORTER_OTLP_LOGS_PROTOCOL` (`grpc` or `http/protobuf`; default is `grpc`)
  - `OTEL_EXPORTER_OTLP_LOGS_INSECURE=true` (if needed)
  - `OTEL_EXPORTER_OTLP_LOGS_HEADERS=key=value`

- The plugin server exposes a Unix socket named `otel-logs` when started by Docker Plugin runtime.

## Install the plugin (prod)

```bash
docker plugin install moritzloewenstein/otel-docker-logging-driver
```

## Install the plugin (local dev)

- Build, replace, and enable the plugin in one step:

```bash
make plugin-up
```

- Verify and configure endpoint to a local collector:

```bash
docker plugin inspect moritzloewenstein/otel-docker-logging-driver:dev

# gRPC example
docker plugin set \
  moritzloewenstein/otel-docker-logging-driver:dev \
  OTEL_EXPORTER_OTLP_LOGS_ENDPOINT=http://localhost:4317 \
  OTEL_EXPORTER_OTLP_LOGS_PROTOCOL=grpc \
  OTEL_EXPORTER_OTLP_LOGS_INSECURE=true

# HTTP example
docker plugin set \
  moritzloewenstein/otel-docker-logging-driver:dev \
  OTEL_EXPORTER_OTLP_LOGS_ENDPOINT=http://localhost:4318 \
  OTEL_EXPORTER_OTLP_LOGS_PROTOCOL=http/protobuf
```

## Configuration

- Plugin-level options (set via `docker plugin set`), defined in [plugin/config.json](plugin/config.json):

  - `OTEL_EXPORTER_OTLP_LOGS_ENDPOINT` – OTLP logs endpoint (e.g., `http://collector:4317` or `http://collector:4318`).
  - `OTEL_EXPORTER_OTLP_LOGS_PROTOCOL` – `grpc` or `http/protobuf`. Default is `grpc` for backward compatibility.
  - `OTEL_EXPORTER_OTLP_LOGS_INSECURE` – set `true` to disable TLS when using `http`.
  - `OTEL_EXPORTER_OTLP_LOGS_HEADERS` – comma-separated headers, `k=v,k2=v2`.
  - `OTEL_EXPORTER_OTLP_LOGS_COMPRESSION` – compression setting for gRPC (e.g., `gzip`).
  - TLS (file-based certificates, gRPC only):
    - `OTEL_EXPORTER_OTLP_LOGS_CERTIFICATE` – path to CA certificate PEM file enabling TLS. If unset, the generic `OTEL_EXPORTER_OTLP_CERTIFICATE` is used as a fallback. TLS creds are applied only when a CA certificate is provided (see the implementation in [internal/otelx/otel.go](internal/otelx/otel.go#L95-L114)).
    - `OTEL_EXPORTER_OTLP_LOGS_CLIENT_CERTIFICATE` – optional path to client certificate PEM for mTLS.
    - `OTEL_EXPORTER_OTLP_LOGS_CLIENT_KEY` – optional path to client private key PEM for mTLS.

- Per-container options (set via `--log-opt` or compose `logging.options`), implemented in [internal/driver/driver.go](internal/driver/driver.go#L172-L187):
  - `include-labels` – `true|1|yes` to include container labels as `docker.label.<key>` attributes.
  - Note: endpoint/headers overrides per container are not yet supported; the driver logs a warning if provided.

## Integration test

- Collector config: [test/integration/collector-config.yaml](test/integration/collector-config.yaml).

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
