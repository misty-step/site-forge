.PHONY: build test install clean

BINARY_NAME=site-forge
GO=go
GOFLAGS=-v

build:
	$(GO) build $(GOFLAGS) -o bin/$(BINARY_NAME) ./cmd/site-forge

test:
	$(GO) test $(GOFLAGS) -v ./internal/checks/...

install:
	$(GO) install $(GOFLAGS) ./cmd/site-forge

clean:
	rm -rf bin/
	rm -f forge-report.json
	rm -rf screenshots/
