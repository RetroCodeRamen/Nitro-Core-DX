# WORK FOR CURSOR

Purpose:
- Track file removals or cleanup actions that may be easier to perform from your Cursor environment.

## Completed Cleanups

1. **`testrom_tools` package collisions (cmd/testrom)** — DONE
   - Extra entrypoints were relocated into dedicated command packages (one `main.go` per package).
   - New layout:
     - `cmd/testrom/input/main.go` (was `main_input.go`)
     - `cmd/testrom/minimal/main.go` (was `main_minimal.go`)
     - `cmd/testrom/cpu-execution/main.go` (was `test_cpu_execution.go`)
     - `cmd/testrom/verify-bytecode/main.go` (was `verify_bytecode.go`)
   - The four original files were removed. Default `cmd/testrom` (main.go) unchanged.

2. **Generator package layout under `test/roms`** — DONE (documented)
   - Generators remain as single-file utilities. They are **not** built as one package.
   - Documented in `test/roms/README_TEST_ROMS.md`: run with `go run -tags testrom_tools ./test/roms/<file>.go <args>`; do not run `go test -tags testrom_tools ./test/roms`.

## Notes

- I will append to this file whenever I hit a removal/cleanup step that should be handled on your side.
