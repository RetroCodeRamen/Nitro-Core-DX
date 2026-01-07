# Nitro-Core-DX Architecture Analysis

**Status**: ✅ All synchronization mechanisms implemented (One-shot completion status, Frame counter, VBlank flag)

**FPGA Compatibility**: ✅ Architecture designed with FPGA implementation in mind - VBlank signal matches real hardware patterns

## Current Execution Model

### Frame Execution Order (Updated)
```
1. APU.UpdateFrame()        - Sets completion flags (one-shot, cleared when read)
2. CPU.ExecuteCycles(166667) - Runs CPU for exactly 166,667 cycles
3. PPU.RenderFrame()        - Renders frame, sets VBlank flag, increments FrameCounter
4. APU.GenerateSamples(735) - Generates audio samples
```

### The Problem: No Synchronization

**Issue 1: CPU Has No Frame Awareness**
- CPU runs for a fixed number of cycles (166,667) per frame
- CPU has no way to know when a frame boundary occurs
- ROM code runs in a tight loop, executing many times per frame
- No VBlank signal or frame boundary indicator

**Issue 2: Completion Status Persists Too Long**
- Completion status is set at the start of the frame (when channel finishes)
- Status persists for the **entire frame** (until next UpdateFrame clears it)
- ROM checks completion status in a tight loop
- ROM sees status set (0x01) and updates note
- ROM loops back immediately and sees status still set (0x01)
- ROM updates note again... and again... and again

**Issue 3: FrameCounter Not Incremented** ✅ FIXED
- ~~PPU has a `FrameCounter` field but it's never incremented~~ → Now increments in RenderFrame()
- ~~FrameCounter is not exposed to the ROM~~ → Now exposed at 0x803F/0x8040
- ~~ROM has no way to detect frame boundaries~~ → ROM can use frame counter or VBlank flag

**Issue 4: No State Management Between Components** ✅ FIXED
- ~~APU, CPU, and PPU operate independently~~ → Now synchronized via VBlank and frame counter
- ~~No synchronization mechanism~~ → Three mechanisms: completion status, frame counter, VBlank
- ~~No way for ROM to wait for next frame~~ → ROM can wait for VBlank or frame counter change

## Root Cause Analysis

The fundamental issue is that **the ROM is checking completion status multiple times per frame**, and since the status persists for the entire frame, it keeps triggering updates.

### Why This Happens

1. **ROM Structure**: The ROM is in a tight loop:
   ```
   main_loop:
       read completion status
       if set, update note
       jump to main_loop
   ```

2. **Execution Speed**: The CPU runs at 10 MHz, executing ~166,667 cycles per frame
   - Each loop iteration might be ~10-50 cycles
   - So the loop runs **thousands of times per frame**
   - Each time it checks completion status, it sees 0x01 (if channel finished)
   - Each time it sees 0x01, it updates the note

3. **Completion Status Behavior**: 
   - Set at start of frame (when channel finishes)
   - Persists for entire frame
   - Cleared at start of next frame
   - So ROM sees it set for the entire frame

## Proposed Solutions

### Solution 1: One-Shot Completion Status (Recommended)

**Change**: Completion status is cleared **immediately after being read**, not at the start of the next frame.

**Benefits**:
- ROM can only see completion status once per frame
- Prevents multiple updates per frame
- Simple to implement
- Matches common hardware behavior (one-shot flags)

**Implementation**:
```go
// In APU.Read8 for completion status:
if offset == 0x21 {
    status := a.ChannelCompletionStatus
    a.ChannelCompletionStatus = 0  // Clear immediately after read
    return status
}
```

**Trade-offs**:
- ROM must read status exactly once per frame
- If ROM reads it multiple times, only first read sees the flag

### Solution 2: Frame Counter Synchronization

**Change**: 
1. Increment FrameCounter in `PPU.RenderFrame()`
2. Expose FrameCounter to ROM (0x801E/0x801F)
3. ROM waits for frame counter to change before checking completion status

**Benefits**:
- ROM can detect frame boundaries
- ROM can wait for next frame before checking status
- More robust synchronization

**Implementation**:
```go
// In PPU.RenderFrame():
p.FrameCounter++  // Increment at start of frame

// In PPU.Read8():
case 0x1E: // FRAME_COUNTER_LOW
    return uint8(p.FrameCounter & 0xFF)
case 0x1F: // FRAME_COUNTER_HIGH
    return uint8(p.FrameCounter >> 8)
```

**ROM Pattern**:
```
main_loop:
    read frame counter into R3
    wait_loop:
        read frame counter into R6
        compare R6 with R3
        if equal, jump to wait_loop  // Wait for frame to change
    // Now we're in a new frame
    read completion status
    if set, update note
    jump to main_loop
```

### Solution 3: VBlank Signal (Most Hardware-Accurate)

**Change**: Add a VBlank flag that is set at the start of each frame and cleared when read.

**Benefits**:
- Matches real hardware behavior (NES, SNES, etc.)
- ROM can wait for VBlank before processing
- Standard pattern for retro game development

**Implementation**:
```go
// In PPU:
VBlankFlag bool  // Set to true at start of RenderFrame()

// In PPU.RenderFrame():
p.VBlankFlag = true  // Set at start of frame

// In PPU.Read8():
case 0x1D: // VBLANK_FLAG
    flag := p.VBlankFlag
    p.VBlankFlag = false  // Clear when read
    if flag {
        return 0x01
    }
    return 0x00
```

**ROM Pattern**:
```
main_loop:
    wait_vblank:
        read VBlank flag
        if not set, jump to wait_vblank
    // Now we're at start of frame
    read completion status
    if set, update note
    jump to main_loop
```

## Recommended Approach

**Combine All Three Solutions** (Implemented!):
1. ✅ **One-shot completion status** (clear after read) - Prevents multiple updates
2. ✅ **Frame counter** (increments once per frame, exposed to ROM) - Precise timing
3. ✅ **VBlank flag** (hardware-accurate, one-shot) - FPGA-compatible, matches real hardware

This gives ROMs maximum flexibility:
- **Simple ROMs**: Use one-shot completion status
- **Timing-sensitive ROMs**: Use frame counter for precise synchronization
- **Hardware-accurate ROMs**: Use VBlank flag (matches NES/SNES pattern)
- **FPGA-ready**: VBlank signal is hardware-accurate and easy to implement in FPGA

## Execution Order Improvements

**Current Order** (has issues):
```
1. APU.UpdateFrame() - Sets completion status
2. CPU runs - ROM checks status (sees it set)
3. CPU runs more - ROM checks again (still sees it set!)
4. PPU.RenderFrame()
```

**Better Order** (with one-shot status):
```
1. APU.UpdateFrame() - Sets completion status
2. CPU runs - ROM checks status (sees it set, status cleared)
3. CPU runs more - ROM checks again (sees 0, no update)
4. PPU.RenderFrame() - Increments frame counter
```

## State Management

**Current State**:
- APU, CPU, PPU are independent
- No shared state or synchronization
- Each component operates in isolation

**Better State Management**:
- Frame counter shared between PPU and CPU (via I/O)
- Completion status one-shot (prevents multiple reads)
- VBlank signal for frame synchronization
- Clear execution order documented

## Timing Guarantees

**Current**: No timing guarantees
- CPU runs for fixed cycles, but ROM doesn't know when frame starts/ends
- Completion status persists for entire frame
- No way to ensure ROM only processes once per frame

**With Fixes**:
- Frame counter increments once per frame (guaranteed)
- Completion status is one-shot (guaranteed to only be seen once)
- ROM can wait for frame boundary (guaranteed synchronization)

## Conclusion

The core issue was **lack of synchronization** between CPU and APU. The completion status persisted for the entire frame, causing the ROM to update multiple times per frame.

**All Three Solutions Implemented** ✅:
1. ✅ **One-shot completion status** - Cleared immediately after read, prevents multiple updates
2. ✅ **Frame counter** - Increments once per frame, exposed at 0x803F/0x8040 for precise timing
3. ✅ **VBlank flag** - Hardware-accurate signal at 0x803E, one-shot, FPGA-compatible

**Benefits of Combined Approach**:
- **Flexibility**: ROMs can choose the synchronization method that fits their needs
- **Hardware-Accuracy**: VBlank signal matches real hardware (NES, SNES pattern)
- **FPGA-Ready**: VBlank signal is easy to implement in FPGA hardware
- **Developer-Friendly**: Multiple options for different use cases
- **Prevents Bugs**: One-shot flags prevent multiple reads per frame

**Execution Order (Now Synchronized)**:
1. **APU.UpdateFrame()** - Sets completion flags
2. **PPU.RenderFrame()** - Sets VBlank flag, increments frame counter
3. **CPU execution** - ROM can check completion status, VBlank, or frame counter
4. **Audio generation** - Samples generated

This architecture is now **cohesive, synchronized, and FPGA-ready**!

