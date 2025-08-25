GO=go
BINARY_NAME=anitr-cli
BUILD_DIR=./build

INSTALL_DIR_LINUX=/usr/bin
INSTALL_DIR_WINDOWS="C:/Program Files/anitr-cli"
INSTALL_DIR_MAC=/usr/local/bin

VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
BUILDENV := $(shell go version)

LDFLAGS=-ldflags="-X 'github.com/axrona/anitr-cli/internal/update.version=$(VERSION)' -X 'github.com/axrona/anitr-cli/internal/update.buildEnv=$(BUILDENV)'"

mod-tidy:
	$(GO) mod tidy

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

build-linux:
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-x86_64
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64

build-windows:
	mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-x86_64.exe

build-macos:
	mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-macos-amd64
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-macos-arm64

build: mod-tidy
	mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)

build-all: mod-tidy build-linux build-windows build-macos

install-linux: build
	chmod +x $(BUILD_DIR)/$(BINARY_NAME)
	sudo mv $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR_LINUX)/$(BINARY_NAME)

install-windows: build
	powershell -Command "New-Item -ItemType Directory -Force -Path $(INSTALL_DIR_WINDOWS)"
	powershell -Command "Copy-Item -Path $(BUILD_DIR)/$(BINARY_NAME) -Destination $(INSTALL_DIR_WINDOWS)/$(BINARY_NAME).exe -Force"

install-macos: build
	sudo mv $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR_MAC)/$(BINARY_NAME)

clean:
	rm -rf $(BUILD_DIR)

all: build-all install
