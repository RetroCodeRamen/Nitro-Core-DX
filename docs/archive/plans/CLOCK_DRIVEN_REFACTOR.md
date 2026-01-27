# Clock-Driven Refactoring Status

## Overview

This document tracks the progress of refactoring Nitro-Core-DX from a frame-driven system to a clock-driven system with accurate cycle-based coordination.

## Completed Work

### 1. Master Clock Scheduler ✅
- **File**: `internal/clock/scheduler.go`
- **Status**: Implemented
- **Features**:
  - Master clock that coordinates CPU, PPU, and APU
  - Cycle-based stepping for all components
  - Configurable clock speeds for each component
  - Component step functions registered with scheduler

### 2. Memory System Split ✅
- **Files**: 
  - `internal/memory/bus.go` - Memory bus (routing)
  - `internal/memory/cartridge.go` - ROM cartridge (data)
- **Status**: Implemented
- **Features**:
  - `Bus` handles memory routing (WRAM, Extended WRAM, I/O)
  - `Cartridge` handles ROM data and entry point lookup
  - Clean separation of concerns

### 3. PPU Scanline/Dot Stepping ✅
- **File**: `internal/ppu/scanline.go`
- **Status**: Implemented
- **Features**:
  - Scanline-by-scanline rendering
  - Dot-by-dot (cycle-by-cycle) stepping
  - Proper VBlank timing
  - HDMA support per scanline
  - Frame counter updates at correct timing

### 4. APU Fixed-Point Audio ✅
- **File**: `internal/apu/fixed_point.go`
- **Status**: Implemented
- **Features**:
  - Fixed-point phase accumulator (32-bit)
  - Fixed-point sample generation
  - Float conversion only at host adapter interface
  - All internal calculations use integer arithmetic

## In Progress

### 5. CPU Cycle Accuracy ⚠️
- **Status**: Partially complete
- **Current State**: CPU instructions have cycle costs, but they may need refinement
- **Needs**: 
  - Review and verify cycle costs per instruction type
  - Ensure cycle costs account for fetch, decode, execute phases
  - Add cycle costs for memory access patterns

## Completed Work (Continued)

### 6. Emulator Integration ✅
- **File**: `internal/emulator/emulator.go` (replaced old implementation)
- **Status**: Complete - Old emulator removed, clock-driven is now the only implementation
- **Features**:
  - Clock-driven `Emulator` struct using clock scheduler
  - Uses `Bus` + `Cartridge` instead of `MemorySystem`
  - Clock-driven `RunFrame()` method
  - Proper audio sample generation during clock stepping
  - Fixed-point audio with float conversion at host adapter
  - Maintains same API for compatibility with UI code
  - **FPGA-Ready**: Cycle-accurate design suitable for FPGA implementation

### 7. Migration Complete ✅
- **Status**: Complete
- **Changes**:
  - Old frame-driven `emulator.go` replaced with clock-driven version
  - All references updated (tests, UI code)
  - `MemorySystem` replaced with `Bus` + `Cartridge`
  - Save state system updated to use Bus
  - All tests passing

## Pending Work

### 7. Test Updates ⏳
- **Status**: Not started
- **Needs**:
  - Update all tests to work with clock-driven system
  - Test clock scheduler coordination
  - Test scanline/dot stepping accuracy
  - Test fixed-point audio generation
  - Verify cycle-accurate timing

## Migration Path

### Step 1: Update Emulator Structure
1. Replace `MemorySystem` with `Bus` + `Cartridge`
2. Add `MasterClock` to emulator
3. Register component step functions with clock

### Step 2: Replace RunFrame()
1. Create `StepCycles()` method that uses clock scheduler
2. Update frame timing to be cycle-based
3. Maintain FPS tracking using cycle counts

### Step 3: Update Components
1. CPU: Ensure cycle costs are accurate
2. PPU: Use `StepPPU()` instead of `RenderFrame()`
3. APU: Use `GenerateSampleFixed()` instead of `GenerateSample()`

### Step 4: Update Tests
1. Update emulator tests
2. Update component tests
3. Add clock scheduler tests

## Technical Details

### Clock Speeds
- **CPU**: 10 MHz (10,000,000 cycles/sec)
- **PPU**: 10 MHz (same as CPU, dot-by-dot)
- **APU**: 44,100 Hz (sample rate)

### PPU Timing
- **Screen**: 320×200 pixels
- **Scanlines**: 220 total (200 visible + 20 VBlank)
- **Dots per scanline**: 360 (320 visible + 40 HBlank)
- **Cycles per frame**: 79,200 (220 scanlines × 360 dots)

### APU Fixed-Point
- **Phase**: 32-bit unsigned integer (0-2^32 represents 0-2π)
- **Phase Increment**: `(frequency * 2^32) / sampleRate`
- **Sample Output**: 16-bit signed integer (-32768 to 32767)
- **Host Conversion**: `float32(sample) / 32768.0`

## Notes

- The old frame-based `RenderFrame()` is still available for compatibility
- Fixed-point APU has both old (float) and new (fixed) implementations
- Memory system split maintains backward compatibility through `MemorySystem` wrapper (if needed)

## Next Steps

1. **Complete CPU cycle accuracy review** ⚠️
   - Review and refine cycle costs per instruction type
   - Ensure cycle costs account for fetch, decode, execute phases
   - Add cycle costs for memory access patterns

2. **FPGA Preparation** ⏳
   - Document clock domain boundaries
   - Identify synchronous vs asynchronous components
   - Plan for FPGA-specific optimizations
   - Consider pipeline stages for CPU

3. **Performance Optimization** ⏳
   - Profile clock-driven execution
   - Optimize hot paths in clock scheduler
   - Consider batch processing for PPU/APU where appropriate

## Usage

### Using the Clock-Driven Emulator

```go
// Create clock-driven emulator (now the default)
emu := emulator.NewEmulator()

// Load ROM
if err := emu.LoadROM(romData); err != nil {
    log.Fatal(err)
}

// Start emulator
emu.Start()

// Run frames (clock-driven, cycle-accurate)
for {
    if err := emu.RunFrame(); err != nil {
        log.Fatal(err)
    }
    
    // Get output
    pixels := emu.GetOutputBuffer()
    audio := emu.GetAudioSamples()
    
    // Render/play...
}
```

**Note**: The emulator is now clock-driven by default. The old frame-driven implementation has been removed.

## Architecture Comparison

### Old (Frame-Driven)
- `RunFrame()` executes:
  1. APU.UpdateFrame()
  2. PPU.RenderFrame() (entire frame at once)
  3. CPU.ExecuteCycles(166667)
  4. APU.GenerateSamples(735)

### New (Clock-Driven)
- `RunFrame()` executes:
  1. Clock.StepCycles(166667) which coordinates:
     - CPU.StepCPU() every cycle
     - PPU.StepPPU() every cycle (dot-by-dot)
     - APU.StepAPU() every ~227 cycles (sample-by-sample)
  2. Audio samples collected during stepping
  3. Frame timing handled by clock scheduler

## Benefits

1. **Cycle-Accurate**: All components run at their correct clock speeds
2. **Hardware-Accurate**: PPU renders scanline-by-scanline, dot-by-dot
3. **Better Synchronization**: Components can interact at cycle boundaries
4. **Fixed-Point Audio**: No floating-point in audio generation (better performance)
5. **Extensible**: Easy to add new components to clock scheduler
6. **Testable**: Can step by individual cycles for precise testing
