# Bouncing Ball ROM

A test ROM for the clock-driven Nitro-Core-DX architecture.

## What It Tests

This ROM demonstrates and tests:

1. **Clock-Driven CPU Execution**: Ball movement calculations run cycle-accurately
2. **PPU Sprite Rendering**: 16x16 white ball sprite bounces around the screen
3. **APU Sound Effects**: Plays a bounce sound when the ball hits walls
4. **VBlank Synchronization**: Waits for VBlank before updating each frame
5. **VRAM Initialization**: Sets up tile data for the sprite

## Building the ROM

```bash
cd test/roms
go run build_bouncing_ball_sprite.go bouncing_ball.rom
```

## Running the ROM

```bash
cd ../..
./nitro-core-dx -rom test/roms/bouncing_ball.rom
```

Or with the emulator binary:

```bash
./cmd/emulator/nitro-core-dx -rom test/roms/bouncing_ball.rom
```

## Expected Behavior

- A white 16x16 square (ball) bounces around a dark blue screen
- The ball moves at 2 pixels per frame in both X and Y directions
- When the ball hits a wall (X < 0, X >= 304, Y < 0, Y >= 184), it:
  - Reverses velocity
  - Plays a bounce sound (440 Hz square wave)
- The ball continuously bounces around the screen

## Technical Details

- **Ball Size**: 16x16 pixels
- **Screen Size**: 320x200 pixels
- **Initial Position**: (160, 100) - center of screen
- **Initial Velocity**: (2, 2) pixels per frame
- **Bounce Sound**: 440 Hz square wave, 192 volume
- **Sprite**: Uses sprite 0, tile 0, palette 1
- **Background**: Dark blue (RGB555: 0x001F)

## Architecture Testing

This ROM specifically tests the clock-driven architecture:

- **CPU**: Executes movement calculations, boundary checks, and velocity updates
- **PPU**: Renders sprite using scanline/dot stepping (clock-driven)
- **APU**: Generates sound using fixed-point audio (clock-driven)
- **Synchronization**: Uses VBlank flag for frame synchronization

## Notes

- The ball is currently a simple filled square (all pixels = color 1)
- A more sophisticated version could use a circular tile pattern
- The bounce sound plays every time the ball hits a wall
- The ROM demonstrates proper use of the clock-driven execution model
