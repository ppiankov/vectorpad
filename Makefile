VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
VERSION_NUM ?= $(shell echo $(VERSION) | sed 's/^v//')
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION_NUM)"

.PHONY: build test lint clean

build:
	go build $(LDFLAGS) -o vectorpad ./cmd/vectorpad

test:
	go test -race ./...

lint:
	golangci-lint run ./...

clean:
	rm -f vectorpad
