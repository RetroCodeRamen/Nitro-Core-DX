# Test ROM Documentation

## Overview

The test ROM (`test.rom`) is a simple demonstration ROM that tests the core functionality of the Nitro-Core-DX emulator:

1. **Movable Block**: A block that can be moved with arrow keys
2. **Color Changes**: Button A changes block color, Button B changes background color
3. **Audio Scale**: Plays a C major scale, one note at a time, with 1 second of sound followed by 1 second of silence

## Building the Test ROM

### Prerequisites

- Go 1.21 or later installed
- The Nitro-Core-DX project built

### Build Steps

1. **Build the test ROM generator:**
   ```bash
   go build -o testrom ./cmd/testrom
   ```

2. **Generate the test ROM:**
   ```bash
   ./testrom test.rom
   ```

   Or use the build script:
   ```bash
   ./build_test.sh
   ```

3. **Build the emulator:**
   ```bash
   go build -o nitro-core-dx ./cmd/emulator
   ```

4. **Run the test ROM:**
   ```bash
   ./nitro-core-dx -rom test.rom
   ```

## Test ROM Features

### Controls

- **Arrow Keys (UP/DOWN/LEFT/RIGHT)**: Move the block
- **A Button**: Change block color (cycles through palettes 0-15)
- **B Button**: Change background color (cycles through palettes 0-15)

### Audio

The ROM plays a C major scale:
- C4 (262 Hz)
- D4 (294 Hz)
- E4 (330 Hz)
- F4 (349 Hz)
- G4 (392 Hz)
- A4 (440 Hz)
- B4 (494 Hz)
- C5 (523 Hz)

Each note plays for 1 second, followed by 1 second of silence, then the next note. The scale loops continuously.

### Visual Elements

- **Block**: A simple 8×8 pixel block (sprite) that can be moved
- **Background**: A solid color background that can be changed
- **Colors**: Uses CGRAM palettes for colors

## Implementation Notes

### ROM Structure

The test ROM is built using the ROM builder tool (`cmd/testrom/main.go`), which:
1. Encodes instructions programmatically
2. Calculates branch offsets correctly
3. Generates a valid ROM header
4. Writes the ROM file in the correct format

### Instruction Encoding

The ROM uses the Nitro-Core-DX instruction set:
- `MOV` for data movement and memory access
- `ADD`, `SUB` for arithmetic
- `AND`, `CMP` for logic and comparison
- `BNE`, `JMP` for control flow
- `SHL` for bit manipulation

### Memory Layout

- **Entry Point**: Bank 1, Offset 0x8000 (standard ROM entry point)
- **Registers Used**:
  - R0: Block X position
  - R1: Block Y position
  - R2: Block color palette index
  - R3: Background color palette index
  - R4: Note index (0-7)
  - R5: Frame timer
  - R6, R7: Temporary registers

### I/O Usage

- **PPU Registers** (0x8000-0x8FFF):
  - BG0 scroll and control
  - VRAM, CGRAM, OAM access
  - Sprite positioning

- **APU Registers** (0x9000-0x9FFF):
  - Channel 0 frequency, volume, control
  - Scale note playback

- **Input Registers** (0xA000-0xAFFF):
  - Controller 1 button state reading
  - Latch mechanism

## Testing the Emulator

### Expected Behavior

1. **ROM Loading**: The emulator should successfully load the ROM and start execution
2. **CPU Execution**: The CPU should execute instructions correctly
3. **Input Handling**: Button presses should be detected (when input is connected)
4. **Audio Generation**: The APU should generate audio samples (when audio output is connected)
5. **PPU Rendering**: The PPU should render frames (when display is connected)

### Current Limitations

- **No UI**: The emulator is currently headless (no graphical display)
- **No Audio Output**: Audio samples are generated but not played
- **No Input**: Keyboard/controller input is not yet connected
- **Basic Rendering**: PPU rendering is a placeholder (needs full implementation)

### Future Enhancements

Once the UI is implemented, you should be able to:
- See the block moving on screen
- See color changes in real-time
- Hear the scale playing
- Use keyboard/controller to control the block

## Troubleshooting

### ROM Won't Load

- Check that the ROM file exists and is readable
- Verify the ROM header is valid (magic number "RMCF")
- Check ROM version compatibility

### Emulator Crashes

- Check for CPU errors (invalid instructions, division by zero)
- Verify memory access is within bounds
- Check I/O register access is correct

### No Output

- The emulator is currently headless - no visual output yet
- Audio samples are generated but not played
- Check console output for errors

## Next Steps

1. **Implement UI**: Add SDL2 or similar for graphical display
2. **Connect Input**: Map keyboard/controller to input system
3. **Audio Output**: Connect APU to audio device
4. **Complete PPU**: Implement full tile and sprite rendering
5. **Enhance Test ROM**: Add more test cases and features



