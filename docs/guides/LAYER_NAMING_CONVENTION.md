# Layer Naming Convention

**Purpose**: Provide a clear, consistent naming convention for the different layers of the Nitro-Core-DX system to facilitate efficient troubleshooting and communication.

---

## Layer Hierarchy

The Nitro-Core-DX system consists of four distinct layers, from highest to lowest level:

### 1. **Interface Layer** (IF)
**Also known as**: UI Layer, Host Layer

**Description**: 
- The user-facing interface and input/output handling
- Translates host system input (keyboard, mouse) into console input
- Renders emulator output to the display
- Handles window management, scaling, and user controls

**Components**:
- Fyne UI (`internal/ui/fyne_ui.go`)
- Input translation (keyboard → controller buttons)
- Display rendering (framebuffer → window)
- Window controls (pause, reset, fullscreen)

**When to reference**: 
- Input not registering (keyboard → controller translation)
- Display issues (scaling, window management)
- UI responsiveness problems
- Visual artifacts that might be rendering issues

**Example issues**:
- "IF: Arrow keys not translating to controller input"
- "IF: Display scaling incorrect"
- "IF: Window not responding to clicks"

---

### 2. **Hardware Spec Layer** (HW)
**Also known as**: Design Layer, Specification Layer

**Description**:
- The system design and architecture documentation
- Defines how the hardware should behave
- Register maps, memory layouts, timing specifications
- Expected behavior and constraints

**Components**:
- `docs/specifications/COMPLETE_HARDWARE_SPECIFICATION_V2.1.md`
- `docs/specifications/HARDWARE_SPECIFICATION.md`
- Register definitions and memory maps
- Timing specifications (CPU speed, frame rate, etc.)

**When to reference**:
- Verifying expected behavior against design
- Understanding register layouts and memory maps
- Checking timing requirements
- Resolving ambiguities in implementation

**Example issues**:
- "HW: According to spec, register 0x8014 should auto-increment"
- "HW: Spec says VBlank occurs at scanline 200"
- "HW: Need to verify CGRAM address calculation matches spec"

---

### 3. **Emulator Layer** (EMU)
**Also known as**: Implementation Layer, Core Layer

**Description**:
- The software implementation of the hardware specification
- CPU, PPU, APU, Memory, Input system implementations
- Cycle-accurate timing and synchronization
- Hardware behavior simulation

**Components**:
- CPU (`internal/cpu/`)
- PPU (`internal/ppu/`)
- APU (`internal/apu/`)
- Memory (`internal/memory/`)
- Input (`internal/input/`)
- Clock/Scheduler (`internal/clock/`)
- Emulator orchestration (`internal/emulator/`)

**When to reference**:
- Implementation bugs (incorrect behavior)
- Timing issues
- Component interaction problems
- Performance issues
- Logic errors in hardware simulation

**Example issues**:
- "EMU: CPU instruction decoding incorrect"
- "EMU: PPU sprite rendering bug"
- "EMU: Input latch not working correctly"
- "EMU: Scheduler timing drift"

---

### 4. **ROM/Cartridge Layer** (ROM)
**Also known as**: Software Layer, Game Layer, Code Layer

**Description**:
- The code/data running on the emulated system
- ROM files and ROM builders
- Game logic and initialization code
- Test ROMs and demos

**Components**:
- ROM files (`test/roms/*.rom`)
- ROM builders (`test/roms/build_*.go`)
- Game code and data
- Test programs

**When to reference**:
- ROM code bugs (incorrect instructions)
- ROM initialization issues
- ROM builder errors
- Game-specific problems
- Test ROM correctness

**Example issues**:
- "ROM: CGRAM address calculation incorrect in ROM builder"
- "ROM: Missing VBlank wait before OAM write"
- "ROM: Incorrect instruction encoding"
- "ROM: Test ROM not initializing correctly"

---

## Usage Guidelines

### When Troubleshooting

1. **Identify the layer** where the issue likely originates
2. **Use the layer prefix** when describing the issue:
   - `[IF]` for Interface Layer issues
   - `[HW]` for Hardware Spec Layer issues
   - `[EMU]` for Emulator Layer issues
   - `[ROM]` for ROM/Cartridge Layer issues

3. **Work from highest to lowest layer**:
   - Start with Interface Layer (most visible)
   - Move down to Emulator Layer (implementation)
   - Check ROM Layer (code running on system)
   - Reference Hardware Spec Layer (design verification)

### Example Troubleshooting Flow

```
Issue: Sprite not moving with arrow keys

1. [IF] Check: Are keyboard events being received?
   → Verify keyboard → controller translation

2. [EMU] Check: Is input system reading controller correctly?
   → Verify latch mechanism, button state reading

3. [ROM] Check: Is ROM reading input correctly?
   → Verify ROM code reads controller and updates sprite

4. [HW] Check: Does behavior match specification?
   → Verify input register layout matches spec
```

### Communication Examples

**Good**:
- "I think this is an [EMU] issue - the input latch isn't working correctly"
- "The [ROM] code looks correct, so this might be an [EMU] problem"
- "According to [HW] spec, this register should auto-increment"
- "[IF] is translating keyboard input correctly, so the issue is likely [EMU] or [ROM]"

**Less Clear**:
- "The input isn't working" (which layer?)
- "Something's wrong with the controller" (interface? emulator? ROM?)
- "The code is broken" (ROM code? Emulator code?)

---

## Layer Interaction Diagram

```
┌─────────────────────────────────────────┐
│         Interface Layer (IF)            │
│  - Keyboard Input                        │
│  - Display Output                        │
│  - Window Management                     │
└──────────────┬──────────────────────────┘
               │
               │ Translates input/output
               │
┌──────────────▼──────────────────────────┐
│      Emulator Layer (EMU)                │
│  - CPU, PPU, APU                        │
│  - Memory System                        │
│  - Input System                         │
│  - Clock/Scheduler                      │
└──────────────┬──────────────────────────┘
               │
               │ Implements
               │
┌──────────────▼──────────────────────────┐
│    Hardware Spec Layer (HW)             │
│  - System Design                        │
│  - Register Maps                        │
│  - Timing Specs                         │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│      ROM/Cartridge Layer (ROM)           │
│  - Game Code                            │
│  - ROM Builders                         │
│  - Test Programs                         │
└──────────────┬──────────────────────────┘
               │
               │ Runs on
               │
┌──────────────▼──────────────────────────┐
│      Emulator Layer (EMU)                │
│  (ROM executes here)                     │
└─────────────────────────────────────────┘
```

---

## Quick Reference

| Layer | Prefix | Primary Concern | Key Files |
|-------|--------|----------------|-----------|
| Interface | `[IF]` | User interaction, display | `internal/ui/` |
| Hardware Spec | `[HW]` | Design documentation | `docs/specifications/` |
| Emulator | `[EMU]` | Hardware implementation | `internal/cpu/`, `internal/ppu/`, etc. |
| ROM/Cartridge | `[ROM]` | Code running on system | `test/roms/` |

---

## Notes

- **Layer boundaries are not always clear**: Some issues span multiple layers
- **Start high, work down**: Interface issues are most visible, so check there first
- **Spec is the truth**: Hardware Spec Layer defines expected behavior
- **ROM is the test**: ROM Layer tests the Emulator Layer implementation

---

**Last Updated**: 2025-02-11
**Version**: 1.0
