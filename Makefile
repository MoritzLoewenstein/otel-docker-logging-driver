.PHONY: plugin-build plugin-create plugin-enable plugin-disable plugin-remove plugin-up plugin-logs plugin-test-grpc plugin-test-http

PLUGIN_NAME?=moritzloewenstein/otel-docker-logging-driver
PLUGIN_TAG?=dev
IMG?=$(PLUGIN_NAME)-rootfs:$(PLUGIN_TAG)
PKG_DIR?=$(CURDIR)/plugin/package
ROOTFS_DIR?=$(PKG_DIR)/rootfs

plugin-build:
	@docker build -f plugin/Dockerfile.plugin -t $(IMG) .
	@rm -rf $(ROOTFS_DIR)
	@mkdir -p $(ROOTFS_DIR)
	@id=$$(docker create $(IMG)); \
	docker export $$id | tar -C $(ROOTFS_DIR) -xvf - > /dev/null; \
	docker rm -v $$id > /dev/null
	@cp plugin/config.json $(PKG_DIR)/config.json

plugin-create: plugin-build
	@docker plugin create $(PLUGIN_NAME):$(PLUGIN_TAG) $(PKG_DIR)

plugin-enable:
	@docker plugin enable $(PLUGIN_NAME):$(PLUGIN_TAG)

plugin-disable:
	@docker plugin disable -f $(PLUGIN_NAME):$(PLUGIN_TAG)

plugin-remove:
	-@docker plugin disable -f $(PLUGIN_NAME):$(PLUGIN_TAG) >/dev/null 2>&1 || true
	@docker plugin rm -f $(PLUGIN_NAME):$(PLUGIN_TAG)

plugin-up:
	@$(MAKE) plugin-build
	-@docker plugin disable -f $(PLUGIN_NAME):$(PLUGIN_TAG) >/dev/null 2>&1 || true
	-@docker plugin rm -f $(PLUGIN_NAME):$(PLUGIN_TAG) >/dev/null 2>&1 || true
	@docker plugin create $(PLUGIN_NAME):$(PLUGIN_TAG) $(PKG_DIR)
	@docker plugin enable $(PLUGIN_NAME):$(PLUGIN_TAG)

plugin-logs:
	@ID=$$(docker plugin inspect -f '{{.ID}}' $(PLUGIN_NAME):$(PLUGIN_TAG)); \
	if [ -z "$$ID" ]; then echo "Plugin $(PLUGIN_NAME):$(PLUGIN_TAG) not found"; exit 1; fi; \
	echo "Following Docker daemon logs for plugin $$ID (Manjaro/systemd). Ctrl-C to stop..."; \
	sudo journalctl -u docker.service -f -o cat | egrep "$$ID|otel-docker-logging-driver|otelx:|consume:|emit:"

plugin-test-grpc:
	@$(MAKE) plugin-up
	-@docker plugin disable -f $(PLUGIN_NAME):$(PLUGIN_TAG) >/dev/null 2>&1 || true
	@docker plugin set $(PLUGIN_NAME):$(PLUGIN_TAG) OTEL_EXPORTER_OTLP_LOGS_ENDPOINT=http://localhost:4317 OTEL_EXPORTER_OTLP_LOGS_INSECURE=true
	@docker plugin enable $(PLUGIN_NAME):$(PLUGIN_TAG)
	cd test/integration && docker compose up

plugin-test-http:
	@$(MAKE) plugin-up
	-@docker plugin disable -f $(PLUGIN_NAME):$(PLUGIN_TAG) >/dev/null 2>&1 || true
	@docker plugin set $(PLUGIN_NAME):$(PLUGIN_TAG) OTEL_EXPORTER_OTLP_LOGS_ENDPOINT=http://localhost:4318 OTEL_EXPORTER_OTLP_LOGS_PROTOCOL=http/protobuf
	@docker plugin enable $(PLUGIN_NAME):$(PLUGIN_TAG)
	cd test/integration && docker compose up
