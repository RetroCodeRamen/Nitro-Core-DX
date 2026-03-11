# YM2608 Compact Stream Format

Purpose:
- replace ROMs that bake YM2608 writes into executable code
- store song data as compact assets
- support multiple songs per cartridge without multi-megabyte code bloat

Status:
- format and encoder implemented
- ROM playback driver not yet integrated

## Container

Header:
- bytes `0x00-0x07`: ASCII magic `NCDXMUS1`
- bytes `0x08-0x0B`: `frame_samples` (`uint32`, little-endian)
- bytes `0x0C-0x0F`: reserved, currently zero

Payload:
- command stream beginning at byte `0x10`

## Commands

- `0x00`
  - `END`
  - terminate stream

- `0x10 <wait_u8>`
  - wait `1..255` frames

- `0x11 <wait_lo> <wait_hi>`
  - wait `1..65535` frames

- `0x20 <addr> <data>`
  - write YM2608 port 0 register

- `0x21 <addr> <data>`
  - write YM2608 port 1 register

- `0x30 <start_addr> <count> <data...>`
  - burst write to YM2608 port 0
  - writes `count` bytes to consecutive addresses starting at `start_addr`

- `0x31 <start_addr> <count> <data...>`
  - burst write to YM2608 port 1
  - writes `count` bytes to consecutive addresses starting at `start_addr`

## Timing Model

- stream timing is frame-based
- `frame_samples` defines one logical music frame
  - `735` for 60 Hz
  - `882` for 50 Hz
- writes happen before the following wait command expires

## Current Encoder Behavior

- VGM waits are collapsed into frame counts
- consecutive empty frames become compact wait commands
- consecutive same-port writes to incrementing register addresses are emitted as burst commands when profitable

## Why This Replaces the Current Pong Music Path

Current Pong ROM path:
- parses VGM
- expands every YM write into CPU instructions
- stores playback as executable code

Compact stream path:
- parses VGM
- stores YM actions as data
- uses one reusable playback routine

Expected result:
- songs measured in tens of KB instead of multiple MB
- multiple songs per ROM become practical
- easier programming-manual story:
  - song asset format
  - playback API
  - loop behavior

