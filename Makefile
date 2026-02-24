.PHONY: test-fast test-full test-long test-emulator test-commands

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
