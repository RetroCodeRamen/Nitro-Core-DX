package corelx

import (
	"fmt"
	"strings"

	"nitro-core-dx/internal/rom"
)

// Part of generateBuiltinCall. See the register contract on that function.
func (cg *CodeGenerator) builtinMatrixPlane(name string, args []Expr, destReg uint8) error {
	switch name {
	case "matrix_plane.load_bitmap":
		// matrix_plane.load_bitmap(asset, channel): upload an external image
		// asset's palette and bitmap onto a matrix plane (bitmap source mode).
		// The bitmap is DMA'd from the ROM data region; palette goes to CGRAM.
		if len(args) != 2 {
			return fmt.Errorf("matrix_plane.load_bitmap requires (asset, channel)")
		}
		ident, ok := args[0].(*IdentExpr)
		if !ok {
			return fmt.Errorf("matrix_plane.load_bitmap: first argument must be an image asset name")
		}
		img := cg.imageAssets[ident.Name]
		if img == nil {
			return fmt.Errorf("matrix_plane.load_bitmap: %q is not an image asset", ident.Name)
		}
		chVal, err := evalConstExpr(args[1], cg.consts)
		if err != nil {
			return fmt.Errorf("matrix_plane.load_bitmap: channel must be a constant: %w", err)
		}
		cg.emitLoadBitmap(img, uint8(chVal))
		return nil

	case "matrix_plane.set_projection":
		// set_projection(channel, mode, horizon): selects the plane, then sets
		// projection mode (0x8091) and horizon scanline (0x8092). mode: 0 none,
		// 1 perspective row projection, 2 vertical projected quad.
		if len(args) != 3 {
			return fmt.Errorf("matrix_plane.set_projection requires (channel, mode, horizon)")
		}
		if err := cg.selectMatrixPlane(args[0]); err != nil {
			return err
		}
		if err := cg.writePlaneByte(args[1], 0x8091); err != nil {
			return err
		}
		return cg.writePlaneByte(args[2], 0x8092)

	case "matrix_plane.set_depth":
		// set_depth(channel, base_distance, focal_length, width_scale): the 8.8
		// projection-depth registers (0x809B, 0x809D, 0x809F).
		if len(args) != 4 {
			return fmt.Errorf("matrix_plane.set_depth requires (channel, base_distance, focal_length, width_scale)")
		}
		if err := cg.selectMatrixPlane(args[0]); err != nil {
			return err
		}
		if err := cg.writePlaneReg16(args[1], 0x809B); err != nil {
			return err
		}
		if err := cg.writePlaneReg16(args[2], 0x809D); err != nil {
			return err
		}
		return cg.writePlaneReg16(args[3], 0x809F)

	case "matrix_plane.set_camera":
		// set_camera(channel, x, y, heading_x, heading_y): camera position
		// (0x8093/0x8095) and 8.8 heading vector (0x8097/0x8099).
		if len(args) != 5 {
			return fmt.Errorf("matrix_plane.set_camera requires (channel, x, y, heading_x, heading_y)")
		}
		if err := cg.selectMatrixPlane(args[0]); err != nil {
			return err
		}
		if err := cg.writePlaneReg16(args[1], 0x8093); err != nil {
			return err
		}
		if err := cg.writePlaneReg16(args[2], 0x8095); err != nil {
			return err
		}
		if err := cg.writePlaneReg16(args[3], 0x8097); err != nil {
			return err
		}
		return cg.writePlaneReg16(args[4], 0x8099)

	case "matrix_plane.set_surface":
		// set_surface(channel, origin_x, origin_y, facing_x, facing_y, height_scale):
		// world anchor (0x80A1/0x80A3), 8.8 facing vector (0x80A5/0x80A7), and
		// 8.8 vertical size (0x80A9) for vertical-projected-quad surfaces.
		if len(args) != 6 {
			return fmt.Errorf("matrix_plane.set_surface requires (channel, origin_x, origin_y, facing_x, facing_y, height_scale)")
		}
		if err := cg.selectMatrixPlane(args[0]); err != nil {
			return err
		}
		if err := cg.writePlaneReg16(args[1], 0x80A1); err != nil {
			return err
		}
		if err := cg.writePlaneReg16(args[2], 0x80A3); err != nil {
			return err
		}
		if err := cg.writePlaneReg16(args[3], 0x80A5); err != nil {
			return err
		}
		if err := cg.writePlaneReg16(args[4], 0x80A7); err != nil {
			return err
		}
		return cg.writePlaneReg16(args[5], 0x80A9)

	case "matrix_plane.set_flags":
		// set_flags(channel, transparent0, two_sided): the plane flags
		// register (0x808C) -- color-key-0 transparency (bit 0) and
		// two-sided/no-backface-cull rendering (bit 1). load_bitmap always
		// leaves this at 0 (opaque, one-sided), which is right for a floor
		// but wrong for a vertical billboard surface, so this exists to set
		// it explicitly afterward. The register is a full overwrite, not
		// additive, so both bits must be combined before the single store.
		if len(args) != 3 {
			return fmt.Errorf("matrix_plane.set_flags requires (channel, transparent0, two_sided)")
		}
		if err := cg.selectMatrixPlane(args[0]); err != nil {
			return err
		}
		// two_sided -> R6 = (two_sided & 1) << 1, saved aside first so the
		// transparent0 evaluation below is free to use R0 as scratch.
		if err := cg.generateExpr(args[2], 0); err != nil {
			return err
		}
		cg.hAndImm(0, 0x0001)
		cg.hShlImm(0, 1)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 0)) // R6 = two_sided<<1
		// transparent0 -> R0, then OR in the saved two_sided bit.
		if err := cg.generateExpr(args[1], 0); err != nil {
			return err
		}
		cg.hAndImm(0, 0x0001)
		cg.builder.AddInstruction(rom.EncodeOR(0, 0, 6))
		cg.storeIOByte(0x808C, 0)
		return nil

	case "matrix_plane.enable":
		// matrix_plane.enable(channel: u8, size: u16)
		// Args: R0 = channel, R1 = size in tiles (32, 64, 128)
		if len(args) != 2 {
			return fmt.Errorf("matrix_plane.enable requires 2 arguments (channel, size)")
		}
		cg.emitSelectMatrixPlane(0, 6, 7)
		// Default to 32x32 enabled.
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
		cg.builder.AddImmediate(0x01)

		// if size == 64 -> control = 0x03
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
		cg.builder.AddImmediate(64)
		cg.builder.AddInstruction(rom.EncodeCMP(0, 1, 6))
		cg.builder.AddInstruction(rom.EncodeBNE())
		not64Pos := cg.builder.GetCodeLength()
		cg.builder.AddImmediate(0)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
		cg.builder.AddImmediate(0x03)
		cg.builder.AddInstruction(rom.EncodeJMP())
		after64Pos := cg.builder.GetCodeLength()
		cg.builder.AddImmediate(0)
		not64PC := uint16(cg.builder.GetCodeLength() * 2)
		cg.builder.SetImmediateAt(not64Pos, uint16(cg.calcBranchOffset(uint16(not64Pos*2), not64PC)))

		// if size == 128 -> control = 0x05
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
		cg.builder.AddImmediate(128)
		cg.builder.AddInstruction(rom.EncodeCMP(0, 1, 6))
		cg.builder.AddInstruction(rom.EncodeBNE())
		not128Pos := cg.builder.GetCodeLength()
		cg.builder.AddImmediate(0)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
		cg.builder.AddImmediate(0x05)
		not128PC := uint16(cg.builder.GetCodeLength() * 2)
		cg.builder.SetImmediateAt(not128Pos, uint16(cg.calcBranchOffset(uint16(not128Pos*2), not128PC)))

		after64PC := uint16(cg.builder.GetCodeLength() * 2)
		cg.builder.SetImmediateAt(after64Pos, uint16(cg.calcBranchOffset(uint16(after64Pos*2), after64PC)))

		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
		cg.builder.AddImmediate(0x8081)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 6, 5))
		return nil

	case "matrix_plane.disable":
		// matrix_plane.disable(channel: u8)
		// Args: R0 = channel
		if len(args) != 1 {
			return fmt.Errorf("matrix_plane.disable requires 1 argument (channel)")
		}
		cg.emitSelectMatrixPlane(0, 6, 7)
		cg.emitLoadMMIO8(5, 0x8081)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
		cg.builder.AddImmediate(0x06)
		cg.builder.AddInstruction(rom.EncodeAND(0, 5, 6))
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
		cg.builder.AddImmediate(0x8081)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 6, 5))
		return nil

	case "matrix_plane.load_tiles":
		// matrix_plane.load_tiles(asset: u16, channel: u8, base: u16) -> u16
		// Args: R0 = asset ID, R1 = channel, R2 = base tile index
		if len(args) != 3 {
			return fmt.Errorf("matrix_plane.load_tiles requires 3 arguments (asset, channel, base)")
		}
		if ident, ok := args[0].(*IdentExpr); ok && strings.HasPrefix(ident.Name, "ASSET_") {
			assetName := strings.TrimPrefix(ident.Name, "ASSET_")
			if asset, exists := cg.assets[assetName]; exists {
				return cg.generateInlineMatrixPlaneTileLoadFromRegs(asset, 1, 2, destReg)
			}
		}
		if len(cg.program.Assets) == 0 {
			return fmt.Errorf("matrix_plane.load_tiles requires declared tile assets in this source file")
		}
		return cg.generateRuntimeMatrixPlaneTileLoadDispatch(destReg)

	case "matrix_plane.load_tilemap":
		// matrix_plane.load_tilemap(asset: u16, channel: u8)
		// Args: R0 = asset ID, R1 = channel
		if len(args) != 2 {
			return fmt.Errorf("matrix_plane.load_tilemap requires 2 arguments (asset, channel)")
		}
		if ident, ok := args[0].(*IdentExpr); ok && strings.HasPrefix(ident.Name, "ASSET_") {
			assetName := strings.TrimPrefix(ident.Name, "ASSET_")
			if asset, exists := cg.assets[assetName]; exists {
				return cg.generateInlineMatrixPlaneTilemapLoadFromChannelReg(asset, 1, destReg)
			}
		}
		if len(cg.program.Assets) == 0 {
			return fmt.Errorf("matrix_plane.load_tilemap requires declared tilemap assets in this source file")
		}
		return cg.generateRuntimeMatrixPlaneTilemapLoadDispatch(destReg)

	case "matrix_plane.set_tile":
		// matrix_plane.set_tile(channel: u8, x: u16, y: u16, tile: u8, attr: u8)
		// Args: R0 = channel, R1 = x, R2 = y, R3 = tile, R4 = attr
		if len(args) != 5 {
			return fmt.Errorf("matrix_plane.set_tile requires 5 arguments (channel, x, y, tile, attr)")
		}
		cg.emitSelectMatrixPlane(0, 6, 7)
		cg.emitSelectedMatrixPlaneShift(5, 6, 7)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 2)) // R6 = y
		cg.builder.AddInstruction(rom.EncodeSHL(0, 6, 5))
		cg.builder.AddInstruction(rom.EncodeADD(0, 6, 1))
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
		cg.builder.AddImmediate(1)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 6, 7))
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
		cg.builder.AddImmediate(0x8082)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 0, 6))
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
		cg.builder.AddImmediate(0x00FF)
		cg.builder.AddInstruction(rom.EncodeAND(0, 0, 5))
		cg.builder.AddInstruction(rom.EncodeMOV(3, 7, 0))
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
		cg.builder.AddImmediate(0x8083)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 0, 6))
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
		cg.builder.AddImmediate(8)
		cg.builder.AddInstruction(rom.EncodeSHR(0, 0, 5))
		cg.builder.AddInstruction(rom.EncodeMOV(3, 7, 0))
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
		cg.builder.AddImmediate(0x8084)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 7, 3))
		cg.builder.AddInstruction(rom.EncodeMOV(3, 7, 4))
		return nil

	case "matrix_plane.fill_rect":
		// matrix_plane.fill_rect(channel: u8, x: u16, y: u16, w: u16, h: u16, tile: u8, attr: u8)
		// Args: R0 = channel, R1 = x, R2 = y, R3 = w, R4 = h, R5 = tile, R6 = attr
		if len(args) != 7 {
			return fmt.Errorf("matrix_plane.fill_rect requires 7 arguments (channel, x, y, w, h, tile, attr)")
		}
		return cg.generateMatrixPlaneFillRect()

	case "matrix_plane.clear":
		// matrix_plane.clear(channel: u8, tile: u8, attr: u8)
		// Args: R0 = channel, R1 = tile, R2 = attr
		if len(args) != 3 {
			return fmt.Errorf("matrix_plane.clear requires 3 arguments (channel, tile, attr)")
		}
		return cg.generateMatrixPlaneClear()

	default:
		return fmt.Errorf("%w: %s", errUnknownBuiltin, name)
	}
}
