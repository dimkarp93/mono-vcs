# mono-vcs (Go) — build / test / run / release.
#
# Requires the Go toolchain on PATH (this project targets Go 1.26.1). If you
# installed Go to a local prefix, e.g. ~/.local/go, run:
#   export PATH="$HOME/.local/go/bin:$PATH"

BINARY      := mono-vcs
PKG         := ./cmd/mono-vcs
DIST        := dist
GO          ?= go
# Bare semver shared with the release workflow and embedded into the binary.
VERSION     := $(shell tr -d '[:space:]' < versions.txt 2>/dev/null)
export GOWORK := off
export GOFLAGS := -mod=vendor
# Fully static, reproducible builds for client distribution.
BUILD_ENV   := CGO_ENABLED=0
CHANNEL     ?= local
# Platforms shipped to GitHub Releases (must match install.sh expectations).
PLATFORMS   := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64
# Pass arguments to `make run`, e.g.  make run ARGS='list --all'
ARGS        ?=

.DEFAULT_GOAL := help

## help: list available targets
.PHONY: help
help:
	@echo "mono-vcs — make targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
	@echo ""
	@echo "Run examples:"
	@echo "  make run ARGS='--help'"
	@echo "  make run ARGS='init'"
	@echo "  make run ARGS='list --all --no-color'"
	@echo "  make run ARGS='clone --gl-url https://gitlab.company.com'"
	@echo "  make run ARGS='pull --dry-run'"
	@echo "  make run ARGS='stash' ; make run ARGS='unstash'"
	@echo "  make run ARGS='features'"
	@echo "  make run ARGS='switch my-feature'"
	@echo "  make run ARGS='do git status -s'"

## build: compile the static binary into ./$(BINARY)
.PHONY: build
build:
	@u=$$(git remote get-url origin 2>/dev/null || true); \
	case "$$u" in \
	  "")    o=local ;; \
	  *://*) h=$${u#*://}; h=$${h#*@}; o="https://$${h%.git}" ;; \
	  *:*)   h=$${u#*@};   o="https://$$(printf '%s' "$${h%.git}" | tr ':' '/')" ;; \
	  *)     o=local ;; \
	esac; \
	if [ -f upstream.txt ]; then up=$$(tr -d '[:space:]' < upstream.txt); else up="$$o"; fi; \
	c=$$(git rev-parse --short HEAD 2>/dev/null || true); \
	$(BUILD_ENV) $(GO) build -trimpath \
	  -ldflags="-s -w -X main.version=$(VERSION) -X main.origin=$$o -X main.upstream=$$up -X main.commit=$$c -X main.channel=$(CHANNEL)" \
	  -o $(BINARY) $(PKG)

## run: build then run the binary; pass flags via ARGS='...'
.PHONY: run
run: build
	./$(BINARY) $(ARGS)

## test: run the full test suite
.PHONY: test
test:
	$(GO) test ./...

## test-v: run the test suite verbosely
.PHONY: test-v
test-v:
	$(GO) test -v ./...

## test-race: run the test suite with the race detector
.PHONY: test-race
test-race:
	$(GO) test -race ./...

## vet: run go vet
.PHONY: vet
vet:
	$(GO) vet ./...

## fmt: gofmt all sources in place
.PHONY: fmt
fmt:
	$(GO) fmt ./...

## check: vet + test (run before committing)
.PHONY: check
check: vet test

## install: install the binary to /usr/local/bin (may need sudo)
.PHONY: install
install: build
	install -m 0755 $(BINARY) /usr/local/bin/$(BINARY)

## dist: cross-compile release archives (<bin>-<os>-<arch>.tar.gz) + SHA256SUMS into ./dist
.PHONY: dist
dist: clean
	@mkdir -p $(DIST)
	@u=$$(git remote get-url origin 2>/dev/null || true); \
	case "$$u" in \
	  "")    o=local ;; \
	  *://*) h=$${u#*://}; h=$${h#*@}; o="https://$${h%.git}" ;; \
	  *:*)   h=$${u#*@};   o="https://$$(printf '%s' "$${h%.git}" | tr ':' '/')" ;; \
	  *)     o=local ;; \
	esac; \
	if [ -f upstream.txt ]; then up=$$(tr -d '[:space:]' < upstream.txt); else up="$$o"; fi; \
	c=$$(git rev-parse --short HEAD 2>/dev/null || true); \
	ld="-s -w -X main.version=$(VERSION) -X main.origin=$$o -X main.upstream=$$up -X main.commit=$$c -X main.channel=$(CHANNEL)"; \
	for target in $(PLATFORMS); do \
	  os=$${target%/*}; arch=$${target#*/}; \
	  name=$(BINARY)-$$os-$$arch; \
	  tmp=$$(mktemp -d); \
	  echo "building $$name"; \
	  $(BUILD_ENV) GOOS=$$os GOARCH=$$arch $(GO) build -trimpath -ldflags="$$ld" -o $$tmp/$(BINARY) $(PKG) || exit 1; \
	  tar -C $$tmp -czf $(DIST)/$$name.tar.gz $(BINARY); \
	  rm -rf $$tmp; \
	done
	@cd $(DIST) && sha256sum *.tar.gz > SHA256SUMS
	@echo "built (version $(VERSION)):" && ls -1 $(DIST)

## version: print the current version from versions.txt
.PHONY: version
version:
	@cat versions.txt

## bump-patch: increment the patch version in versions.txt (X.Y.Z -> X.Y.Z+1)
.PHONY: bump-patch
bump-patch:
	@awk -F. '{printf "%d.%d.%d\n",$$1,$$2,$$3+1}' versions.txt > versions.tmp && mv versions.tmp versions.txt && cat versions.txt
	@$(MAKE) --no-print-directory _bump-commit LEVEL=patch

## bump-minor: increment the minor version in versions.txt (X.Y.Z -> X.Y+1.0)
.PHONY: bump-minor
bump-minor:
	@awk -F. '{printf "%d.%d.%d\n",$$1,$$2+1,0}' versions.txt > versions.tmp && mv versions.tmp versions.txt && cat versions.txt
	@$(MAKE) --no-print-directory _bump-commit LEVEL=minor

## bump-major: increment the major version in versions.txt (X.Y.Z -> X+1.0.0)
.PHONY: bump-major
bump-major:
	@awk -F. '{printf "%d.%d.%d\n",$$1+1,0,0}' versions.txt > versions.tmp && mv versions.tmp versions.txt && cat versions.txt
	@$(MAKE) --no-print-directory _bump-commit LEVEL=major

.PHONY: _bump-commit
_bump-commit:
	@v=$$(tr -d '[:space:]' < versions.txt); \
	if git rev-parse -q --verify "refs/tags/v$$v" >/dev/null; then \
		git checkout -- versions.txt; echo "tag v$$v already exists" >&2; exit 1; \
	fi; \
	git commit -q -m "bump $(LEVEL)" -- versions.txt && git tag "v$$v" || exit 1; \
	rc=0; for r in $$(git remote); do \
		git push -q "$$r" HEAD --tags || { echo "push to $$r failed" >&2; rc=1; }; \
	done; \
	echo "Tagged v$$v"; exit $$rc

## clean: remove build artifacts
.PHONY: clean
clean:
	rm -f $(BINARY)
	rm -rf $(DIST)

.PHONY: vendor
vendor:
	GOWORK=off go mod tidy
	GOWORK=off go mod vendor

.PHONY: vendor-check
vendor-check:
	GOWORK=off go mod vendor
	test -z "$$(git status --porcelain -- go.mod go.sum vendor/ | tee /dev/stderr)"
