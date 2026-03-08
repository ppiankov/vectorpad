VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
VERSION_NUM ?= $(shell echo $(VERSION) | sed 's/^v//')
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION_NUM)"

.PHONY: build test coverage lint clean fmt

build:
	go build $(LDFLAGS) -o vectorpad ./cmd/vectorpad

test:
	go test -race ./...

coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1
	@echo "Detail: go tool cover -html=coverage.out"

lint:
	golangci-lint run ./...

clean:
	rm -f vectorpad
