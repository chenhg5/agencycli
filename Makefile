BINARY := agentorg
BUILD_DIR := dist
MAIN := ./cmd/agentorg

.PHONY: build install clean test lint

build:
	go build -o $(BUILD_DIR)/$(BINARY) $(MAIN)

install:
	go install $(MAIN)

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR)

run: build
	./$(BUILD_DIR)/$(BINARY)
