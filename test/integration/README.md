# Integration test

This launches an OpenTelemetry Collector and a demo container that emits logs. You must have the plugin created and enabled on your Docker daemon first.

## Steps

1. Build and install the plugin:

```bash
make plugin-redeploy
# Configure endpoint to your collector (here compose exposes 4317 on localhost)
docker plugin set moritzloewenstein/otel-docker-logging-driver:dev OTEL_EXPORTER_OTLP_LOGS_ENDPOINT=http://localhost:4317 OTEL_EXPORTER_OTLP_LOGS_INSECURE=true
```

2. Start the collector and demo app:

```bash
cd test/integration
mkdir -p data
docker compose up --force-recreate --remove-orphans
```

3. Verify

- Collector will print received logs to stdout.
- A file with logs is written at `test/integration/data/otel-logs.json`.
- Demo container uses `include-labels=true`; expect attributes like `docker.label.test.label=demo`.

4. Cleanup

```bash
docker compose down -v
make plugin-remove PLUGIN_NAME=moritzloewenstein/otel-docker-logging-driver PLUGIN_TAG=dev
```
