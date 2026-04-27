##@ Core
.PHONY: help
help:  ## Display this help message.
	@echo "Usage:"
	@echo "  make [target]"
	@awk 'BEGIN {FS = ":.*?## "} \
		/^[a-zA-Z0-9_-]+:.*?## / { \
			printf "\033[36m  %-45s\033[0m %s\n", $$1, $$2 \
		} \
		/^##@/ { \
			printf "\n\033[1m%s\033[0m\n", substr($$0, 5) \
		}' $(MAKEFILE_LIST)

##@ Go sidecar

.PHONY: gen-ent
gen-ent: ## Run Ent codegen — must run after schema changes in db/ent/schema/.
	@cd db/ent && go generate ./

.PHONY: tidy
tidy: ## go mod tidy — fetch new module deps and prune unused.
	@go mod tidy

.PHONY: build-sidecar
build-sidecar: ## Build the Go sidecar for the host platform and drop into Tauri's binaries dir.
	@bash scripts/build-sidecar.sh

.PHONY: build-sidecar-darwin-arm64
build-sidecar-darwin-arm64: ## Cross-compile sidecar for macOS arm64.
	@GOOS=darwin GOARCH=arm64 bash scripts/build-sidecar.sh

.PHONY: build-sidecar-darwin-amd64
build-sidecar-darwin-amd64: ## Cross-compile sidecar for macOS amd64.
	@GOOS=darwin GOARCH=amd64 bash scripts/build-sidecar.sh

.PHONY: build-sidecar-darwin-universal
build-sidecar-darwin-universal: build-sidecar-darwin-arm64 build-sidecar-darwin-amd64 ## Build both Mac archs and lipo into a universal binary.
	@bash scripts/lipo-darwin.sh

.PHONY: build-sidecar-windows
build-sidecar-windows: ## Cross-compile sidecar for Windows amd64.
	@GOOS=windows GOARCH=amd64 bash scripts/build-sidecar.sh

.PHONY: build-sidecar-linux
build-sidecar-linux: ## Cross-compile sidecar for Linux amd64.
	@GOOS=linux GOARCH=amd64 bash scripts/build-sidecar.sh

.PHONY: test
test: ## Run Go tests.
	@go test ./...

.PHONY: vet
vet: ## Run go vet.
	@go vet ./...

##@ Frontend / Desktop (Tauri)

.PHONY: install
install: ## Install pnpm workspace dependencies.
	@pnpm install

.PHONY: desktop-kill
desktop-kill: ## Kill any leftover Vigil dev processes (orphaned tray icons, stuck sidecars).
	@killall vigil-desktop 2>/dev/null || true
	@killall vigil-sidecar 2>/dev/null || true
	@killall Vigil 2>/dev/null || true

.PHONY: desktop-dev
desktop-dev: desktop-kill build-sidecar ## Kill leftovers, build the sidecar, then run the Tauri desktop app in dev mode.
	@cd apps/desktop && pnpm tauri dev

.PHONY: desktop-build
desktop-build: build-sidecar ## Build the Tauri desktop app for the current platform.
	@cd apps/desktop && pnpm tauri build

.PHONY: desktop-build-debug
desktop-build-debug: build-sidecar ## Build the Tauri desktop app in debug mode.
	@cd apps/desktop && pnpm tauri build --debug

.PHONY: desktop-icons
desktop-icons: ## Generate Tauri app icons from a source image (apps/desktop/app-icon.png).
	@cd apps/desktop && pnpm tauri icon app-icon.png

##@ Quality

.PHONY: lint
lint: ## Run all linters via pre-commit.
	@pre-commit run -v --all-files

.PHONY: install-tools
install-tools: ## Install pre-commit hooks for this repo.
	@which pre-commit > /dev/null || echo "pre-commit not installed, see https://pre-commit.com/#install"
	@pre-commit install --install-hooks

##@ Legacy Python (will be removed once Go port reaches feature parity)

.PHONY: python-monitor
python-monitor: ## Run the legacy Python monitor (reference impl).
	@uv run pingscraper monitor

.PHONY: python-report
python-report: ## Generate a report from the legacy Python tool.
	@uv run pingscraper report
