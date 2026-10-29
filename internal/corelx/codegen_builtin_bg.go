package corelx

import (
	"fmt"
	"strings"

	"nitro-core-dx/internal/rom"
)

// Part of generateBuiltinCall. See the register contract on that function.
func (cg *CodeGenerator) builtinBG(name string, args []Expr, destReg uint8) error {
	switch name {
	case "bg.set_scroll":
		// bg.set_scroll(layer: u8, scroll_x: i16, scroll_y: i16)
		// Args: R0 = layer (0-3), R1 = scroll_x, R2 = scroll_y
		// BG scroll register layout:
		//   BG0: 0x8000/01 (X), 0x8002/03 (Y)
		//   BG1: 0x8004/05 (X), 0x8006/07 (Y)
		//   BG2: 0x800A/0B (X), 0x800C/0D (Y)
		//   BG3: 0x8022/23 (X), 0x8024/25 (Y)
		// For simplicity, generate a lookup for each layer.
		if len(args) != 3 {
			return fmt.Errorf("bg.set_scroll requires 3 arguments (layer, x, y)")
		}
		// Save args
		cg.builder.AddInstruction(rom.EncodeMOV(0, 3, 0)) // R3 = layer
		cg.builder.AddInstruction(rom.EncodeMOV(0, 4, 1)) // R4 = scroll_x
		cg.builder.AddInstruction(rom.EncodeMOV(0, 5, 2)) // R5 = scroll_y

		// Helper: write scroll_x low and high bytes, then scroll_y low and high bytes.
		// We'll generate inline code for each layer check.
		bgScrollAddrs := [][2]uint16{
			{0x8000, 0x8002}, // BG0 X/Y base
			{0x8004, 0x8006}, // BG1 X/Y base
			{0x800A, 0x800C}, // BG2 X/Y base
			{0x8022, 0x8024}, // BG3 X/Y base
		}
		jumpToEnd := make([]int, 0, 4)
		for i, addrs := range bgScrollAddrs {
			// if R3 != i, skip
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(uint16(i))
			cg.builder.AddInstruction(rom.EncodeCMP(0, 3, 6))
			cg.builder.AddInstruction(rom.EncodeBNE())
			skipPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)

			// Write scroll_x low byte
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(addrs[0])
			cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 4))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 0, 0))
			cg.builder.AddImmediate(0xFF)
			cg.builder.AddInstruction(rom.EncodeAND(0, 7, 0))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 6, 7))

			// Write scroll_x high byte
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(addrs[0] + 1)
			cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 4))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 0, 0))
			cg.builder.AddImmediate(8)
			cg.builder.AddInstruction(rom.EncodeSHR(0, 7, 0))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 6, 7))

			// Write scroll_y low byte
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(addrs[1])
			cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 5))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 0, 0))
			cg.builder.AddImmediate(0xFF)
			cg.builder.AddInstruction(rom.EncodeAND(0, 7, 0))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 6, 7))

			// Write scroll_y high byte
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(addrs[1] + 1)
			cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 5))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 0, 0))
			cg.builder.AddImmediate(8)
			cg.builder.AddInstruction(rom.EncodeSHR(0, 7, 0))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 6, 7))

			// Jump to end
			cg.builder.AddInstruction(rom.EncodeJMP())
			jumpPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			jumpToEnd = append(jumpToEnd, jumpPos)

			// Patch skip
			nextPC := uint16(cg.builder.GetCodeLength() * 2)
			skipPC := uint16(skipPos * 2)
			cg.builder.SetImmediateAt(skipPos, uint16(cg.calcBranchOffset(skipPC, nextPC)))
		}
		// Patch all jumps to end
		endPC := uint16(cg.builder.GetCodeLength() * 2)
		for _, jp := range jumpToEnd {
			jpPC := uint16(jp * 2)
			cg.builder.SetImmediateAt(jp, uint16(cg.calcBranchOffset(jpPC, endPC)))
		}
		return nil

	case "bg.enable":
		// bg.enable(layer: u8)
		// Args: R0 = layer (0-3)
		// BG control registers: BG0=0x8008, BG1=0x8009, BG2=0x8021, BG3=0x8026
		// Set bit 0 to enable.
		if len(args) != 1 {
			return fmt.Errorf("bg.enable requires 1 argument (layer)")
		}
		bgCtrlAddrs := []uint16{0x8008, 0x8009, 0x8021, 0x8026}
		jumpToEnd2 := make([]int, 0, 4)
		for i, addr := range bgCtrlAddrs {
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(uint16(i))
			cg.builder.AddInstruction(rom.EncodeCMP(0, 0, 6))
			cg.builder.AddInstruction(rom.EncodeBNE())
			skipPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)

			// Read current control, OR with 0x01, write back
			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
			cg.builder.AddImmediate(addr)
			cg.builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // read
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(0x01)
			cg.builder.AddInstruction(rom.EncodeOR(0, 5, 6))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // write

			cg.builder.AddInstruction(rom.EncodeJMP())
			jumpPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			jumpToEnd2 = append(jumpToEnd2, jumpPos)

			nextPC := uint16(cg.builder.GetCodeLength() * 2)
			skipPC := uint16(skipPos * 2)
			cg.builder.SetImmediateAt(skipPos, uint16(cg.calcBranchOffset(skipPC, nextPC)))
		}
		endPC2 := uint16(cg.builder.GetCodeLength() * 2)
		for _, jp := range jumpToEnd2 {
			jpPC := uint16(jp * 2)
			cg.builder.SetImmediateAt(jp, uint16(cg.calcBranchOffset(jpPC, endPC2)))
		}
		return nil

	case "bg.disable":
		// bg.disable(layer: u8)
		// Args: R0 = layer (0-3)
		// BG control registers: BG0=0x8008, BG1=0x8009, BG2=0x8021, BG3=0x8026
		// Clear bit 0 to disable while preserving other control bits.
		if len(args) != 1 {
			return fmt.Errorf("bg.disable requires 1 argument (layer)")
		}
		bgCtrlAddrs := []uint16{0x8008, 0x8009, 0x8021, 0x8026}
		jumpToEnd2 := make([]int, 0, 4)
		for i, addr := range bgCtrlAddrs {
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(uint16(i))
			cg.builder.AddInstruction(rom.EncodeCMP(0, 0, 6))
			cg.builder.AddInstruction(rom.EncodeBNE())
			skipPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)

			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
			cg.builder.AddImmediate(addr)
			cg.builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // read
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(0xFE)
			cg.builder.AddInstruction(rom.EncodeAND(0, 5, 6))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // write

			cg.builder.AddInstruction(rom.EncodeJMP())
			jumpPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			jumpToEnd2 = append(jumpToEnd2, jumpPos)

			nextPC := uint16(cg.builder.GetCodeLength() * 2)
			skipPC := uint16(skipPos * 2)
			cg.builder.SetImmediateAt(skipPos, uint16(cg.calcBranchOffset(skipPC, nextPC)))
		}
		endPC2 := uint16(cg.builder.GetCodeLength() * 2)
		for _, jp := range jumpToEnd2 {
			jpPC := uint16(jp * 2)
			cg.builder.SetImmediateAt(jp, uint16(cg.calcBranchOffset(jpPC, endPC2)))
		}
		return nil

	case "bg.set_tile_size":
		// bg.set_tile_size(layer: u8, size: u16)
		// Args: R0 = layer (0-3), R1 = size (8 or 16)
		// Bit 1 in BG control selects 16x16 tiles when set.
		if len(args) != 2 {
			return fmt.Errorf("bg.set_tile_size requires 2 arguments (layer, size)")
		}
		bgCtrlAddrs := []uint16{0x8008, 0x8009, 0x8021, 0x8026}
		jumpToEnd := make([]int, 0, 4)
		for i, addr := range bgCtrlAddrs {
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(uint16(i))
			cg.builder.AddInstruction(rom.EncodeCMP(0, 0, 6))
			cg.builder.AddInstruction(rom.EncodeBNE())
			skipPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)

			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
			cg.builder.AddImmediate(addr)
			cg.builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // current control
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(0xFD)                     // clear tile-size bit
			cg.builder.AddInstruction(rom.EncodeAND(0, 5, 6)) // R5 &= 0xFD

			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(16)
			cg.builder.AddInstruction(rom.EncodeCMP(0, 1, 6))
			cg.builder.AddInstruction(rom.EncodeBNE())
			noTileSetPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)

			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(0x02)
			cg.builder.AddInstruction(rom.EncodeOR(0, 5, 6))

			noTileSetPC := uint16(cg.builder.GetCodeLength() * 2)
			noTileSetBranchPC := uint16(noTileSetPos * 2)
			cg.builder.SetImmediateAt(noTileSetPos, uint16(cg.calcBranchOffset(noTileSetBranchPC, noTileSetPC)))

			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

			cg.builder.AddInstruction(rom.EncodeJMP())
			jumpPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			jumpToEnd = append(jumpToEnd, jumpPos)

			nextPC := uint16(cg.builder.GetCodeLength() * 2)
			skipPC := uint16(skipPos * 2)
			cg.builder.SetImmediateAt(skipPos, uint16(cg.calcBranchOffset(skipPC, nextPC)))
		}
		endPC := uint16(cg.builder.GetCodeLength() * 2)
		for _, jp := range jumpToEnd {
			jpPC := uint16(jp * 2)
			cg.builder.SetImmediateAt(jp, uint16(cg.calcBranchOffset(jpPC, endPC)))
		}
		return nil

	case "bg.set_priority":
		// bg.set_priority(layer: u8, priority: u8)
		// Args: R0 = layer, R1 = priority (0-3)
		// BG control registers: BG0=0x8008, BG1=0x8009, BG2=0x8021, BG3=0x8026
		// Bits [3:2] carry explicit layer priority.
		if len(args) != 2 {
			return fmt.Errorf("bg.set_priority requires 2 arguments (layer, priority)")
		}
		bgCtrlAddrs := []uint16{0x8008, 0x8009, 0x8021, 0x8026}
		jumpToEnd := make([]int, 0, 4)
		for i, addr := range bgCtrlAddrs {
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(uint16(i))
			cg.builder.AddInstruction(rom.EncodeCMP(0, 0, 6))
			cg.builder.AddInstruction(rom.EncodeBNE())
			skipPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)

			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
			cg.builder.AddImmediate(addr)
			cg.builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // R5 = current control
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(0xF3)                     // clear bits [3:2]
			cg.builder.AddInstruction(rom.EncodeAND(0, 5, 6)) // R5 = current & 0xF3

			cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 1)) // R7 = priority
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(0x03)
			cg.builder.AddInstruction(rom.EncodeAND(0, 7, 6)) // R7 &= 0x03
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(2)
			cg.builder.AddInstruction(rom.EncodeSHL(0, 7, 6)) // R7 <<= 2
			cg.builder.AddInstruction(rom.EncodeOR(0, 5, 7))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

			cg.builder.AddInstruction(rom.EncodeJMP())
			jumpPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			jumpToEnd = append(jumpToEnd, jumpPos)

			nextPC := uint16(cg.builder.GetCodeLength() * 2)
			skipPC := uint16(skipPos * 2)
			cg.builder.SetImmediateAt(skipPos, uint16(cg.calcBranchOffset(skipPC, nextPC)))
		}
		endPC := uint16(cg.builder.GetCodeLength() * 2)
		for _, jp := range jumpToEnd {
			jpPC := uint16(jp * 2)
			cg.builder.SetImmediateAt(jp, uint16(cg.calcBranchOffset(jpPC, endPC)))
		}
		return nil

	case "bg.set_tilemap_base":
		// bg.set_tilemap_base(layer: u8, base: u16)
		// Args: R0 = layer, R1 = tilemap base
		// Register pairs: BG0=0x8077/78, BG1=0x8079/7A, BG2=0x807B/7C, BG3=0x807D/7E
		if len(args) != 2 {
			return fmt.Errorf("bg.set_tilemap_base requires 2 arguments (layer, base)")
		}
		bgTilemapAddrs := []uint16{0x8077, 0x8079, 0x807B, 0x807D}
		jumpToEnd := make([]int, 0, 4)
		for i, addr := range bgTilemapAddrs {
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(uint16(i))
			cg.builder.AddInstruction(rom.EncodeCMP(0, 0, 6))
			cg.builder.AddInstruction(rom.EncodeBNE())
			skipPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)

			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
			cg.builder.AddImmediate(addr)
			cg.builder.AddInstruction(rom.EncodeMOV(0, 5, 1))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(0xFF)
			cg.builder.AddInstruction(rom.EncodeAND(0, 5, 6))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
			cg.builder.AddImmediate(addr + 1)
			cg.builder.AddInstruction(rom.EncodeMOV(0, 5, 1))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(8)
			cg.builder.AddInstruction(rom.EncodeSHR(0, 5, 6))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

			cg.builder.AddInstruction(rom.EncodeJMP())
			jumpPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			jumpToEnd = append(jumpToEnd, jumpPos)

			nextPC := uint16(cg.builder.GetCodeLength() * 2)
			skipPC := uint16(skipPos * 2)
			cg.builder.SetImmediateAt(skipPos, uint16(cg.calcBranchOffset(skipPC, nextPC)))
		}
		endPC := uint16(cg.builder.GetCodeLength() * 2)
		for _, jp := range jumpToEnd {
			jpPC := uint16(jp * 2)
			cg.builder.SetImmediateAt(jp, uint16(cg.calcBranchOffset(jpPC, endPC)))
		}
		return nil

	case "bg.load_tilemap":
		// bg.load_tilemap(asset: u16, layer: u8) -> u16
		// Args: R0 = asset ID (ASSET_* constant), R1 = layer
		// Loads packed tilemap bytes into the bound layer tilemap base. Returns the
		// tilemap base used for the transfer.
		if len(args) != 2 {
			return fmt.Errorf("bg.load_tilemap requires 2 arguments (asset, layer)")
		}

		if ident, ok := args[0].(*IdentExpr); ok && strings.HasPrefix(ident.Name, "ASSET_") {
			assetName := strings.TrimPrefix(ident.Name, "ASSET_")
			if asset, exists := cg.assets[assetName]; exists {
				return cg.generateInlineTilemapLoad(asset, args[1], destReg)
			}
		}

		if len(cg.program.Assets) == 0 {
			return fmt.Errorf("bg.load_tilemap requires declared tilemap assets in this source file")
		}
		return cg.generateRuntimeTilemapLoadDispatch(destReg)

	case "bg.set_tile":
		// bg.set_tile(layer: u8, x: u16, y: u16, tile: u8, attr: u8)
		// Args: R0 = layer, R1 = x, R2 = y, R3 = tile, R4 = attr
		// Writes a single 2-byte tilemap entry at the active layer tilemap base.
		// If the tilemap base register is unset (0), the helper falls back to 0x4000
		// to match the renderer's default tilemap base.
		if len(args) != 5 {
			return fmt.Errorf("bg.set_tile requires 5 arguments (layer, x, y, tile, attr)")
		}
		bgTilemapAddrs := []uint16{0x8077, 0x8079, 0x807B, 0x807D}
		jumpToEnd := make([]int, 0, 4)
		for i, addr := range bgTilemapAddrs {
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(uint16(i))
			cg.builder.AddInstruction(rom.EncodeCMP(0, 0, 6))
			cg.builder.AddInstruction(rom.EncodeBNE())
			skipPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)

			// Resolve tilemap base into R5.
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(addr)
			cg.builder.AddInstruction(rom.EncodeMOV(2, 5, 6)) // R5 = low byte
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(addr + 1)
			cg.builder.AddInstruction(rom.EncodeMOV(2, 6, 6)) // R6 = high byte
			cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
			cg.builder.AddImmediate(8)
			cg.builder.AddInstruction(rom.EncodeSHL(0, 6, 7))
			cg.builder.AddInstruction(rom.EncodeOR(0, 5, 6))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeCMP(0, 5, 7))
			cg.builder.AddInstruction(rom.EncodeBNE())
			baseReadyPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
			cg.builder.AddImmediate(0x4000)
			baseReadyPC := uint16(cg.builder.GetCodeLength() * 2)
			cg.builder.SetImmediateAt(baseReadyPos, uint16(cg.calcBranchOffset(uint16(baseReadyPos*2), baseReadyPC)))

			// offset = ((y << 5) + x) << 1
			cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 2))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
			cg.builder.AddImmediate(5)
			cg.builder.AddInstruction(rom.EncodeSHL(0, 6, 7))
			cg.builder.AddInstruction(rom.EncodeADD(0, 6, 1))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
			cg.builder.AddImmediate(1)
			cg.builder.AddInstruction(rom.EncodeSHL(0, 6, 7))
			cg.builder.AddInstruction(rom.EncodeADD(0, 5, 6))

			// Program VRAM address registers with R5.
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(0x800E)
			cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 5))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 0, 0))
			cg.builder.AddImmediate(0x00FF)
			cg.builder.AddInstruction(rom.EncodeAND(0, 7, 0))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 6, 7))

			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(0x800F)
			cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 5))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 0, 0))
			cg.builder.AddImmediate(8)
			cg.builder.AddInstruction(rom.EncodeSHR(0, 7, 0))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 6, 7))

			// Write tile + attr via auto-incrementing VRAM_DATA.
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(0x8010)
			cg.builder.AddInstruction(rom.EncodeMOV(3, 6, 3))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 6, 4))

			cg.builder.AddInstruction(rom.EncodeJMP())
			jumpPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			jumpToEnd = append(jumpToEnd, jumpPos)

			nextPC := uint16(cg.builder.GetCodeLength() * 2)
			skipPC := uint16(skipPos * 2)
			cg.builder.SetImmediateAt(skipPos, uint16(cg.calcBranchOffset(skipPC, nextPC)))
		}
		endPC := uint16(cg.builder.GetCodeLength() * 2)
		for _, jp := range jumpToEnd {
			jpPC := uint16(jp * 2)
			cg.builder.SetImmediateAt(jp, uint16(cg.calcBranchOffset(jpPC, endPC)))
		}
		return nil

	case "bg.fill_span":
		// bg.fill_span(layer: u8, x: u16, y: u16, count: u16, tile: u8, attr: u8)
		// Args: R0 = layer, R1 = x, R2 = y, R3 = count, R4 = tile, R5 = attr
		// Fills a contiguous run of tilemap entries on a single row.
		if len(args) != 6 {
			return fmt.Errorf("bg.fill_span requires 6 arguments (layer, x, y, count, tile, attr)")
		}
		bgTilemapAddrs := []uint16{0x8077, 0x8079, 0x807B, 0x807D}
		jumpToEnd := make([]int, 0, 4)
		for i, addr := range bgTilemapAddrs {
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(uint16(i))
			cg.builder.AddInstruction(rom.EncodeCMP(0, 0, 6))
			cg.builder.AddInstruction(rom.EncodeBNE())
			skipPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)

			// Resolve tilemap base into R6.
			cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
			cg.builder.AddImmediate(addr)
			cg.builder.AddInstruction(rom.EncodeMOV(2, 6, 7))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
			cg.builder.AddImmediate(addr + 1)
			cg.builder.AddInstruction(rom.EncodeMOV(2, 7, 7))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 0, 0))
			cg.builder.AddImmediate(8)
			cg.builder.AddInstruction(rom.EncodeSHL(0, 7, 0))
			cg.builder.AddInstruction(rom.EncodeOR(0, 6, 7))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeCMP(0, 6, 7))
			cg.builder.AddInstruction(rom.EncodeBNE())
			baseReadyPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(0x4000)
			baseReadyPC := uint16(cg.builder.GetCodeLength() * 2)
			cg.builder.SetImmediateAt(baseReadyPos, uint16(cg.calcBranchOffset(uint16(baseReadyPos*2), baseReadyPC)))

			// startAddr = base + (((y << 5) + x) << 1)
			cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 2))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 0, 0))
			cg.builder.AddImmediate(5)
			cg.builder.AddInstruction(rom.EncodeSHL(0, 7, 0))
			cg.builder.AddInstruction(rom.EncodeADD(0, 7, 1))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 0, 0))
			cg.builder.AddImmediate(1)
			cg.builder.AddInstruction(rom.EncodeSHL(0, 7, 0))
			cg.builder.AddInstruction(rom.EncodeADD(0, 6, 7))

			// Program VRAM address once; VRAM_DATA auto-increments.
			cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
			cg.builder.AddImmediate(0x800E)
			cg.builder.AddInstruction(rom.EncodeMOV(0, 1, 6))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 0, 0))
			cg.builder.AddImmediate(0x00FF)
			cg.builder.AddInstruction(rom.EncodeAND(0, 1, 0))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 7, 1))

			cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
			cg.builder.AddImmediate(0x800F)
			cg.builder.AddInstruction(rom.EncodeMOV(0, 1, 6))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 0, 0))
			cg.builder.AddImmediate(8)
			cg.builder.AddInstruction(rom.EncodeSHR(0, 1, 0))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 7, 1))

			// Loop count times writing tile/attr pairs.
			loopStart := cg.builder.GetCodeLength()
			cg.builder.AddInstruction(rom.EncodeMOV(1, 0, 0))
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeCMP(0, 3, 0))
			cg.builder.AddInstruction(rom.EncodeBEQ())
			loopEndPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)

			cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
			cg.builder.AddImmediate(0x8010)
			cg.builder.AddInstruction(rom.EncodeMOV(3, 7, 4))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 7, 5))

			cg.builder.AddInstruction(rom.EncodeMOV(1, 0, 0))
			cg.builder.AddImmediate(1)
			cg.builder.AddInstruction(rom.EncodeSUB(0, 3, 0))

			cg.builder.AddInstruction(rom.EncodeJMP())
			loopJumpPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(uint16(cg.calcBranchOffset(uint16(loopJumpPos*2), uint16(loopStart*2))))

			loopEndPC := uint16(cg.builder.GetCodeLength() * 2)
			cg.builder.SetImmediateAt(loopEndPos, uint16(cg.calcBranchOffset(uint16(loopEndPos*2), loopEndPC)))

			cg.builder.AddInstruction(rom.EncodeJMP())
			jumpPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			jumpToEnd = append(jumpToEnd, jumpPos)

			nextPC := uint16(cg.builder.GetCodeLength() * 2)
			skipPC := uint16(skipPos * 2)
			cg.builder.SetImmediateAt(skipPos, uint16(cg.calcBranchOffset(skipPC, nextPC)))
		}
		endPC := uint16(cg.builder.GetCodeLength() * 2)
		for _, jp := range jumpToEnd {
			jpPC := uint16(jp * 2)
			cg.builder.SetImmediateAt(jp, uint16(cg.calcBranchOffset(jpPC, endPC)))
		}
		return nil

	case "bg.clear":
		// bg.clear(layer: u8, tile: u8, attr: u8)
		// Args: R0 = layer, R1 = tile, R2 = attr
		// Clears the full 32x32 tilemap for a layer.
		if len(args) != 3 {
			return fmt.Errorf("bg.clear requires 3 arguments (layer, tile, attr)")
		}
		bgTilemapAddrs := []uint16{0x8077, 0x8079, 0x807B, 0x807D}
		jumpToEnd := make([]int, 0, 4)
		for i, addr := range bgTilemapAddrs {
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(uint16(i))
			cg.builder.AddInstruction(rom.EncodeCMP(0, 0, 6))
			cg.builder.AddInstruction(rom.EncodeBNE())
			skipPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)

			// Resolve tilemap base into R4.
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
			cg.builder.AddImmediate(addr)
			cg.builder.AddInstruction(rom.EncodeMOV(2, 4, 5))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
			cg.builder.AddImmediate(addr + 1)
			cg.builder.AddInstruction(rom.EncodeMOV(2, 5, 5))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(8)
			cg.builder.AddInstruction(rom.EncodeSHL(0, 5, 6))
			cg.builder.AddInstruction(rom.EncodeOR(0, 4, 5))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeCMP(0, 4, 6))
			cg.builder.AddInstruction(rom.EncodeBNE())
			baseReadyPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
			cg.builder.AddImmediate(0x4000)
			baseReadyPC := uint16(cg.builder.GetCodeLength() * 2)
			cg.builder.SetImmediateAt(baseReadyPos, uint16(cg.calcBranchOffset(uint16(baseReadyPos*2), baseReadyPC)))

			// Program VRAM address once.
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
			cg.builder.AddImmediate(0x800E)
			cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 4))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
			cg.builder.AddImmediate(0x00FF)
			cg.builder.AddInstruction(rom.EncodeAND(0, 6, 7))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 5, 6))

			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
			cg.builder.AddImmediate(0x800F)
			cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 4))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
			cg.builder.AddImmediate(8)
			cg.builder.AddInstruction(rom.EncodeSHR(0, 6, 7))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 5, 6))

			cg.builder.AddInstruction(rom.EncodeMOV(1, 3, 0))
			cg.builder.AddImmediate(1024)

			loopStart := cg.builder.GetCodeLength()
			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeCMP(0, 3, 4))
			cg.builder.AddInstruction(rom.EncodeBEQ())
			loopEndPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)

			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
			cg.builder.AddImmediate(0x8010)
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 1))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 2))

			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
			cg.builder.AddImmediate(1)
			cg.builder.AddInstruction(rom.EncodeSUB(0, 3, 4))

			cg.builder.AddInstruction(rom.EncodeJMP())
			loopJumpPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(uint16(cg.calcBranchOffset(uint16(loopJumpPos*2), uint16(loopStart*2))))

			loopEndPC := uint16(cg.builder.GetCodeLength() * 2)
			cg.builder.SetImmediateAt(loopEndPos, uint16(cg.calcBranchOffset(uint16(loopEndPos*2), loopEndPC)))

			cg.builder.AddInstruction(rom.EncodeJMP())
			jumpPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			jumpToEnd = append(jumpToEnd, jumpPos)

			nextPC := uint16(cg.builder.GetCodeLength() * 2)
			skipPC := uint16(skipPos * 2)
			cg.builder.SetImmediateAt(skipPos, uint16(cg.calcBranchOffset(skipPC, nextPC)))
		}
		endPC := uint16(cg.builder.GetCodeLength() * 2)
		for _, jp := range jumpToEnd {
			jpPC := uint16(jp * 2)
			cg.builder.SetImmediateAt(jp, uint16(cg.calcBranchOffset(jpPC, endPC)))
		}
		return nil

	case "bg.set_source_mode":
		// bg.set_source_mode(layer: u8, mode: u8)
		// Args: R0 = layer, R1 = source mode (0=tilemap, 1=bitmap)
		if len(args) != 2 {
			return fmt.Errorf("bg.set_source_mode requires 2 arguments (layer, mode)")
		}
		bgSourceModeAddrs := []uint16{0x8068, 0x8069, 0x806A, 0x806B}
		jumpToEnd := make([]int, 0, 4)
		for i, addr := range bgSourceModeAddrs {
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(uint16(i))
			cg.builder.AddInstruction(rom.EncodeCMP(0, 0, 6))
			cg.builder.AddInstruction(rom.EncodeBNE())
			skipPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)

			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
			cg.builder.AddImmediate(addr)
			cg.builder.AddInstruction(rom.EncodeMOV(0, 5, 1))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(0x01)
			cg.builder.AddInstruction(rom.EncodeAND(0, 5, 6))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

			cg.builder.AddInstruction(rom.EncodeJMP())
			jumpPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			jumpToEnd = append(jumpToEnd, jumpPos)

			nextPC := uint16(cg.builder.GetCodeLength() * 2)
			skipPC := uint16(skipPos * 2)
			cg.builder.SetImmediateAt(skipPos, uint16(cg.calcBranchOffset(skipPC, nextPC)))
		}
		endPC := uint16(cg.builder.GetCodeLength() * 2)
		for _, jp := range jumpToEnd {
			jpPC := uint16(jp * 2)
			cg.builder.SetImmediateAt(jp, uint16(cg.calcBranchOffset(jpPC, endPC)))
		}
		return nil

	case "bg.bind_transform", "matrix.bind":
		// bg.bind_transform(layer: u8, channel: u8)
		// matrix.bind(layer: u8, channel: u8)
		// Args: R0 = layer, R1 = transform channel (0-3)
		if len(args) != 2 {
			return fmt.Errorf("%s requires 2 arguments (layer, channel)", name)
		}
		bgTransformBindAddrs := []uint16{0x806C, 0x806D, 0x806E, 0x806F}
		jumpToEnd := make([]int, 0, 4)
		for i, addr := range bgTransformBindAddrs {
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(uint16(i))
			cg.builder.AddInstruction(rom.EncodeCMP(0, 0, 6))
			cg.builder.AddInstruction(rom.EncodeBNE())
			skipPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)

			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
			cg.builder.AddImmediate(addr)
			cg.builder.AddInstruction(rom.EncodeMOV(0, 5, 1))
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(0x03)
			cg.builder.AddInstruction(rom.EncodeAND(0, 5, 6))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

			cg.builder.AddInstruction(rom.EncodeJMP())
			jumpPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			jumpToEnd = append(jumpToEnd, jumpPos)

			nextPC := uint16(cg.builder.GetCodeLength() * 2)
			skipPC := uint16(skipPos * 2)
			cg.builder.SetImmediateAt(skipPos, uint16(cg.calcBranchOffset(skipPC, nextPC)))
		}
		endPC := uint16(cg.builder.GetCodeLength() * 2)
		for _, jp := range jumpToEnd {
			jpPC := uint16(jp * 2)
			cg.builder.SetImmediateAt(jp, uint16(cg.calcBranchOffset(jpPC, endPC)))
		}
		return nil

	default:
		return fmt.Errorf("%w: %s", errUnknownBuiltin, name)
	}
}
