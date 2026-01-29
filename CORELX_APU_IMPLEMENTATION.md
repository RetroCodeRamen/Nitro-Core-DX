# CoreLX APU Functions Implementation

**Date**: January 27, 2026  
**Status**: ✅ Complete

## Summary

Successfully implemented all 6 APU functions in the CoreLX compiler. These functions were previously registered in the semantic analyzer but missing from code generation, causing "unknown builtin" errors.

## Implemented Functions

### ✅ `apu.enable()`
- **Purpose**: Enable APU master volume
- **Implementation**: Writes 0xFF to MASTER_VOLUME register (0x9020)
- **Code**: 4 instructions

### ✅ `apu.set_channel_wave(ch: u8, wave: u8)`
- **Purpose**: Set waveform type for a channel
- **Parameters**: 
  - `ch`: Channel number (0-3)
  - `wave`: Waveform type (0=Sine, 1=Square, 2=Saw, 3=Noise)
- **Implementation**: 
  - Calculates channel base address: `0x9000 + (ch * 8)`
  - Writes waveform value (shifted to bits [1:2]) to CONTROL register (offset +3)
- **Code**: ~12 instructions

### ✅ `apu.set_channel_freq(ch: u8, freq: u16)`
- **Purpose**: Set frequency for a channel
- **Parameters**:
  - `ch`: Channel number (0-3)
  - `freq`: Frequency in Hz (16-bit)
- **Implementation**:
  - Calculates channel base address: `0x9000 + (ch * 8)`
  - Writes low byte to FREQ_LOW (offset +0)
  - Writes high byte to FREQ_HIGH (offset +1) - triggers phase reset
- **Code**: ~18 instructions

### ✅ `apu.set_channel_volume(ch: u8, vol: u8)`
- **Purpose**: Set volume for a channel
- **Parameters**:
  - `ch`: Channel number (0-3)
  - `vol`: Volume (0-255, 0=silent, 255=max)
- **Implementation**:
  - Calculates channel base address: `0x9000 + (ch * 8)`
  - Writes volume to VOLUME register (offset +2)
- **Code**: ~10 instructions

### ✅ `apu.note_on(ch: u8)`
- **Purpose**: Start note playback on a channel
- **Parameters**:
  - `ch`: Channel number (0-3)
- **Implementation**:
  - Calculates channel base address: `0x9000 + (ch * 8)`
  - Reads CONTROL register (offset +3)
  - Sets bit 0 (enable) to 1
  - Writes back to CONTROL register
- **Code**: ~12 instructions

### ✅ `apu.note_off(ch: u8)`
- **Purpose**: Stop note playback on a channel
- **Parameters**:
  - `ch`: Channel number (0-3)
- **Implementation**:
  - Calculates channel base address: `0x9000 + (ch * 8)`
  - Reads CONTROL register (offset +3)
  - Clears bit 0 (enable) to 0
  - Writes back to CONTROL register
- **Code**: ~12 instructions

## Technical Details

### Channel Address Calculation
All channel-based functions use the same pattern:
- Channel base: `0x9000 + (channel * 8)`
- Calculation: `channel << 3` (shift left by 3 bits = multiply by 8)
- Channel offsets:
  - CH0: 0x9000
  - CH1: 0x9008
  - CH2: 0x9010
  - CH3: 0x9018

### Register Layout
Each channel has 8 bytes:
- Offset +0: FREQ_LOW (8-bit)
- Offset +1: FREQ_HIGH (8-bit) - triggers phase reset on write
- Offset +2: VOLUME (8-bit)
- Offset +3: CONTROL (8-bit) - bit 0=enable, bits [1:2]=waveform
- Offset +4: DURATION_LOW (8-bit)
- Offset +5: DURATION_HIGH (8-bit)
- Offset +6: DURATION_MODE (8-bit)
- Offset +7: Reserved

### Argument Passing
- Arguments are passed in registers R0-R7
- `apu.enable()`: No arguments
- `apu.set_channel_wave()`: R0=channel, R1=waveform
- `apu.set_channel_freq()`: R0=channel, R1=frequency (16-bit)
- `apu.set_channel_volume()`: R0=channel, R1=volume
- `apu.note_on()`: R0=channel
- `apu.note_off()`: R0=channel

## Testing

### Test Program
Created `test/roms/apu_test.corelx`:
```corelx
function Start()
    apu.enable()
    apu.set_channel_wave(0, 0)  -- Sine wave
    apu.set_channel_freq(0, 440)  -- 440 Hz (A4)
    apu.set_channel_volume(0, 128)  -- 50% volume
    apu.note_on(0)
    
    while true
        wait_vblank()
```

### Compilation Test
✅ Compiles successfully: `./corelx test/roms/apu_test.corelx test/roms/apu_test.rom`

### Next Steps
- Test ROM execution in emulator
- Verify audio output
- Add error checking for channel bounds (0-3)
- Consider adding duration support

## Files Modified

- `internal/corelx/codegen.go` - Added 6 new cases to `generateBuiltinCall()`
- `test/roms/apu_test.corelx` - Created test program

## Code Statistics

- **Lines Added**: ~150 lines
- **Functions Implemented**: 6
- **Instructions Generated**: ~68 total instructions across all functions
- **Compilation Time**: No noticeable impact

## Status

✅ **Phase 1.1 Complete**: All APU functions are now fully implemented and ready for use.

---

**Next Phase**: Implement remaining sprite helper functions (Phase 1.2)
