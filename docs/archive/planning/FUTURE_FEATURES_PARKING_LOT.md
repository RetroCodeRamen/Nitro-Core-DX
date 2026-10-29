# Future Features Parking Lot

**Purpose:** Capture promising ideas so they are not lost while the team focuses on current milestones.

> These are intentionally not active commitments. They are future-facing concepts to revisit after core emulator correctness, performance, and CoreLX developer experience are in a stable state.

---

## Near-Mid Term Direction (After Core Stabilization)

### 1. Development Kit Focus (Beyond Emulator-Only)

Goal:
- Move Nitro-Core-DX from “emulator project” to a small but usable development kit

Includes:
- CoreLX compiler/runtime polish
- Example projects and libraries
- Better ROM/tooling workflow
- Debugger and diagnostics refinements

### 2. Emulator Validation Milestone

Before major platform expansion, validate the main subsystems:
- CPU
- Memory/Bus
- PPU
- APU (legacy + future FM extension)
- Input path

This should include repeatable test ROMs and deterministic regression tests.

---

## Future Platform / Hardware Expansion Ideas

### Keyboard Input for the Console (Future)

Concept:
- Add optional keyboard input capability for:
  - simple interpreter/scripting experiments
  - utilities / “desktop-like” apps
  - ports/tools that benefit from keyboard input

Possible implementation paths:

1. **Custom small keyboard accessory**
- Dedicated keyboard peripheral connected to an expanded controller/input bus

2. **USB HID keyboard bridge (microcontroller-based)**
- A small onboard or external microcontroller reads a standard USB HID keyboard
- Converts key events into signals/register writes the Nitro-Core-DX system understands

Benefits:
- Leverages cheap modern keyboards
- Easier user access during development
- Good fit for FPGA console bring-up and tooling workflows

Design constraints (important):
- Keep controller bus behavior deterministic
- Define a clean MMIO/input protocol for keyboard events
- Preserve compatibility with existing controller input model
- Make FPGA implementation straightforward (bridge logic can be discrete MCU or future integrated core)

Status:
- **Parking lot only (not active work)**

---

## Suggested Revisit Order

1. Emulator correctness/performance baseline
2. CoreLX usability and developer workflow
3. FM audio extension (software-first, FPGA-aligned)
4. Controller bus expansion / keyboard support
5. Desktop-like apps / interpreter experiments

