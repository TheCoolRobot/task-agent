BINARY := task-agent
BUILD_DIR := ./bin
MODULE := github.com/thecoolrobot/task-agent
MAIN := ./cmd/task-agent

.PHONY: build install run tui deps clean

## build: Compile the binary
build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY) $(MAIN)
	@echo "✅ Built: $(BUILD_DIR)/$(BINARY)"

## install: Install to /usr/local/bin
install: build
	cp $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)
	@echo "✅ Installed to /usr/local/bin/$(BINARY)"

## deps: Download dependencies
deps:
	go mod download
	go mod tidy

## tui: Build and launch TUI
tui: build
	$(BUILD_DIR)/$(BINARY) tui

## run: Build and run with args (make run ARGS="list")
run: build
	$(BUILD_DIR)/$(BINARY) $(ARGS)

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)

## help: Show this help
help:
	@grep -E '^##' Makefile | sed 's/## //'
