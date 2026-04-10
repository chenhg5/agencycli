BINARY     := agencycli
BUILD_DIR  := dist
MAIN       := ./cmd/agencycli
NPM_DIR    := npm
WEB_DIR    := web

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

.PHONY: build build-go install deploy clean test lint release release-all web web-install web-dev $(PLATFORMS)

# ── Local deploy paths ────────────────────────────────────────────────────────
INSTALL_BIN ?= /root/.local/bin/agencycli
AGENCY_DIR  ?= /root/code/TechStudio

# ── Web frontend ──────────────────────────────────────────────────────────────

web-install:
	@echo "── Installing web dependencies ──────────────────────────────────────"
	cd $(WEB_DIR) && npm install --prefer-offline

web: web-install
	@echo "── Building web frontend ────────────────────────────────────────────"
	cd $(WEB_DIR) && npx vite build
	@echo "  ✓ $(WEB_DIR)/dist ready (embedded into Go binary via //go:embed)"

web-dev: web-install
	cd $(WEB_DIR) && npx vite dev

# ── Local build ────────────────────────────────────────────────────────────────

build-go:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) $(MAIN)
	@echo "  ✓ $(BUILD_DIR)/$(BINARY)  ($(VERSION))"

build: web build-go
	@echo ""
	@echo "Build complete: $(BUILD_DIR)/$(BINARY) ($(VERSION))"
	@echo "  Web console embedded. Run: ./$(BUILD_DIR)/$(BINARY) start"

install: web
	go install -ldflags "$(LDFLAGS)" $(MAIN)
	@echo "Installed $(BINARY) $(VERSION) (with web console)"

# ── One-step local deploy (build → install → restart supervisor) ──────────────

deploy: build
	@mkdir -p $(dir $(INSTALL_BIN))
	cp -f $(BUILD_DIR)/$(BINARY) $(INSTALL_BIN)
	@echo "  ✓ Installed to $(INSTALL_BIN)"
	@if supervisorctl status agencycli >/dev/null 2>&1; then \
		supervisorctl restart agencycli && echo "  ✓ Restarted supervisor:agencycli"; \
	else \
		echo "  ⚠ supervisor program 'agencycli' not found — run: supervisorctl reread && supervisorctl update"; \
	fi
	@echo ""
	@echo "Deploy complete: $(INSTALL_BIN) ($(VERSION))"

# ── Cross-platform release ────────────────────────────────────────────────────
# Web is built once; each platform embeds the same dist/ via go:embed.

$(PLATFORMS): web
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

# Build + archive every platform
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
	rm -rf $(WEB_DIR)/dist $(WEB_DIR)/node_modules/.vite

run: build
	./$(BUILD_DIR)/$(BINARY) start

# Print version that would be stamped into the binary
version:
	@echo $(VERSION)
