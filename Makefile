BINARY     := agencycli
BUILD_DIR  := dist
MAIN       := ./cmd/agencycli
NPM_DIR    := npm

# ── Version info (injected at link time) ──────────────────────────────────────
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.buildDate=$(BUILD_DATE)

# ── Cross-compilation targets ──────────────────────────────────────────────────
PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64 \
	windows/arm64

.PHONY: build install clean test lint release release-all $(PLATFORMS)

# ── Local build ────────────────────────────────────────────────────────────────
build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) $(MAIN)
	@echo "Built $(BUILD_DIR)/$(BINARY)  ($(VERSION))"

install:
	go install -ldflags "$(LDFLAGS)" $(MAIN)
	@echo "Installed $(BINARY) $(VERSION)"

# ── Cross-platform release ────────────────────────────────────────────────────
# Build a single platform:  make linux/amd64
$(PLATFORMS):
	$(eval OS   := $(word 1,$(subst /, ,$@)))
	$(eval ARCH := $(word 2,$(subst /, ,$@)))
	$(eval EXT  := $(if $(filter windows,$(OS)),.exe,))
	$(eval NAME := $(BINARY)-$(VERSION)-$(OS)-$(ARCH)$(EXT))
	@mkdir -p $(BUILD_DIR)
	GOOS=$(OS) GOARCH=$(ARCH) go build \
		-ldflags "$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(NAME) \
		$(MAIN)
	@echo "  ✓ $(BUILD_DIR)/$(NAME)"

# Build + archive every platform (creates .tar.gz / .zip for each)
release: $(PLATFORMS)
	@echo ""
	@echo "── Packaging archives ───────────────────────────────────────────────"
	@cd $(BUILD_DIR) && for f in $(BINARY)-$(VERSION)-*; do \
		case "$$f" in \
		  *windows*) \
		    zip -q "$$f.zip" "$$f" && echo "  ✓ $$f.zip" ;; \
		  *) \
		    tar czf "$$f.tar.gz" "$$f" && echo "  ✓ $$f.tar.gz" ;; \
		esac; \
	done
	@echo ""
	@echo "Release $(VERSION) ready in $(BUILD_DIR)/"

# ── Dev helpers ────────────────────────────────────────────────────────────────
test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR)

run: build
	./$(BUILD_DIR)/$(BINARY)

# Print version that would be stamped into the binary
version:
	@echo $(VERSION)
