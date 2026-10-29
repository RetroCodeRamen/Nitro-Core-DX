package corelx

import (
	"fmt"

	"nitro-core-dx/internal/rom"
)

// Part of generateBuiltinCall. See the register contract on that function.
func (cg *CodeGenerator) builtinSprite(name string, args []Expr, destReg uint8) error {
	switch name {
	case "sprite.set_pos":
		// sprite.set_pos(s: *Sprite, x: i16, y: u8)
		// Args: R0 = sprite pointer, R1 = x (i16), R2 = y (u8)
		// Store x and y to sprite struct
		// R0 has sprite address (from &hero), R1 has x, R2 has y
		// Write x_lo (low byte of x)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 3, 0)) // MOV R3, R0 (save sprite addr)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 4, 1)) // MOV R4, R1 (save x, kept pristine)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 5, 2)) // MOV R5, R2 (save y)

		// Write x_lo (offset 0) - low byte of x, into scratch R6 (not R4)
		// so R4 still holds the original, unmasked x below.
		cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 4)) // MOV R6, R4 (x copy)
		cg.hAndImm(6, 0xFF)
		cg.builder.AddInstruction(rom.EncodeMOV(7, 3, 6)) // MOV [R3], R6 (8-bit store x_lo)

		// Write x_hi (offset 1): sign bit (bit 0), preserving whatever
		// size-code bits (bits [3:1]) sprite.set_size already put there --
		// set_pos has no size/ctrl information of its own and must not
		// clobber them, so this is a read-modify-write, not a blind
		// overwrite. Previously this read the ALREADY-masked-to-low-byte
		// R4 (from the x_lo step above) rather than the original x, so
		// shifting right by 8 always gave 0 -- every sprite moved via
		// set_pos silently got x_hi=0 regardless of sign. Harmless only
		// because every existing caller uses small positive X (the hero's
		// fixed on-screen position).
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #1
		cg.builder.AddImmediate(1)
		cg.builder.AddInstruction(rom.EncodeADD(0, 3, 6)) // ADD R3, R6 (offset 1: x_hi)
		cg.builder.AddInstruction(rom.EncodeMOV(6, 6, 3)) // MOV R6, [R3] (existing x_hi, 8-bit load)
		cg.hAndImm(6, 0xFE)                               // clear old sign bit, keep size-code bits
		cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 4)) // MOV R7, R4 (pristine x)
		cg.hShrImm(7, 8)                                  // R7 = x >> 8
		cg.hAndImm(7, 1)                                  // R7 = sign bit only
		cg.builder.AddInstruction(rom.EncodeOR(0, 6, 7))  // OR R6, R7 -> merged x_hi
		cg.builder.AddInstruction(rom.EncodeMOV(7, 3, 6)) // MOV [R3], R6 (8-bit store x_hi)

		// Write y (offset 2)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #1
		cg.builder.AddImmediate(1)
		cg.builder.AddInstruction(rom.EncodeADD(0, 3, 6)) // ADD R3, R6 (increment to offset 2)
		cg.builder.AddInstruction(rom.EncodeMOV(7, 3, 5)) // MOV [R3], R5 (write y)
		return nil

	case "sprite.set_size":
		// sprite.set_size(s: *Sprite, size_flags: u16)
		// Args: R0 = sprite pointer, R1 = a SPR_SIZE_* value (the same
		// constants used for oam.write_sprite_data's ctrl argument, or a
		// Sprite's .ctrl field -- see emitSizeCodeBits).
		// Sets the sprite's 3-bit size code in the struct's x_hi byte
		// (offset 1), preserving whatever sign bit sprite.set_pos has
		// already written there (or will write later -- the two can be
		// called in either order, each preserves the other's bits via
		// read-modify-write). Needed because a Sprite struct's size lives
		// in x_hi, not ctrl (ctrl's bits are already fully packed with
		// enable/blend mode/alpha), and unlike oam.write_sprite_data
		// (which gets ctrl and x together in one call), a struct's x/y and
		// ctrl are typically set by separate statements at different
		// times, so there's no single call site to fold the size code into
		// automatically.
		if len(args) != 2 {
			return fmt.Errorf("sprite.set_size requires 2 arguments")
		}
		cg.builder.AddInstruction(rom.EncodeMOV(0, 3, 0)) // MOV R3, R0 (sprite addr)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #1
		cg.builder.AddImmediate(1)
		cg.builder.AddInstruction(rom.EncodeADD(0, 3, 4)) // ADD R3, R4 (offset 1: x_hi)
		cg.builder.AddInstruction(rom.EncodeMOV(6, 5, 3)) // MOV R5, [R3] (existing x_hi, 8-bit load)
		cg.hAndImm(5, 0x01)                               // keep only the sign bit
		cg.emitSizeCodeBits(5, 1, 4)
		cg.builder.AddInstruction(rom.EncodeMOV(7, 3, 5)) // MOV [R3], R5 (8-bit store x_hi)
		return nil

	case "oam.write":
		// oam.write(id: u8, s: *Sprite)
		// Args: R0 = sprite id, R1 = sprite pointer
		// Set OAM_ADDR to the raw sprite id (0-127); the PPU internally
		// multiplies by 6 to get the byte offset -- writing id*6 here would
		// double-multiply and corrupt every id > 0.

		// Save sprite pointer (R1) to R3
		cg.builder.AddInstruction(rom.EncodeMOV(0, 3, 1)) // MOV R3, R1 (sprite pointer)

		// Save id (R0) to R6
		cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 0)) // MOV R6, R0 (save id)

		// Set OAM_ADDR (0x8014) to raw id
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8014
		cg.builder.AddImmediate(0x8014)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 5, 6)) // MOV R5, R6 (sprite id)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write OAM_ADDR)

		// Set OAM_DATA address (0x8015)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8015
		cg.builder.AddImmediate(0x8015)

		// Read sprite struct and write to OAM_DATA
		// Sprite format: x_lo (offset 0), x_hi (offset 1), y (offset 2), tile (offset 3), attr (offset 4), ctrl (offset 5)
		// R3 = sprite pointer (save original in R7 for later use if needed)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 3)) // MOV R7, R3 (save original sprite pointer)

		// Write x_lo (offset 0)
		cg.builder.AddInstruction(rom.EncodeMOV(6, 5, 3)) // MOV R5, [R3] (8-bit load, mode 6)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write to OAM_DATA)

		// Write x_hi (offset 1)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #1
		cg.builder.AddImmediate(1)
		cg.builder.AddInstruction(rom.EncodeADD(0, 3, 6)) // ADD R3, R6 (increment to offset 1)
		cg.builder.AddInstruction(rom.EncodeMOV(6, 5, 3)) // MOV R5, [R3]
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

		// Write y (offset 2)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #1
		cg.builder.AddImmediate(1)
		cg.builder.AddInstruction(rom.EncodeADD(0, 3, 6)) // ADD R3, R6 (increment to offset 2)
		cg.builder.AddInstruction(rom.EncodeMOV(6, 5, 3)) // MOV R5, [R3]
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

		// Write tile (offset 3)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #1
		cg.builder.AddImmediate(1)
		cg.builder.AddInstruction(rom.EncodeADD(0, 3, 6)) // ADD R3, R6 (increment to offset 3)
		cg.builder.AddInstruction(rom.EncodeMOV(6, 5, 3)) // MOV R5, [R3]
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

		// Write attr (offset 4)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #1
		cg.builder.AddImmediate(1)
		cg.builder.AddInstruction(rom.EncodeADD(0, 3, 6)) // ADD R3, R6 (increment to offset 4)
		cg.builder.AddInstruction(rom.EncodeMOV(6, 5, 3)) // MOV R5, [R3]
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

		// Write ctrl (offset 5)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #1
		cg.builder.AddImmediate(1)
		cg.builder.AddInstruction(rom.EncodeADD(0, 3, 6)) // ADD R3, R6 (increment to offset 5)
		cg.builder.AddInstruction(rom.EncodeMOV(6, 5, 3)) // MOV R5, [R3]
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

		return nil

	case "oam.write_sprite_data":
		// oam.write_sprite_data(id: u8, x: i16, y: u8, tile: u8, attr: u8, ctrl: u8)
		// Args: R0=id, R1=x, R2=y, R3=tile, R4=attr, R5=ctrl
		// Preserve y in R6 and keep attr/ctrl in R4/R5 by using R0/R2/R7 as temporaries.
		if len(args) != 6 {
			return fmt.Errorf("oam.write_sprite_data requires 6 arguments")
		}

		idReg := uint8(0)
		xReg := uint8(1)
		yReg := uint8(2)
		tileReg := uint8(3)
		attrReg := uint8(4)
		ctrlReg := uint8(5)
		// Save y (R2) because R2 will be reused for x temp values
		cg.builder.AddInstruction(rom.EncodeMOV(0, 6, yReg)) // MOV R6, R2 (save y)

		// Set OAM_ADDR to sprite ID (0-127), NOT id * 6
		// The PPU internally multiplies by 6 to get the byte offset
		// Write sprite ID directly to OAM_ADDR (0x8014)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 2, 0)) // MOV R2, #0x8014
		cg.builder.AddImmediate(0x8014)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 2, idReg)) // MOV [R2], R0 (sprite ID)

		// Write sprite data to OAM_DATA (0x8015) using R0 as pointer (id no longer needed)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 0, 0)) // MOV R0, #0x8015
		cg.builder.AddImmediate(0x8015)

		// X low byte (R1)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 2, xReg)) // MOV R2, R1 (x temp)
		cg.hAndImm(2, 0xFF)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 0, 2)) // MOV [R0], R2

		// X high byte: sign bit (bit 0) | sprite size code (bits [3:1],
		// derived from ctrl -- see emitSizeCodeBits). x (R1) is no longer
		// needed after this point (X-low already consumed it above), so
		// it's safe to reuse as scratch here. This also fixes a latent
		// bug: the old code did a bare SHR with no masking, so for
		// negative x the *entire* upper byte (not just the sign bit) ended
		// up set via two's-complement sign extension -- harmless before
		// (the PPU only ever consulted bit 0), but would have silently
		// corrupted every negative-X sprite's new size code.
		cg.hShrImm(xReg, 8)
		cg.hAndImm(xReg, 1)
		cg.emitSizeCodeBits(xReg, ctrlReg, 2)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 0, xReg)) // MOV [R0], R1 (x_hi)

		// Y (from saved R6)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 0, 6)) // MOV [R0], R6

		// Tile (R3)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 0, tileReg)) // MOV [R0], R3

		// Attr (R4)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 0, attrReg)) // MOV [R0], R4

		// Ctrl (R5)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 0, ctrlReg)) // MOV [R0], R5
		return nil

	case "oam.clear_sprite":
		// oam.clear_sprite(id: u8)
		// Args: R0 = sprite id
		// Disables sprite by setting control byte to 0
		if len(args) != 1 {
			return fmt.Errorf("oam.clear_sprite requires 1 argument")
		}

		idReg := uint8(0)

		// Set OAM_ADDR to sprite ID, then write 0 to control byte (byte 5)
		// Write sprite ID to OAM_ADDR (0x8014)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8014
		cg.builder.AddImmediate(0x8014)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 6, idReg)) // MOV R6, R0 (sprite ID)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 6))     // MOV [R4], R6 (write sprite ID to OAM_ADDR)

		// Set OAM_ADDR to sprite ID again, but this time we need to write to byte 5 (control)
		// The PPU uses OAM_ADDR * 6 + byte_index, so we need to write 5 dummy bytes first
		// Actually, simpler: just write 0 to all 6 bytes to completely disable the sprite
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8015
		cg.builder.AddImmediate(0x8015)
		// Write 0 to all 6 bytes (X_low, X_high, Y, Tile, Attr, Ctrl)
		for i := 0; i < 6; i++ {
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 6)) // MOV [R4], R6
		}
		return nil

	case "SPR_PAL":
		// SPR_PAL(p: u8) -> u8
		// Returns palette value (p & 0x0F)
		if len(args) != 1 {
			return fmt.Errorf("SPR_PAL requires 1 argument")
		}
		// Arg is in R0, mask to 4 bits
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0x0F
		cg.builder.AddImmediate(0x0F)
		cg.builder.AddInstruction(rom.EncodeAND(0, destReg, 7)) // AND R{destReg}, R7
		return nil

	case "SPR_PRI":
		// SPR_PRI(p: u8) -> u8
		// Returns priority value shifted to bits [7:6] of attr byte
		// Priority is in bits [7:6] of byte 4 (Attributes)
		// Shift priority value left by 6 bits: p << 6
		if len(args) != 1 {
			return fmt.Errorf("SPR_PRI requires 1 argument")
		}
		// Arg is in R0, shift left by 6 bits
		cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 0)) // MOV R{destReg}, R0
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))       // MOV R7, #6
		cg.builder.AddImmediate(6)
		cg.builder.AddInstruction(rom.EncodeSHL(0, destReg, 7)) // SHL R{destReg}, R7 -> priority << 6
		return nil

	case "SPR_HFLIP":
		// SPR_HFLIP() -> u8
		// Returns 0x10 (horizontal flip bit, bit 4 of attr byte)
		cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
		cg.builder.AddImmediate(0x10)
		return nil

	case "SPR_VFLIP":
		// SPR_VFLIP() -> u8
		// Returns 0x20 (vertical flip bit, bit 5 of attr byte)
		cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
		cg.builder.AddImmediate(0x20)
		return nil

	case "SPR_SIZE_8":
		// SPR_SIZE_8() -> u8
		// Returns 0x00 (8×8 size, bit 1 of ctrl byte = 0)
		cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
		cg.builder.AddImmediate(0x00)
		return nil

	case "SPR_ENABLE":
		// SPR_ENABLE() -> u8
		// Returns 0x01 (enable bit, bit 0 of ctrl byte)
		cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
		cg.builder.AddImmediate(0x01)
		return nil

	case "SPR_SIZE_16":
		// SPR_SIZE_16() -> u8
		// Returns 0x02 (16×16 size bit, bit 1 of ctrl byte = 1)
		cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
		cg.builder.AddImmediate(0x02)
		return nil

	// SPR_SIZE_32X16 through SPR_SIZE_128X128: the larger sprite sizes
	// (see spriteSizeTable, internal/ppu/scanline.go). These return a
	// wider ctrl-shaped value than SPR_SIZE_8/16 -- bit 1 plus new bits
	// 8/9 -- which oam.write_sprite_data and sprite.set_size split apart
	// via emitSizeCodeBits: bit 1 still goes to the control byte
	// (harmlessly, it's the now-vestigial legacy size bit) while bits
	// 8/9 exist only to be extracted into the X-high byte's size-code
	// field. Never write one of these values into a Sprite's .ctrl field
	// directly and expect it alone to work -- .ctrl is a plain 8-bit
	// struct field, so bits 8/9 would simply be truncated away; always go
	// through sprite.set_size (for Sprite structs) or
	// oam.write_sprite_data's ctrl argument (which does the extraction
	// inline) instead.
	case "SPR_SIZE_32X16":
		cg.hMovImm(destReg, 0x0100)
		return nil
	case "SPR_SIZE_32X32":
		cg.hMovImm(destReg, 0x0102)
		return nil
	case "SPR_SIZE_64X32":
		cg.hMovImm(destReg, 0x0200)
		return nil
	case "SPR_SIZE_64X64":
		cg.hMovImm(destReg, 0x0202)
		return nil
	case "SPR_SIZE_128X64":
		cg.hMovImm(destReg, 0x0300)
		return nil
	case "SPR_SIZE_128X128":
		cg.hMovImm(destReg, 0x0302)
		return nil

	case "SPR_BLEND":
		// SPR_BLEND(mode: u8) -> u8
		// Returns blend mode shifted to bits [3:2] of ctrl byte
		// Blend mode is in bits [3:2] of byte 5 (Control)
		// Shift mode left by 2 bits: mode << 2
		if len(args) != 1 {
			return fmt.Errorf("SPR_BLEND requires 1 argument")
		}
		// Arg is in R0, shift left by 2 bits
		cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 0)) // MOV R{destReg}, R0
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))       // MOV R7, #2
		cg.builder.AddImmediate(2)
		cg.builder.AddInstruction(rom.EncodeSHL(0, destReg, 7)) // SHL R{destReg}, R7 -> mode << 2
		return nil

	case "SPR_ALPHA":
		// SPR_ALPHA(a: u8) -> u8
		// Returns alpha value shifted to bits [7:4] of ctrl byte
		// Alpha is in bits [7:4] of byte 5 (Control)
		// Shift alpha left by 4 bits: a << 4
		if len(args) != 1 {
			return fmt.Errorf("SPR_ALPHA requires 1 argument")
		}
		// Arg is in R0, mask to 4 bits first (alpha is 0-15), then shift left by 4
		cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 0)) // MOV R{destReg}, R0
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))       // MOV R7, #0x0F
		cg.builder.AddImmediate(0x0F)
		cg.builder.AddInstruction(rom.EncodeAND(0, destReg, 7)) // AND R{destReg}, R7 -> mask to 4 bits
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))       // MOV R7, #4
		cg.builder.AddImmediate(4)
		cg.builder.AddInstruction(rom.EncodeSHL(0, destReg, 7)) // SHL R{destReg}, R7 -> alpha << 4
		return nil

	case "oam.flush":
		// oam.flush() - no-op for now
		return nil

	default:
		return fmt.Errorf("%w: %s", errUnknownBuiltin, name)
	}
}
