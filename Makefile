.PHONY: test-fast test-full test-long test-emulator test-commands check-ym2608-manifest check-ym2608-manifest-strict check-ym2608-policy ci-audio release-linux

GO ?= go
GO_TEST_TAGS ?= no_sdl_ttf
GO_TEST_COMMON = $(GO) test -tags $(GO_TEST_TAGS)

# Fast local checks (skip long-running emulator timing tests)
test-fast:
	$(GO_TEST_COMMON) ./internal/apu ./internal/corelx ./internal/cpu ./internal/input ./internal/ppu -timeout 60s

# Emulator package (includes some slower integration/timing tests)
test-emulator:
	$(GO_TEST_COMMON) ./internal/emulator -timeout 120s

# Command package build sanity checks (via go test package compile)
test-commands:
	$(GO_TEST_COMMON) ./cmd/... -timeout 120s

# Full default local baseline (works without SDL2_ttf dev libs)
test-full:
	$(GO_TEST_COMMON) ./... -timeout 120s

# Explicit long-running timing tests
test-long:
	$(GO_TEST_COMMON) ./internal/emulator -run 'TestAudioTimingLongRun|TestAudioTimingFractionalAccumulator' -v -timeout 180s

# YM2608 extraction manifest parity check (fails on deterministic drift)
check-ym2608-manifest:
	@if [ -f ./scripts/check_ym2608_extraction_manifest.sh ]; then \
		bash ./scripts/check_ym2608_extraction_manifest.sh; \
	else \
		echo "Skipping YM2608 manifest check (scripts/check_ym2608_extraction_manifest.sh missing)"; \
	fi

# Optional strict parity check: requires commit pins when git metadata exists in source repos.
check-ym2608-manifest-strict:
	@if [ -f ./scripts/check_ym2608_extraction_manifest.sh ]; then \
		YM2608_STRICT_COMMIT_PINS=1 bash ./scripts/check_ym2608_extraction_manifest.sh; \
	else \
		echo "Skipping strict YM2608 manifest check (scripts/check_ym2608_extraction_manifest.sh missing)"; \
	fi

# Consolidated policy-validation bundle (default + strict + policy tests)
check-ym2608-policy: check-ym2608-manifest check-ym2608-manifest-strict
	@if [ -d ./cmd/ym2608_manifest_gen ]; then \
		$(GO_TEST_COMMON) ./cmd/ym2608_manifest_gen -timeout 120s; \
	else \
		echo "Skipping ./cmd/ym2608_manifest_gen tests (directory missing)"; \
	fi
	$(GO_TEST_COMMON) ./internal/apu -run 'TestYM2608Stage5Slice21' -timeout 120s

# CI-focused audio gate (manifest + core YM2608 test packages)
ci-audio: check-ym2608-manifest
	$(GO_TEST_COMMON) ./internal/apu ./internal/memory ./internal/emulator -timeout 180s

# Package Nitro-Core-DX integrated app (Linux archive for Releases)
release-linux:
	bash scripts/package_release.sh
