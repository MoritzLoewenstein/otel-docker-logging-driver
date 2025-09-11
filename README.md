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
# Set OTEL endpoint/envs for the plugin process
docker plugin set \
  moritzloewenstein/otel-docker-logging-driver:dev \
  OTEL_EXPORTER_OTLP_LOGS_ENDPOINT=http://localhost:4317 \
  OTEL_EXPORTER_OTLP_LOGS_INSECURE=true
```

## Configuration

- Plugin-level options (set via `docker plugin set`), defined in [plugin/config.json](plugin/config.json):

  - `OTEL_EXPORTER_OTLP_LOGS_ENDPOINT` – OTLP logs endpoint (e.g., `http://collector:4317`).
  - `OTEL_EXPORTER_OTLP_LOGS_INSECURE` – set `true` to disable TLS when using `http`.
  - `OTEL_EXPORTER_OTLP_LOGS_HEADERS` – comma-separated headers, `k=v,k2=v2`.
  - `OTEL_EXPORTER_OTLP_LOGS_COMPRESSION` – compression setting (e.g., `gzip`).

- Per-container options (set via `--log-opt` or compose `logging.options`), implemented in [internal/driver/driver.go](internal/driver/driver.go):
  - `include-labels` – `true|1|yes` to include container labels as `docker.label.<key>` attributes.
  - Note: endpoint/headers overrides per container are not yet supported; the driver logs a warning if provided (see [warnings](internal/driver/driver.go)).

### Examples

- Docker run:

```bash
docker run \
  --log-driver=moritzloewenstein/otel-docker-logging-driver:dev \
  --log-opt include-labels=true \
  --label env=dev \
  busybox sh -c 'echo hello'
```

- Compose snippet (full file at [test/integration/docker-compose.yml](test/integration/docker-compose.yml)):

```yaml
services:
  app:
    image: busybox
    logging:
      driver: "moritzloewenstein/otel-docker-logging-driver:dev"
      options:
        include-labels: "true"
```

## Integration test

- Collector config: [test/integration/collector-config.yaml](test/integration/collector-config.yaml).
- Bring up the stack:

```bash
cd test/integration
mkdir -p data
docker compose up
```

Logs should appear in collector stdout and `test/integration/data/otel-logs.json`.
