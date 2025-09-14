.PHONY: plugin-build plugin-create plugin-enable plugin-disable plugin-remove plugin-up plugin-logs test test-unit test-e2e lint lint-fix

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

test-unit:
	go test ./... -v

test-e2e:
	DOCKER_INTEGRATION=1 go test -tags=integration ./test/integration -v

test:
	DOCKER_INTEGRATION=1 go test -tags=integration ./... -v


lint:
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint not installed. See https://golangci-lint.run/docs/welcome/install/"; \
		exit 1; \
	}
	golangci-lint run

lint-fix:
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint not installed. See https://golangci-lint.run/docs/welcome/install/"; \
		exit 1; \
	}
	golangci-lint run --fix
