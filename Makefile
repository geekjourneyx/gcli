CLI_NAME ?= gcli
VERSION ?= 0.1.0
GO ?= go
GOCACHE ?= /tmp/go-build
GOMODCACHE ?= /tmp/go-mod-cache
GOLANGCI_LINT_CACHE ?= /tmp/golangci-lint-cache

.PHONY: build fmt vet lint test e2e release-check

build:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) CGO_ENABLED=1 $(GO) build -o ./bin/$(CLI_NAME) ./main.go

fmt:
	@out="$$(gofmt -l .)"; \
	if [ -n "$$out" ]; then \
		echo "gofmt check failed:"; \
		echo "$$out"; \
		exit 1; \
	fi

vet:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) $(GO) vet ./...

lint:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) GOLANGCI_LINT_CACHE=$(GOLANGCI_LINT_CACHE) golangci-lint run

test:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) CGO_ENABLED=1 $(GO) test -count=1 ./...

e2e:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) CGO_ENABLED=1 $(GO) test -count=1 ./e2e

release-check:
	./scripts/release_check.sh
