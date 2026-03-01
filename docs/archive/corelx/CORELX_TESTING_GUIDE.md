# CoreLX Testing Guide

**Date**: January 27, 2026  
**Status**: ✅ Testing Framework Implemented

## Overview

The CoreLX compiler now has comprehensive testing that verifies both **compilation** (code generation) and **runtime execution** (platform behavior). Tests ensure that CoreLX programs not only compile correctly but also run correctly on the Nitro Core DX platform.

---

## Testing Framework

### Test Structure

Tests are located in `internal/corelx/corelx_test.go` and use Go's standard `testing` package.

**Two Types of Tests:**

1. **Compilation Tests** - Verify code compiles to valid ROMs
2. **Runtime Tests** - Verify code executes correctly and produces expected behavior

### Test Categories

#### 1. Compilation Tests (`TestCoreLXCompilation`)

Verifies that CoreLX source files compile successfully:

- ✅ Reads source file
- ✅ Compiles to ROM
- ✅ Verifies ROM header (magic number "RMCF")
- ✅ Checks ROM size

**Test Files:**
- `simple_test.corelx`
- `example.corelx`
- `full_example.corelx`
- `apu_test.corelx`

#### 2. APU Function Tests (`TestAPUFunctions`)

Verifies all APU functions work end-to-end:

- ✅ `apu.enable()` - Sets master volume
- ✅ `apu.set_channel_wave()` - Sets waveform type
- ✅ `apu.set_channel_freq()` - Sets frequency
- ✅ `apu.set_channel_volume()` - Sets volume
- ✅ `apu.note_on()` - Enables channel
- ✅ `apu.note_off()` - Disables channel

**Verification:**
- Checks APU register state after execution
- Verifies values match expected parameters

#### 3. Individual APU Function Tests (`TestAPUFunctionIndividual`)

Tests each APU function in isolation:

- `apu_enable` - Verifies master volume = 0xFF
- `apu_set_channel_wave` - Verifies waveform is set correctly
- `apu_set_channel_freq` - Verifies frequency is set correctly
- `apu_set_channel_volume` - Verifies volume is set correctly
- `apu_note_on` - Verifies channel is enabled
- `apu_note_off` - Verifies channel is disabled

#### 4. Sprite Function Tests (`TestSpriteFunctions`)

Verifies sprite-related functions:

- ✅ `ppu.enable_display()` - Enables PPU
- ✅ `sprite.set_pos()` - Sets sprite position
- ✅ `oam.write()` - Writes sprite to OAM
- ✅ Sprite attribute helpers

**Verification:**
- Checks PPU control register
- Verifies OAM writes occur

#### 5. VBlank Sync Tests (`TestVBlankSync`)

Verifies frame synchronization:

- ✅ `wait_vblank()` - Waits for VBlank correctly
- ✅ Frame loop executes without errors

---

## Running Tests

### Run All Tests

```bash
go test ./internal/corelx -v
```

### Run Specific Test

```bash
# Test APU functions
go test ./internal/corelx -v -run TestAPUFunctions

# Test compilation
go test ./internal/corelx -v -run TestCoreLXCompilation

# Test individual APU functions
go test ./internal/corelx -v -run TestAPUFunctionIndividual
```

### Run with Coverage

```bash
go test ./internal/corelx -cover
```

---

## Test Implementation Details

### Compilation Testing

The `CompileFile()` helper function:
1. Reads source file
2. Tokenizes source code
3. Parses tokens into AST
4. Performs semantic analysis
5. Generates machine code
6. Builds ROM file

### Runtime Testing

Runtime tests:
1. Compile CoreLX source to ROM
2. Load ROM into emulator
3. Execute instructions directly (or run frames)
4. Verify hardware state matches expectations

**Key Pattern:**
```go
// Compile
source := `function Start() ...`
CompileFile(sourcePath, outputPath)

// Load ROM
romData, _ := os.ReadFile(outputPath)
emu := emulator.NewEmulator()
emu.LoadROM(romData)

// Execute
for i := 0; i < cycles; i++ {
    emu.CPU.ExecuteInstruction()
}

// Verify
if emu.APU.MasterVolume != expected {
    t.Errorf("Master volume mismatch")
}
```

---

## Adding New Tests

### Template for Feature Tests

```go
func TestNewFeature(t *testing.T) {
    source := `function Start()
        -- Test code here
        new_feature()
        while true
            wait_vblank()
`
    
    tmpDir := t.TempDir()
    sourcePath := filepath.Join(tmpDir, "test.corelx")
    outputPath := filepath.Join(tmpDir, "test.rom")
    
    // Write source
    os.WriteFile(sourcePath, []byte(source), 0644)
    
    // Compile
    if err := CompileFile(sourcePath, outputPath); err != nil {
        t.Fatalf("Compilation failed: %v", err)
    }
    
    // Load and run
    romData, _ := os.ReadFile(outputPath)
    emu := emulator.NewEmulator()
    emu.LoadROM(romData)
    
    // Execute
    for i := 0; i < 1000; i++ {
        emu.CPU.ExecuteInstruction()
    }
    
    // Verify
    // Check expected state changes
}
```

### Testing Checklist

When adding a new feature, create tests that verify:

- [ ] **Compilation**: Code compiles without errors
- [ ] **Code Generation**: Correct instructions are generated
- [ ] **Execution**: Code runs without crashes
- [ ] **State Changes**: Hardware state changes as expected
- [ ] **Edge Cases**: Handle boundary conditions
- [ ] **Error Cases**: Invalid inputs are handled

---

## Current Test Coverage

### ✅ Fully Tested

- **APU Functions**: All 6 functions tested individually and together
- **Compilation**: Multiple test programs verified
- **VBlank Sync**: Frame synchronization verified
- **Sprite Functions**: Basic sprite operations tested

### ⚠️ Partially Tested

- **Sprite Helpers**: Some helpers not yet tested
- **Asset System**: Not yet tested (not fully implemented)
- **Struct Member Access**: Not yet tested (broken)

### ❌ Not Yet Tested

- **User-Defined Functions**: Not implemented yet
- **Complex Expressions**: Basic tests only
- **Control Flow**: Basic tests only
- **Variable Storage**: Not fully tested

---

## Test Results

### Current Status

```
✅ TestCoreLXCompilation: PASS
✅ TestAPUFunctions: PASS
✅ TestAPUFunctionIndividual: PASS (6/6 subtests)
✅ TestSpriteFunctions: PASS
✅ TestVBlankSync: PASS
```

### Coverage Goals

- **Phase 1 (Built-in Functions)**: 100% ✅
- **Phase 2 (Asset System)**: 0% (not implemented)
- **Phase 3 (Struct System)**: 0% (broken)
- **Phase 4 (Variable Storage)**: 50% (basic)
- **Phase 5 (User Functions)**: 0% (not implemented)

---

## Continuous Integration

Tests should be run:

1. **Before Committing**: Run `go test ./internal/corelx`
2. **In CI/CD**: Automatically on every push
3. **Before Release**: Full test suite + coverage report

### CI Integration Example

```yaml
# .github/workflows/test.yml
test:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v2
    - uses: actions/setup-go@v2
    - run: go test ./internal/corelx -v -cover
```

---

## Debugging Failed Tests

### Common Issues

1. **Compilation Fails**
   - Check source syntax
   - Verify all required functions exist
   - Check for semantic errors

2. **Runtime Fails**
   - Verify code actually executes (check PC changes)
   - Run more cycles if needed
   - Check hardware state before/after

3. **State Verification Fails**
   - Verify register addresses are correct
   - Check byte order (little-endian)
   - Verify register write timing

### Debugging Tools

```go
// Add logging to tests
t.Logf("PC: 0x%04X, Cycles: %d", emu.CPU.State.PCOffset, cycles)
t.Logf("APU Master Volume: 0x%02X", emu.APU.MasterVolume)
t.Logf("Channel 0: Freq=%d, Vol=%d, Enabled=%v", 
    emu.APU.Channels[0].Frequency,
    emu.APU.Channels[0].Volume,
    emu.APU.Channels[0].Enabled)
```

---

## Best Practices

1. **Isolate Tests**: Each test should be independent
2. **Clear Names**: Test names should describe what they test
3. **Verify State**: Don't just check for errors - verify expected behavior
4. **Test Edge Cases**: Test boundary conditions
5. **Keep Tests Fast**: Use direct CPU execution for unit tests
6. **Document Expected Behavior**: Comments explain what should happen

---

## Future Improvements

1. **Visual Regression Tests**: Compare rendered frames
2. **Performance Tests**: Benchmark compilation speed
3. **Stress Tests**: Long-running programs
4. **Fuzzing**: Random input testing
5. **Property-Based Tests**: Test invariants
6. **Integration Tests**: Full game scenarios

---

## Summary

The CoreLX testing framework provides:

✅ **Compilation Verification**: Ensures code compiles correctly  
✅ **Runtime Verification**: Ensures code executes correctly  
✅ **State Verification**: Ensures hardware behaves as expected  
✅ **Easy Extension**: Simple pattern for adding new tests  
✅ **CI/CD Ready**: Can be integrated into automated testing  

**Next Steps**: Add tests for remaining features as they're implemented (sprite helpers, assets, structs, etc.)
