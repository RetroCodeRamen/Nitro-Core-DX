package corelx

import (
	"fmt"
	"strings"

	"nitro-core-dx/internal/rom"
)

// Part of generateBuiltinCall. See the register contract on that function.
func (cg *CodeGenerator) builtinGfx(name string, args []Expr, destReg uint8) error {
	switch name {
	case "gfx.set_palette":
		// gfx.set_palette(palette: u8, color_index: u8, color: u16)
		// Args: R0 = palette (0-15), R1 = color_index (0-15), R2 = color (RGB555, 16-bit)
		// Sets a color in CGRAM
		// CGRAM address = (palette * 16 + color_index) * 2
		// CGRAM is RGB555 format, stored as 2 bytes (low, high)

		// Calculate CGRAM color index address: (palette * 16 + color_index)
		// Note: PPU CGRAM_ADDR register is in color-index units (the PPU multiplies by 2 internally
		// when writing low/high bytes into CGRAM storage), so we must NOT multiply by 2 here.
		// palette * 16 = palette << 4
		cg.builder.AddInstruction(rom.EncodeMOV(0, 3, 0)) // MOV R3, R0 (save palette)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #4
		cg.builder.AddImmediate(4)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 3, 4)) // SHL R3, R4 -> R3 = palette << 4 = palette * 16
		cg.builder.AddInstruction(rom.EncodeADD(0, 3, 1)) // ADD R3, R1 -> R3 = palette * 16 + color_index

		// Set CGRAM_ADDR (0x8012)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0x8012
		cg.builder.AddImmediate(0x8012)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 3)) // MOV R7, R3 (CGRAM color index address)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0xFF
		cg.builder.AddImmediate(0xFF)
		cg.builder.AddInstruction(rom.EncodeAND(0, 7, 5)) // AND R7, R5 (mask to 8 bits for CGRAM_ADDR)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 6, 7)) // MOV [R6], R7 (write CGRAM_ADDR)

		// Write color to CGRAM_DATA (0x8013)
		// CGRAM_DATA requires two writes: low byte, then high byte (both to same address)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0x8013
		cg.builder.AddImmediate(0x8013)

		// Write low byte first
		cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 2)) // MOV R7, R2 (color)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0xFF
		cg.builder.AddImmediate(0xFF)
		cg.builder.AddInstruction(rom.EncodeAND(0, 7, 5)) // AND R7, R5 (mask to low byte)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 6, 7)) // MOV [R6], R7 (write low byte)

		// Write high byte (triggers CGRAM write)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 2)) // MOV R7, R2 (color)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #8
		cg.builder.AddImmediate(8)
		cg.builder.AddInstruction(rom.EncodeSHR(0, 7, 5)) // SHR R7, R5 -> R7 = color >> 8 (high byte)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 6, 7)) // MOV [R6], R7 (write high byte, triggers write)
		return nil

	case "gfx.set_palette_color":
		// gfx.set_palette_color(cgram_index: u8, color: u16)
		// Args: R0 = cgram_index (flat 0-255 CGRAM color index), R1 = color (RGB555, 16-bit)
		// Direct single-index CGRAM write — the same CGRAM_ADDR/CGRAM_DATA
		// protocol as gfx.set_palette, but the caller supplies the flat
		// palette+color-index address directly instead of two components.
		if len(args) != 2 {
			return fmt.Errorf("gfx.set_palette_color requires 2 arguments (cgram_index, color)")
		}

		// Set CGRAM_ADDR (0x8012)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0x8012
		cg.builder.AddImmediate(0x8012)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 0)) // MOV R7, R0 (cgram_index)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0xFF
		cg.builder.AddImmediate(0xFF)
		cg.builder.AddInstruction(rom.EncodeAND(0, 7, 5)) // AND R7, R5 (mask to 8 bits for CGRAM_ADDR)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 6, 7)) // MOV [R6], R7 (write CGRAM_ADDR)

		// Write color to CGRAM_DATA (0x8013): low byte, then high byte (triggers write).
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0x8013
		cg.builder.AddImmediate(0x8013)

		cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 1)) // MOV R7, R1 (color)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0xFF
		cg.builder.AddImmediate(0xFF)
		cg.builder.AddInstruction(rom.EncodeAND(0, 7, 5)) // AND R7, R5 (mask to low byte)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 6, 7)) // MOV [R6], R7 (write low byte)

		cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 1)) // MOV R7, R1 (color)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #8
		cg.builder.AddImmediate(8)
		cg.builder.AddInstruction(rom.EncodeSHR(0, 7, 5)) // SHR R7, R5 -> R7 = color >> 8 (high byte)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 6, 7)) // MOV [R6], R7 (write high byte, triggers write)
		return nil

	case "gfx.init_default_palettes":
		// gfx.init_default_palettes()
		// Initializes default palettes with basic colors
		// Palette 0: Grayscale (black to white)
		// Palette 1: Blue tones
		// Palette 2: Green tones
		// Palette 3: Red tones

		// Initialize palette 0 (grayscale)
		for i := 0; i < 16; i++ {
			// Color value: RGB555, grayscale = (i*31/15, i*31/15, i*31/15)
			// Simplified: use i*2 for each component (0-30 range)
			comp := uint16(i * 2)
			if comp > 31 {
				comp = 31
			}
			// RGB555: RRRRR GGGGG BBBBB (bits 15-11=R, 10-6=G, 5-1=B, bit 0 unused)
			color := (comp << 11) | (comp << 6) | (comp << 1)

			// Set palette 0, color i
			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012 (CGRAM_ADDR)
			cg.builder.AddImmediate(0x8012)
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #(i*2)
			cg.builder.AddImmediate(uint16(i * 2))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

			// Write color
			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013 (CGRAM_DATA)
			cg.builder.AddImmediate(0x8013)
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #color_low
			cg.builder.AddImmediate(color & 0xFF)
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (low byte)
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #color_high
			cg.builder.AddImmediate((color >> 8) & 0xFF)
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (high byte)
		}

		// Initialize palette 1 (blue tones) - simplified, just set a few colors
		// Color 0 = black, Color 15 = bright blue
		for i := 0; i < 16; i++ {
			comp := uint16(i * 2)
			if comp > 31 {
				comp = 31
			}
			// Blue: (0, 0, comp)
			color := (comp << 1) // Blue in bits 5-1

			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012
			cg.builder.AddImmediate(0x8012)
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #(16*2 + i*2)
			cg.builder.AddImmediate(uint16(16*2 + i*2))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013
			cg.builder.AddImmediate(0x8013)
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #color_low
			cg.builder.AddImmediate(color & 0xFF)
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #color_high
			cg.builder.AddImmediate((color >> 8) & 0xFF)
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
		}

		// Initialize palette 2 (green tones)
		for i := 0; i < 16; i++ {
			comp := uint16(i * 2)
			if comp > 31 {
				comp = 31
			}
			// Green: (0, comp, 0)
			color := (comp << 6) // Green in bits 10-6

			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012
			cg.builder.AddImmediate(0x8012)
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #(32*2 + i*2)
			cg.builder.AddImmediate(uint16(32*2 + i*2))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013
			cg.builder.AddImmediate(0x8013)
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #color_low
			cg.builder.AddImmediate(color & 0xFF)
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #color_high
			cg.builder.AddImmediate((color >> 8) & 0xFF)
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
		}

		// Initialize palette 3 (red tones)
		for i := 0; i < 16; i++ {
			comp := uint16(i * 2)
			if comp > 31 {
				comp = 31
			}
			// Red: (comp, 0, 0)
			color := (comp << 11) // Red in bits 15-11

			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012
			cg.builder.AddImmediate(0x8012)
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #(48*2 + i*2)
			cg.builder.AddImmediate(uint16(48*2 + i*2))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013
			cg.builder.AddImmediate(0x8013)
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #color_low
			cg.builder.AddImmediate(color & 0xFF)
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #color_high
			cg.builder.AddImmediate((color >> 8) & 0xFF)
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
		}

		return nil

	case "ppu.enable_display":
		// There is no separate master-display register in this PPU -- this
		// sets BG0_CONTROL's (0x8008) bit 0, same as bg.enable(0). Must be a
		// read-modify-write (OR in bit 0), not a blind overwrite: a blind
		// MOV [0x8008], #0x01 silently clobbered every other bit already set
		// on BG0's control byte, including bg.set_priority(0, ...)'s bits
		// [3:2] -- any program calling bg.set_priority(0, N) followed by
		// ppu.enable_display() (the normal Start() ordering) would have its
		// BG0 priority reset back to 0 right before the first frame renders.
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8008
		cg.builder.AddImmediate(0x8008)
		cg.builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (read current)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0x01
		cg.builder.AddImmediate(0x01)
		cg.builder.AddInstruction(rom.EncodeOR(0, 5, 6))  // R5 |= 0x01
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write back)
		return nil

	case "boot.show_default":
		// boot.show_default(): call the merged-in default logo slide+hold
		// sequence (see boot_splash.go's injectBootEntry, which always
		// merges __boot_show_default() + the __BootLogo asset in). Callable
		// from a user-defined __Boot() to show the stock presentation
		// before doing its own thing (e.g. reading input) and eventually
		// calling Start() itself.
		if len(args) != 0 {
			return fmt.Errorf("boot.show_default takes no arguments")
		}
		cg.emitCallPatch(bootShowDefaultFuncName)
		return nil

	case "gfx.load_tiles":
		// gfx.load_tiles(asset: u16, base: u16) -> u16
		// Args: R0 = asset ID (ASSET_* constant), R1 = base tile index
		// Loads tile data from asset to VRAM starting at base * 32 bytes
		// Returns base tile index (for chaining)

		// Check if first arg is an ASSET_ constant (compile-time known)
		if len(args) > 0 {
			if ident, ok := args[0].(*IdentExpr); ok && strings.HasPrefix(ident.Name, "ASSET_") {
				assetName := strings.TrimPrefix(ident.Name, "ASSET_")
				if asset, exists := cg.assets[assetName]; exists {
					// We know the asset at compile time - inline the data writes
					return cg.generateInlineTileLoad(asset, args[1], destReg)
				}
			}
		}

		if len(cg.program.Assets) == 0 {
			return fmt.Errorf("gfx.load_tiles requires declared assets in this source file")
		}
		return cg.generateRuntimeTileLoadDispatch(destReg)

	default:
		return fmt.Errorf("%w: %s", errUnknownBuiltin, name)
	}
}
