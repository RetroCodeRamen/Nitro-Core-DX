package corelx

import "fmt"

// Part of generateBuiltinCall. See the register contract on that function.
func (cg *CodeGenerator) builtinMusic(name string, args []Expr, destReg uint8) error {
	switch name {
	case "ym.write":
		// ym.write(addr: u8, value: u8) - YM2608 audio subsystem, port 0.
		// Writes the register address to the host address-select register
		// (FMRegAddr, 0x9100) then the data to the host data port (FMRegData,
		// 0x9101). Args: R0 = addr, R1 = value (storeIOByte uses only R7).
		if len(args) != 2 {
			return fmt.Errorf("ym.write requires 2 arguments (addr, value)")
		}
		cg.storeIOByte(0x9100, 0) // FMRegAddr: select register
		cg.storeIOByte(0x9101, 1) // FMRegData: data
		return nil

	case "ym.write_port1":
		// ym.write_port1(addr: u8, value: u8) - YM2608 audio subsystem, port 1
		// (upper port). Uses the host port-1 address/data registers (0x9104 /
		// 0x9105), the same path the bus/backend use for upper-port YM writes.
		// Args: R0 = addr, R1 = value.
		if len(args) != 2 {
			return fmt.Errorf("ym.write_port1 requires 2 arguments (addr, value)")
		}
		cg.storeIOByte(0x9104, 0) // port-1 address select
		cg.storeIOByte(0x9105, 1) // port-1 data
		return nil

	case "music.stop":
		// music.stop() - silence the chip and mark nothing playing.
		if len(args) != 0 {
			return fmt.Errorf("music.stop takes no arguments")
		}
		cg.emitYM2608Silence()
		cg.hMovImm(6, 0)
		cg.hStore16(musicActiveSlot, 6)
		return nil

	case "music.set_volume":
		// music.set_volume(value: u8) - sets FMRegVolume (0x9106) immediately,
		// canceling any in-progress fade (an explicit set wins over a stale
		// fade still counting down). Arg is in R0.
		if len(args) != 1 {
			return fmt.Errorf("music.set_volume requires 1 argument (0-255)")
		}
		cg.hMovImm(6, 0)
		cg.hStore16(musicFadeActiveSlot, 6)
		cg.storeIOByte(0x9106, 0)
		return nil

	case "music.fade_to":
		// music.fade_to(value: u8, frames: int) - fades FMRegVolume to value
		// over the given number of real frames. Advanced once per real frame
		// by __musicadvance's emitMusicFadeStep, which requires the program
		// to declare a music asset (that's what wires __musicadvance into
		// wait_vblank() at all). Args: R0 = value, R1 = frames.
		if len(args) != 2 {
			return fmt.Errorf("music.fade_to requires 2 arguments (value, frames)")
		}
		if len(cg.musicAssets) == 0 {
			return fmt.Errorf("music.fade_to requires the program to declare a music asset (its per-frame advance is driven by the same helper that plays songs)")
		}
		cg.hStore16(musicFadeTargetSlot, 0)
		cg.hStore16(musicFadeFramesLeftSlot, 1)
		cg.hMovImm(6, 1)
		cg.hStore16(musicFadeActiveSlot, 6)
		return nil

	default:
		return fmt.Errorf("%w: %s", errUnknownBuiltin, name)
	}
}
