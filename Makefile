MODULE := github.com/bdobrica/ThinkPixelSpider
BINARY_CLI := thinkpixelspider
BINARY_DAEMON := thinkpixelspiderd
BUILD_DIR := bin

GO := go
GOFLAGS := -trimpath
LDFLAGS := -s -w

.PHONY: all build build-cli build-daemon test lint run-cli clean

all: build

build: build-cli build-daemon

build-cli:
	$(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BUILD_DIR)/$(BINARY_CLI) ./cmd/thinkpixelspider

build-daemon:
	$(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BUILD_DIR)/$(BINARY_DAEMON) ./cmd/thinkpixelspiderd

test:
	$(GO) test -race -count=1 ./...

lint:
	golangci-lint run ./...

run-cli: build-cli
	./$(BUILD_DIR)/$(BINARY_CLI) $(ARGS)

clean:
	rm -rf $(BUILD_DIR)
