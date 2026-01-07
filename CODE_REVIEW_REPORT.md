# Nitro-Core-DX Deep Code Review Report

**Review Date:** January 7, 2026  
**Reviewer:** Senior Emulator Engineer & Software Quality Auditor  
**Codebase Version:** Current as of review date  
**Review Scope:** Complete codebase including CPU, PPU, APU, Memory, Emulator Core, UI, Logging, and Documentation

---

## A) Executive Summary

### Top 5 "Ship-Stoppers" (Blocker Severity)

1. **CPU Reset() Corruption Bug** - `Reset()` sets `PCBank=0` but ROM code must be in bank 1+. If `Reset()` is called after `LoadROM()`, CPU will try to execute from I/O space, causing crashes or undefined behavior. **Location:** `internal/cpu/cpu.go:74-92`, `internal/emulator/emulator.go:202-229`

2. **Frame Execution Order Bug** - VBlank flag is set AFTER CPU execution, but documentation says CPU should see it at frame start. This breaks hardware-accurate synchronization. **Location:** `internal/emulator/emulator.go:124-151`, `internal/ppu/ppu.go:331-338`

3. **Memory Write Mode 3 Bug** - `MOV [R1], R2` (mode 3) writes only low byte to I/O addresses, but should write full 16-bit for non-I/O addresses. Current code always writes 8-bit to I/O, breaking 16-bit I/O register writes. **Location:** `internal/cpu/instructions.go:38-53`

4. **Logger Goroutine Leak** - Logger starts a goroutine that never shuts down, causing resource leaks. No `Shutdown()` call in emulator cleanup. **Location:** `internal/debug/logger.go:60-64`, `internal/ui/ui.go:641-658`

5. **No Save State Implementation** - Save states are mentioned in documentation but not implemented. Critical for debugging and testing. **Location:** Documentation references exist, but no code implementation found.

### Top 5 "Technical Debt Time Bombs"

1. **Interrupt System Not Implemented** - Interrupt handling is stubbed with TODO. This will break any ROM that relies on interrupts. **Location:** `internal/cpu/cpu.go:356-358`

2. **Division by Zero Returns 0xFFFF** - Silent failure instead of proper error handling. Could cause subtle bugs in ROMs. **Location:** `internal/cpu/instructions.go:171-178`

3. **Stack Overflow/Underflow Protection Incomplete** - Stack wrap-around logic exists but doesn't prevent corruption. `Pop16()` returns 0 on underflow without proper error. **Location:** `internal/cpu/instructions.go:480-517`

4. **Cycle Counting May Be Inaccurate** - No verification that all instructions have correct cycle counts. Some instructions may be missing cycle penalties (e.g., page crossing, branch taken). **Location:** Throughout `internal/cpu/instructions.go`

5. **APU Duration Loop Mode Broken** - Loop mode (duration mode 1) doesn't reload initial duration, so it just plays indefinitely instead of looping. **Location:** `internal/apu/apu.go:447-450`

### Top 3 "Performance Killers"

1. **PPU Renders All Pixels Every Frame** - No optimization, renders 64,000 pixels (320×200) every frame even if nothing changed. No dirty rectangle tracking, no layer culling. **Location:** `internal/ppu/ppu.go:443-570`

2. **Sprite Rendering O(n) with No Culling** - Loops through all 128 sprites every frame, rendering each pixel individually. No bounding box culling, no early exit for disabled sprites. **Location:** `internal/ppu/ppu.go:579-680`

3. **Audio Buffer Allocation in Hot Path** - Allocates new byte slice every frame for audio conversion. Should use pre-allocated buffer pool. **Location:** `internal/ui/ui.go:196-210`

### Top 5 "Documentation Mismatches or Likely Mismatches"

1. **Frame Execution Order Mismatch** - Documentation says VBlank flag is set at frame start, but code sets it after CPU execution. **Location:** `SYSTEM_MANUAL.md:83-103` vs `internal/emulator/emulator.go:124-151`

2. **CGRAM Write Behavior Unclear** - Documentation says "two 8-bit writes: low byte first, then high byte" but doesn't specify what happens with 16-bit writes. Code handles it but behavior is ambiguous. **Location:** `NITRO_CORE_DX_PROGRAMMING_MANUAL.md:328` vs `internal/ppu/ppu.go:191-211`

3. **I/O Register Write Size Ambiguity** - Documentation doesn't clearly state that I/O registers are 8-bit only. Code writes only low byte, but this isn't documented. **Location:** `NITRO_CORE_DX_PROGRAMMING_MANUAL.md:298-368`

4. **Save States Documented But Not Implemented** - `SDK_FEATURES_PLAN.md` mentions save states, but no implementation exists. **Location:** `SDK_FEATURES_PLAN.md:207-221`

5. **Matrix Mode Not Implemented** - Documentation describes Matrix Mode in detail, but code just calls `renderBackgroundLayer(0)`. **Location:** `NITRO_CORE_DX_PROGRAMMING_MANUAL.md:449-565` vs `internal/ppu/ppu.go:572-577`

---

## B) Findings Table

| Severity | Module | Category | Location | Symptom | Root Cause | Fix Summary | Doc Update Required | Docs Affected |
|----------|--------|----------|----------|---------|------------|-------------|---------------------|---------------|
| Blocker | CPU | Correctness | `cpu.go:74-92` | Reset() sets PCBank=0, crashes if called after LoadROM | Reset() doesn't preserve entry point | Reset() should reload entry point from ROM header | YES | SYSTEM_MANUAL.md |
| Blocker | Emulator Core | Timing | `emulator.go:124-151` | VBlank flag set after CPU runs, breaks sync | Wrong execution order | Move PPU.RenderFrame() before CPU execution | YES | SYSTEM_MANUAL.md, NITRO_CORE_DX_PROGRAMMING_MANUAL.md |
| Blocker | CPU | Correctness | `instructions.go:38-53` | MOV mode 3 always writes 8-bit to I/O, breaks 16-bit writes | I/O check happens before mode check | Write 16-bit to non-I/O addresses, 8-bit to I/O | YES | NITRO_CORE_DX_PROGRAMMING_MANUAL.md |
| Blocker | Logging | Resource Leak | `logger.go:60-64` | Logger goroutine never shuts down | No Shutdown() call in cleanup | Call logger.Shutdown() in UI.Cleanup() | NO | None |
| Blocker | Emulator Core | Architecture | Missing | Save states not implemented | Feature planned but not coded | Implement save/load state functions | YES | SYSTEM_MANUAL.md |
| High | CPU | Correctness | `cpu.go:356-358` | Interrupts not implemented | TODO stub | Implement interrupt handling | YES | NITRO_CORE_DX_PROGRAMMING_MANUAL.md |
| High | CPU | Correctness | `instructions.go:171-178` | Division by zero returns 0xFFFF | Silent failure | Return error or set specific flag | YES | NITRO_CORE_DX_PROGRAMMING_MANUAL.md |
| High | CPU | Correctness | `instructions.go:495-504` | Stack underflow returns 0 without error | Incomplete error handling | Return error on underflow | NO | None |
| High | PPU | Correctness | `ppu.go:572-577` | Matrix Mode not implemented | TODO stub | Implement transformation matrix | YES | NITRO_CORE_DX_PROGRAMMING_MANUAL.md |
| High | APU | Correctness | `apu.go:447-450` | Duration loop mode broken | No initial duration storage | Store initial duration for loop mode | YES | NITRO_CORE_DX_PROGRAMMING_MANUAL.md |
| Medium | CPU | Performance | `instructions.go` (all) | Cycle counts may be inaccurate | No verification | Audit all cycle counts | NO | None |
| Medium | PPU | Performance | `ppu.go:443-570` | Renders all pixels every frame | No optimization | Add dirty rectangle tracking | NO | None |
| Medium | PPU | Performance | `ppu.go:579-680` | Sprite rendering O(n) with no culling | No optimization | Add bounding box culling | NO | None |
| Medium | UI | Performance | `ui.go:196-210` | Audio buffer allocation every frame | No buffer pool | Use pre-allocated buffer | NO | None |
| Medium | Memory | Correctness | `memory.go:156-173` | CGRAM 16-bit write handling unclear | Ambiguous behavior | Clarify and document | YES | NITRO_CORE_DX_PROGRAMMING_MANUAL.md |
| Low | PPU | Architecture | `ppu.go:88-94` | Mode 8 treated as NOP | Reserved mode | Document as reserved | YES | NITRO_CORE_DX_PROGRAMMING_MANUAL.md |
| Low | CPU | Code Quality | `cpu.go:113` | Panic on SetEntryPoint failure | Should return error | Return error instead of panic | NO | None |
| Low | UI | Code Quality | `ui.go:376-402` | TODO comments for menu items | Incomplete features | Implement or remove | NO | None |

---

## C) Deep Dive by Module

### CPU Module

#### Correctness Risks

**CRITICAL: Reset() Corruption Bug**
- **Location:** `internal/cpu/cpu.go:74-92`
- **Issue:** `Reset()` sets `PCBank=0` and `PCOffset=0`, but ROM code must be in bank 1+ at offset 0x8000+. If `Reset()` is called after `LoadROM()`, the CPU will try to execute from I/O space (bank 0, offset 0x8000+), causing crashes.
- **Evidence:** 
  ```go
  func (c *CPU) Reset() {
      // ... sets PCBank = 0, PCOffset = 0
  }
  ```
- **Fix:** `Reset()` should not reset PCBank/PCOffset if ROM is loaded. Instead, `emulator.Reset()` should reload the entry point.
- **Risk:** Emulator crashes on reset, breaks save/load state functionality.

**Division by Zero Handling**
- **Location:** `internal/cpu/instructions.go:171-178`
- **Issue:** Division by zero returns `0xFFFF` silently instead of proper error handling.
- **Evidence:**
  ```go
  if value == 0 {
      c.SetRegister(reg1, 0xFFFF)
      c.UpdateFlags(0xFFFF)
      return nil
  }
  ```
- **Fix:** Set a specific flag or return an error. Consider setting a "division by zero" flag in flags register.
- **Risk:** Subtle bugs in ROMs that don't check for division by zero.

**Stack Underflow Protection**
- **Location:** `internal/cpu/instructions.go:495-504`
- **Issue:** `Pop16()` returns 0 on underflow without proper error handling.
- **Evidence:**
  ```go
  if c.State.SP < 0x0100 {
      return 0  // Silent failure
  }
  ```
- **Fix:** Return error or set a stack overflow flag.
- **Risk:** Silent stack corruption, hard-to-debug crashes.

#### Timing Risks

**Cycle Counting Accuracy**
- **Location:** Throughout `internal/cpu/instructions.go`
- **Issue:** No verification that all instructions have correct cycle counts. Some instructions may be missing cycle penalties (e.g., branch taken penalty, page crossing penalty).
- **Evidence:** Instructions add cycles manually, but no centralized cycle counting table.
- **Fix:** Create a cycle count lookup table and verify all instructions.
- **Risk:** Timing-sensitive ROMs will fail.

**Interrupt Handling Not Implemented**
- **Location:** `internal/cpu/cpu.go:356-358`
- **Issue:** Interrupt handling is stubbed with TODO.
- **Evidence:**
  ```go
  if c.State.InterruptPending != 0 && !c.GetFlag(FlagI) {
      // TODO: Handle interrupt
  }
  ```
- **Fix:** Implement interrupt vector table and interrupt handling.
- **Risk:** ROMs that rely on interrupts will fail.

#### Performance Risks

**Instruction Dispatch Overhead**
- **Location:** `internal/cpu/cpu.go:273-346`
- **Issue:** Large switch statement for instruction dispatch. Could be optimized with lookup table.
- **Fix:** Use function pointer table for faster dispatch.
- **Risk:** Minor performance impact, but acceptable for now.

#### Test Gaps

- No unit tests for instruction correctness
- No cycle count verification tests
- No flag calculation tests
- No edge case tests (division by zero, stack overflow, etc.)

#### Design Smells

- PCBank and PBR can get out of sync (code tries to sync them, but this is a code smell)
- Too many safety checks in hot paths (could be optimized with assertions in debug builds)

---

### PPU Module

#### Correctness Risks

**Matrix Mode Not Implemented**
- **Location:** `internal/ppu/ppu.go:572-577`
- **Issue:** Matrix Mode just calls `renderBackgroundLayer(0)` instead of implementing transformation.
- **Evidence:**
  ```go
  func (p *PPU) renderMatrixMode() {
      // TODO: Implement Matrix Mode transformation
      p.renderBackgroundLayer(0)
  }
  ```
- **Fix:** Implement 2x2 matrix transformation with 8.8 fixed point math.
- **Risk:** ROMs using Matrix Mode will not work correctly.

**CGRAM Write Handling Ambiguity**
- **Location:** `internal/ppu/ppu.go:191-211`
- **Issue:** CGRAM write handling uses a latch system, but behavior with 16-bit writes is unclear.
- **Evidence:** Code handles two 8-bit writes, but `Write16()` may bypass this.
- **Fix:** Document behavior clearly, ensure `Write16()` works correctly.
- **Risk:** Palette writes may fail silently.

**Sprite Priority Not Implemented**
- **Location:** `internal/ppu/ppu.go:579-680`
- **Issue:** Sprite rendering doesn't sort by priority. Code reads priority but doesn't use it.
- **Evidence:** Priority is read but not used in rendering order.
- **Fix:** Sort sprites by priority before rendering.
- **Risk:** Sprite layering will be incorrect.

#### Timing Risks

**VBlank Flag Timing**
- **Location:** `internal/ppu/ppu.go:331-338`
- **Issue:** VBlank flag is set at start of `RenderFrame()`, but `RenderFrame()` is called AFTER CPU execution. This means CPU can't see VBlank at frame start.
- **Evidence:** Execution order is: APU.UpdateFrame() → CPU.ExecuteCycles() → PPU.RenderFrame()
- **Fix:** Move VBlank flag setting to before CPU execution, or restructure frame execution.
- **Risk:** Breaks hardware-accurate synchronization.

#### Performance Risks

**No Rendering Optimization**
- **Location:** `internal/ppu/ppu.go:443-570`
- **Issue:** Renders all 64,000 pixels every frame, even if nothing changed.
- **Fix:** Add dirty rectangle tracking, layer culling, or frame skip optimization.
- **Risk:** Performance bottleneck, especially on slower systems.

**Sprite Rendering O(n) with No Culling**
- **Location:** `internal/ppu/ppu.go:579-680`
- **Issue:** Loops through all 128 sprites, rendering each pixel individually. No bounding box culling.
- **Fix:** Add bounding box culling, early exit for disabled sprites, sprite sorting optimization.
- **Risk:** Performance bottleneck with many sprites.

#### Test Gaps

- No rendering correctness tests
- No palette conversion tests
- No sprite priority tests
- No Matrix Mode transformation tests

#### Design Smells

- Large `renderBackgroundLayer()` function (570 lines) - should be split
- Duplicate code between sprite and background rendering
- Hard-coded tilemap base address (0x4000)

---

### APU Module

#### Correctness Risks

**Duration Loop Mode Broken**
- **Location:** `internal/apu/apu.go:447-450`
- **Issue:** Loop mode doesn't reload initial duration, so it just plays indefinitely.
- **Evidence:**
  ```go
  if ch.DurationMode == 1 {
      // Loop mode: restart the note (reload duration from... wait, we need to store initial duration)
      // TODO: Store initial duration for proper looping
  }
  ```
- **Fix:** Store initial duration when channel is enabled, reload on loop.
- **Risk:** Looping notes won't work correctly.

**Phase Reset Logic**
- **Location:** `internal/apu/apu.go:212-235`
- **Issue:** Phase is reset only when frequency changes, but documentation says it should reset on frequency high byte write.
- **Evidence:** Code checks if frequency actually changed before resetting phase.
- **Fix:** Verify behavior matches documentation, or update documentation.
- **Risk:** Audio artifacts if behavior doesn't match expectations.

#### Timing Risks

**Audio Buffer Management**
- **Location:** `internal/ui/ui.go:186-220`
- **Issue:** Audio queue size check may cause dropouts if queue is full.
- **Evidence:** Code skips queuing if queue is too full, which can cause audio gaps.
- **Fix:** Implement better audio buffer management, or increase buffer size.
- **Risk:** Audio stuttering or dropouts.

#### Performance Risks

**Audio Sample Generation**
- **Location:** `internal/apu/apu.go:334-422`
- **Issue:** Generates samples one at a time in a loop. Could be optimized with SIMD or batch processing.
- **Fix:** Batch process samples, or use SIMD for waveform generation.
- **Risk:** Minor performance impact, but acceptable for now.

#### Test Gaps

- No audio correctness tests
- No frequency calculation tests
- No phase accumulator tests
- No duration countdown tests

#### Design Smells

- Complex frequency update logic with pending state
- Debug logging code mixed with production code

---

### Emulator Core Module

#### Correctness Risks

**Frame Execution Order Bug**
- **Location:** `internal/emulator/emulator.go:124-151`
- **Issue:** Execution order is: APU.UpdateFrame() → CPU.ExecuteCycles() → PPU.RenderFrame(). But VBlank flag is set in PPU.RenderFrame(), so CPU can't see it at frame start.
- **Evidence:**
  ```go
  e.APU.UpdateFrame()
  e.CPU.ExecuteCycles(targetCycles)
  e.PPU.RenderFrame()  // VBlank flag set here
  ```
- **Fix:** Restructure to set VBlank flag before CPU execution, or move PPU.RenderFrame() before CPU.
- **Risk:** Breaks hardware-accurate synchronization.

**Reset() Doesn't Reload Entry Point**
- **Location:** `internal/emulator/emulator.go:202-229`
- **Issue:** `Reset()` calls `CPU.Reset()` which sets PCBank=0, but doesn't reload entry point from ROM.
- **Evidence:** Code tries to reload entry point, but CPU.Reset() has already corrupted PCBank.
- **Fix:** Reload entry point BEFORE calling CPU.Reset(), or fix CPU.Reset() to not reset PC.
- **Risk:** Emulator crashes on reset.

#### Timing Risks

**Frame Timing Accuracy**
- **Location:** `internal/emulator/emulator.go:166-175`
- **Issue:** Frame limiting uses `time.Sleep()` which may not be accurate enough for 60 FPS.
- **Fix:** Use more accurate timing (e.g., `time.Sleep()` with correction, or vsync).
- **Risk:** Frame rate may not be exactly 60 FPS.

#### Performance Risks

**No Save State Implementation**
- **Location:** Missing
- **Issue:** Save states are mentioned in documentation but not implemented.
- **Fix:** Implement save/load state functions for CPU, PPU, APU, Memory.
- **Risk:** No way to debug or test specific game states.

#### Test Gaps

- No frame timing tests
- No save/load state tests
- No reset behavior tests

#### Design Smells

- Frame execution order is hard-coded, should be configurable
- No way to step through instructions one at a time (for debugging)

---

### UI Module

#### Correctness Risks

**Logger Goroutine Leak**
- **Location:** `internal/ui/ui.go:641-658`
- **Issue:** Logger starts a goroutine in `NewLogger()`, but `UI.Cleanup()` doesn't call `logger.Shutdown()`.
- **Evidence:** Logger has `Shutdown()` method, but it's never called.
- **Fix:** Call `logger.Shutdown()` in `UI.Cleanup()`.
- **Risk:** Resource leak, goroutine never exits.

**Audio Device Cleanup**
- **Location:** `internal/ui/ui.go:645-647`
- **Issue:** Audio device may not be closed on all error paths.
- **Fix:** Ensure audio device is closed in all cleanup paths.
- **Risk:** Audio device leak.

#### Performance Risks

**Audio Buffer Allocation**
- **Location:** `internal/ui/ui.go:196-210`
- **Issue:** Allocates new byte slice every frame for audio conversion.
- **Fix:** Use pre-allocated buffer pool.
- **Risk:** GC pressure, performance impact.

#### Test Gaps

- No UI rendering tests
- No input handling tests
- No audio queue tests

#### Design Smells

- Large `Run()` function with many responsibilities
- TODO comments for incomplete features

---

### Logging Module

#### Correctness Risks

**Goroutine Leak**
- **Location:** `internal/debug/logger.go:60-64`
- **Issue:** Logger starts a goroutine that never shuts down.
- **Evidence:** `processLogs()` goroutine runs forever, no shutdown call.
- **Fix:** Call `Shutdown()` in cleanup code.
- **Risk:** Resource leak.

**Log Entry Dropping**
- **Location:** `internal/debug/logger.go:134-139`
- **Issue:** Log entries are dropped if channel is full (non-blocking send).
- **Fix:** Increase buffer size or use blocking send with timeout.
- **Risk:** Important logs may be lost.

#### Performance Risks

**Channel Overhead**
- **Location:** `internal/debug/logger.go:27-31`
- **Issue:** Logger uses goroutine + channel for thread-safe logging, which adds overhead.
- **Fix:** Consider lock-free logging for hot paths.
- **Risk:** Performance impact when logging is enabled.

#### Test Gaps

- No logging correctness tests
- No thread-safety tests
- No buffer overflow tests

---

## D) Documentation Review

### Confirmed Mismatches

#### 1. Frame Execution Order Mismatch

**Documentation Says:**
- `SYSTEM_MANUAL.md:83-103` states execution order is:
  1. APU.UpdateFrame()
  2. PPU.RenderFrame() (sets VBlank flag)
  3. CPU.ExecuteCycles()
  4. APU.GenerateSamples()

**Code Does:**
- `internal/emulator/emulator.go:124-151` shows:
  1. APU.UpdateFrame()
  2. CPU.ExecuteCycles()
  3. PPU.RenderFrame() (sets VBlank flag)
  4. APU.GenerateSamples()

**Fix Required:**
```markdown
### Frame Execution Order (Synchronized)

```
Frame Start:
  1. APU.UpdateFrame()
     - Decrements channel durations
     - Sets completion flags (if channels finished)
  
  2. PPU.RenderFrame()
     - Sets VBlank flag = true (at START of frame, before CPU runs)
     - Increments FrameCounter
     - Renders frame
  
  3. CPU.ExecuteCycles(166667)
     - ROM can read:
       * VBlank flag (0x803E) - will see 1, then cleared
       * Frame counter (0x803F/0x8040) - current frame number
       * Completion status (0x9021) - will see flags, then cleared
  
  4. APU.GenerateSamples(735)
     - Generate audio for this frame
```
```

**OR** Update code to match documentation (move PPU.RenderFrame() before CPU execution).

#### 2. I/O Register Write Size Ambiguity

**Documentation Says:**
- `NITRO_CORE_DX_PROGRAMMING_MANUAL.md:298-368` lists I/O registers but doesn't clearly state they are 8-bit only.

**Code Does:**
- `internal/cpu/instructions.go:38-53` writes only low byte to I/O addresses.

**Fix Required:**
Add to `NITRO_CORE_DX_PROGRAMMING_MANUAL.md` section "Memory Map" → "I/O Register Map":

```markdown
**Important:** All I/O registers are 8-bit only. When writing 16-bit values to I/O addresses, only the low byte is written. The high byte is ignored. This applies to all PPU, APU, and Input registers.
```

#### 3. Matrix Mode Not Implemented

**Documentation Says:**
- `NITRO_CORE_DX_PROGRAMMING_MANUAL.md:449-565` describes Matrix Mode in detail with examples.

**Code Does:**
- `internal/ppu/ppu.go:572-577` just calls `renderBackgroundLayer(0)`.

**Fix Required:**
Add warning to `NITRO_CORE_DX_PROGRAMMING_MANUAL.md` section "PPU (Graphics System)" → "Matrix Mode":

```markdown
> **⚠️ IMPLEMENTATION STATUS**: Matrix Mode is currently not fully implemented. The transformation matrix registers are available, but the rendering pipeline does not yet apply the transformation. This feature is planned for a future release.
```

#### 4. Save States Documented But Not Implemented

**Documentation Says:**
- `SDK_FEATURES_PLAN.md:207-221` describes save state features.

**Code Does:**
- No save state implementation found.

**Fix Required:**
Add note to `SDK_FEATURES_PLAN.md`:

```markdown
> **⚠️ STATUS**: Save states are planned but not yet implemented. This feature will be added in a future release.
```

### Ambiguities

1. **CGRAM Write Behavior** - Documentation says "two 8-bit writes: low byte first, then high byte" but doesn't specify what happens with 16-bit writes. Code handles it, but behavior should be documented.

2. **Stack Overflow Behavior** - Documentation doesn't specify what happens on stack overflow. Code wraps around, but this should be documented.

3. **Division by Zero** - Documentation doesn't specify behavior. Code returns 0xFFFF, but this should be documented.

### Missing Spec Areas

1. **Interrupt System** - No documentation for interrupt handling (because it's not implemented).

2. **Error Handling** - No documentation for error conditions (division by zero, stack overflow, invalid addresses).

3. **Cycle Counting** - No documentation for instruction cycle counts.

4. **Determinism** - No documentation about determinism guarantees or non-deterministic operations.

---

## E) Recommended Plan

### Week 1: Critical Fixes (Must-Fix Before Release)

1. **Fix CPU Reset() Bug** (Day 1)
   - Modify `CPU.Reset()` to not reset PCBank/PCOffset
   - Fix `emulator.Reset()` to reload entry point correctly
   - Test: Reset emulator after loading ROM, verify PC is correct

2. **Fix Frame Execution Order** (Day 1-2)
   - Move `PPU.RenderFrame()` before `CPU.ExecuteCycles()`
   - OR update documentation to match code
   - Test: Verify VBlank flag is visible to CPU at frame start

3. **Fix MOV Mode 3 I/O Write Bug** (Day 2)
   - Fix `executeMOV()` to write 16-bit to non-I/O addresses
   - Test: Write 16-bit value to WRAM, verify both bytes written

4. **Fix Logger Goroutine Leak** (Day 2)
   - Call `logger.Shutdown()` in `UI.Cleanup()`
   - Test: Run emulator, close window, verify no goroutine leaks

5. **Document Critical Mismatches** (Day 3)
   - Update documentation for frame execution order
   - Document I/O register write behavior
   - Add warnings for unimplemented features

### Week 2: High Priority Fixes

1. **Implement Save States** (Day 1-3)
   - Add `SaveState()` and `LoadState()` methods to Emulator
   - Serialize CPU, PPU, APU, Memory state
   - Test: Save state, modify state, load state, verify correctness

2. **Fix Division by Zero** (Day 4)
   - Add "division by zero" flag or return error
   - Update documentation
   - Test: Divide by zero, verify flag is set

3. **Fix Stack Underflow** (Day 4)
   - Return error on stack underflow
   - Test: Pop from empty stack, verify error

4. **Fix APU Duration Loop Mode** (Day 5)
   - Store initial duration when channel enabled
   - Reload on loop
   - Test: Enable channel with loop mode, verify it loops

### Week 3: Medium Priority Fixes

1. **Implement Matrix Mode** (Day 1-3)
   - Implement 2x2 matrix transformation
   - Test: Enable Matrix Mode, verify transformation works

2. **Fix Sprite Priority** (Day 4)
   - Sort sprites by priority before rendering
   - Test: Multiple sprites with different priorities

3. **Performance Optimizations** (Day 5)
   - Add audio buffer pool
   - Add sprite culling
   - Test: Performance benchmarks

### "Must-Fix Before Release" Checklist

- [ ] CPU Reset() bug fixed
- [ ] Frame execution order fixed or documented
- [ ] MOV mode 3 I/O write bug fixed
- [ ] Logger goroutine leak fixed
- [ ] Critical documentation mismatches updated
- [ ] Save states implemented (or removed from docs)
- [ ] Division by zero handling documented
- [ ] Stack underflow handling fixed
- [ ] APU duration loop mode fixed
- [ ] Matrix Mode implemented or documented as not implemented

### "Nice-to-Have After Release" List

- [ ] Interrupt system implementation
- [ ] Performance optimizations (dirty rectangles, sprite culling)
- [ ] Cycle count verification and optimization
- [ ] Comprehensive test suite
- [ ] Debugger interface
- [ ] Input replay system

### Documentation Update Plan

**Priority 1 (Week 1):**
1. Frame execution order section in SYSTEM_MANUAL.md
2. I/O register write behavior in NITRO_CORE_DX_PROGRAMMING_MANUAL.md
3. Matrix Mode status warning in NITRO_CORE_DX_PROGRAMMING_MANUAL.md
4. Save state status in SDK_FEATURES_PLAN.md

**Priority 2 (Week 2):**
1. Division by zero behavior in NITRO_CORE_DX_PROGRAMMING_MANUAL.md
2. Stack overflow behavior in NITRO_CORE_DX_PROGRAMMING_MANUAL.md
3. Error handling section in NITRO_CORE_DX_PROGRAMMING_MANUAL.md

**Priority 3 (Week 3):**
1. Cycle counting documentation (if implemented)
2. Determinism guarantees documentation
3. Interrupt system documentation (if implemented)

---

## F) Test & Tooling Recommendations

### Specific Test Types

1. **Instruction Correctness Tests**
   - Test each instruction with known inputs/outputs
   - Test flag calculations
   - Test edge cases (overflow, underflow, division by zero)

2. **Cycle Count Verification Tests**
   - Verify all instructions have correct cycle counts
   - Test branch taken/not taken penalties
   - Test page crossing penalties (if applicable)

3. **Memory Map Tests**
   - Test all memory regions (WRAM, ROM, I/O, Extended WRAM)
   - Test memory mirroring (if any)
   - Test I/O register read/write behavior

4. **Frame Timing Tests**
   - Verify exactly 166,667 cycles per frame
   - Verify VBlank flag timing
   - Verify frame counter increments correctly

5. **Save/Load State Tests**
   - Save state, modify state, load state, verify correctness
   - Test with different ROM states
   - Test state file format compatibility

6. **Audio Tests**
   - Test frequency calculation
   - Test phase accumulator
   - Test duration countdown
   - Test completion status

7. **PPU Rendering Tests**
   - Test palette conversion (RGB555 → RGB888)
   - Test sprite rendering
   - Test background layer rendering
   - Test Matrix Mode transformation (when implemented)

8. **Determinism Tests**
   - Run same ROM with same inputs, verify identical output
   - Test with different frame rates
   - Test with different audio sample rates

9. **Input Tests**
   - Test button state latching
   - Test controller 1 and 2 independently
   - Test all button combinations

10. **Error Handling Tests**
    - Test division by zero
    - Test stack overflow/underflow
    - Test invalid memory accesses
    - Test invalid instruction opcodes

### Concrete Tests to Add

1. **TestCPU_Reset_PreservesEntryPoint**
   - Load ROM, call Reset(), verify PC is correct

2. **TestCPU_DivisionByZero**
   - Divide by zero, verify flag is set or error is returned

3. **TestCPU_StackUnderflow**
   - Pop from empty stack, verify error

4. **TestPPU_VBlankTiming**
   - Verify VBlank flag is set before CPU execution

5. **TestAPU_DurationLoopMode**
   - Enable channel with loop mode, verify it loops correctly

6. **TestMemory_IOWrite16Bit**
   - Write 16-bit value to I/O register, verify only low byte written

7. **TestEmulator_SaveLoadState**
   - Save state, modify state, load state, verify correctness

8. **TestPPU_MatrixMode**
   - Enable Matrix Mode, set transformation, verify transformation applied

9. **TestCPU_CycleCounts**
   - Verify all instructions have correct cycle counts

10. **TestDeterminism**
    - Run same ROM twice, verify identical output

### Static Analysis / Sanitizers / Profiling Tools

1. **Go Race Detector**
   - Run with `go test -race` to detect race conditions
   - Especially important for logger goroutine

2. **Go Memory Profiler**
   - Use `go tool pprof` to find memory leaks
   - Check for goroutine leaks

3. **Go CPU Profiler**
   - Use `go tool pprof` to find performance bottlenecks
   - Focus on PPU rendering and audio generation

4. **Static Analysis**
   - Use `golangci-lint` or `staticcheck` for code quality
   - Check for unused code, dead code, etc.

5. **Coverage Analysis**
   - Use `go test -cover` to measure test coverage
   - Aim for >80% coverage on critical paths

### Metrics/Logs to Add for Debugging

1. **Frame Timing Metrics**
   - Frame time (ms)
   - CPU cycles per frame
   - PPU render time
   - Audio generation time

2. **Audio Metrics**
   - Audio queue size
   - Audio underflow/overflow count
   - Sample rate drift

3. **Memory Metrics**
   - Memory allocation rate
   - GC pause time
   - Heap size

4. **Performance Metrics**
   - Instructions per second
   - Pixels rendered per second
   - Audio samples generated per second

### Determinism Testing + Input Replay

1. **Input Recording**
   - Record all input events with timestamps
   - Save to file for replay

2. **Input Replay**
   - Load recorded input, replay with same timing
   - Verify identical output

3. **Determinism Verification**
   - Run same ROM with same inputs multiple times
   - Compare output frames (pixel-by-pixel)
   - Compare audio samples (sample-by-sample)

4. **State Comparison**
   - Save state at specific points
   - Compare states from different runs
   - Verify identical state

---

## Conclusion

This codebase shows good structure and organization, but has several critical correctness bugs that must be fixed before release. The most critical issues are:

1. CPU Reset() corruption bug
2. Frame execution order mismatch
3. MOV mode 3 I/O write bug
4. Logger goroutine leak
5. Missing save state implementation

The codebase also has significant technical debt in the form of unimplemented features (interrupts, Matrix Mode) and performance issues (no rendering optimization, no sprite culling).

**Recommendation:** Focus on the Week 1 critical fixes first, then address high-priority issues in Week 2. The codebase is close to being release-ready, but these fixes are essential for stability and correctness.

---

**End of Code Review Report**

