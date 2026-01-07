# FPGA Compatibility Notes

## Design Philosophy

The Nitro-Core-DX architecture is designed with **FPGA implementation in mind**. All synchronization signals and timing mechanisms are hardware-accurate and can be directly translated to FPGA logic.

## Hardware-Accurate Signals

### VBlank Flag (0x803E)
- **Behavior**: One-shot flag set at start of frame, cleared when read
- **FPGA Implementation**: Simple D flip-flop with read-clear logic
- **Hardware Pattern**: Matches NES, SNES, and other retro consoles
- **Use Case**: Primary frame synchronization signal

**FPGA Logic**:
```verilog
// VBlank flag generation
always @(posedge clk) begin
    if (frame_start) begin
        vblank_flag <= 1'b1;
    end else if (read_vblank) begin
        vblank_flag <= 1'b0;  // Clear when read
    end
end
```

### Frame Counter (0x803F/0x8040)
- **Behavior**: 16-bit counter increments once per frame
- **FPGA Implementation**: Simple 16-bit counter with frame_start increment
- **Use Case**: Precise timing, frame-perfect synchronization

**FPGA Logic**:
```verilog
// Frame counter
reg [15:0] frame_counter;
always @(posedge clk) begin
    if (frame_start) begin
        frame_counter <= frame_counter + 1;
    end
end
```

### Completion Status (0x9021)
- **Behavior**: One-shot flag, cleared immediately after read
- **FPGA Implementation**: Register with read-clear logic
- **Use Case**: Audio channel completion detection

**FPGA Logic**:
```verilog
// Completion status (one-shot)
reg [3:0] completion_status;
always @(posedge clk) begin
    if (channel_finished) begin
        completion_status[channel] <= 1'b1;
    end else if (read_completion) begin
        completion_status <= 4'b0000;  // Clear when read
    end
end
```

## Execution Order (FPGA-Friendly)

The emulator's execution order is designed to match hardware behavior:

1. **APU Update** (start of frame)
   - Decrement durations
   - Set completion flags
   - Clear completion status (if not already cleared)

2. **CPU Execution** (during frame)
   - CPU runs for fixed cycles (166,667 @ 10 MHz)
   - Can read completion status, VBlank, frame counter
   - All signals are stable during CPU execution

3. **PPU Rendering** (end of frame)
   - Render frame
   - Set VBlank flag
   - Increment frame counter

4. **Audio Generation** (continuous)
   - Generate samples at 44.1 kHz
   - Independent of frame timing

## FPGA Implementation Benefits

### Synchronization Signals
- **VBlank**: Hardware-accurate, matches real console behavior
- **Frame Counter**: Simple counter, easy to implement
- **Completion Status**: One-shot logic, prevents race conditions

### Timing Guarantees
- All signals are set/cleared at well-defined points
- No race conditions between CPU and APU/PPU
- Frame boundaries are clearly defined

### Register Layout
- All I/O registers are byte-addressable
- Simple address decoding (0x8000-0x8FFF = PPU, 0x9000-0x9FFF = APU)
- No complex state machines required

## Recommended FPGA Architecture

### Clock Domains
- **CPU Clock**: 10 MHz (main system clock)
- **Audio Clock**: 44.1 kHz (audio sample generation)
- **Video Clock**: 60 Hz (frame timing, VBlank)

### Synchronization
- VBlank signal synchronized to video clock
- Frame counter synchronized to video clock
- Completion status synchronized to CPU clock
- Cross-clock domain signals use proper synchronization

### State Machines
- **APU**: Simple state machine for channel control
- **PPU**: Rendering pipeline with clear stages
- **CPU**: Instruction fetch/decode/execute pipeline

## Migration Path

When migrating to FPGA:

1. **Keep Register Layout**: All I/O addresses remain the same
2. **Keep Signal Behavior**: VBlank, frame counter, completion status work identically
3. **Replace Emulation Logic**: Replace Go emulation code with Verilog/VHDL
4. **Add Hardware Peripherals**: Real audio DAC, video DAC, etc.

## Testing Strategy

For FPGA verification:
1. **Test ROMs**: Use same test ROMs as emulator
2. **Signal Verification**: Verify VBlank, frame counter, completion status match emulator
3. **Timing Verification**: Ensure frame boundaries align correctly
4. **Audio Verification**: Verify audio samples match emulator output

## Conclusion

The architecture is **FPGA-ready** with:
- ✅ Hardware-accurate synchronization signals
- ✅ Clear execution order
- ✅ Simple register layout
- ✅ No complex state dependencies
- ✅ Well-defined timing guarantees

This makes the emulator a perfect **reference implementation** for FPGA development!

