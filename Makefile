UNIT_PACKAGES := $(shell go list ./... | grep -v '/test/integration$$')
INTEGRATION_PACKAGES := $(shell go list ./... | grep '/test/integration$$')

.PHONY: up down unit sleep integration

up:
	docker compose up -d

down:
	docker compose down -v

unit:
	go test $(UNIT_PACKAGES)
	go clean -testcache

sleep:
	sleep 5

integration:
	$(MAKE) up
	sleep 5
	ANALYZER_BASE_URL=http://localhost:8080 FIXTURE_BASE_URL=http://web go test $(INTEGRATION_PACKAGES) -v
	go clean -testcache
	$(MAKE) down
