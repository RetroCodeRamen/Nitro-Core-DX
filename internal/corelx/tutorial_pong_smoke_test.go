package corelx_test

import (
	"path/filepath"
	"testing"

	"nitro-core-dx/internal/corelx"
	"nitro-core-dx/internal/emulator"
)

func spriteX(oam *[768]uint8, spriteIndex int) int {
	off := spriteIndex * 6
	return int(oam[off]) | (int(oam[off+1]) << 8)
}

func spriteY(oam *[768]uint8, spriteIndex int) int {
	off := spriteIndex * 6
	return int(oam[off+2])
}

func countNonBlack(buf []uint32, limit int) int {
	n := 0
	for _, px := range buf {
		if px != 0 {
			n++
			if n >= limit {
				return n
			}
		}
	}
	return n
}

func TestTutorialPongCompileAndRunSmoke(t *testing.T) {
	sourcePath := filepath.Join("..", "..", "test", "roms", "tutorial_pong.corelx")

	result, err := corelx.CompileProject(sourcePath, nil)
	if result == nil {
		t.Fatalf("CompileProject returned nil result (err=%v)", err)
	}
	if corelx.HasErrors(result.Diagnostics) {
		t.Fatalf("tutorial_pong.corelx has compile diagnostics: %+v", result.Diagnostics)
	}
	if err != nil {
		t.Fatalf("CompileProject returned error without error diagnostics: %v", err)
	}
	if len(result.ROMBytes) == 0 {
		t.Fatalf("CompileProject returned no ROM bytes")
	}

	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(result.ROMBytes); err != nil {
		t.Fatalf("LoadROM failed: %v", err)
	}
	emu.Start()

	initialLeftY := -1
	minLeftY := 1 << 30
	maxLeftY := -1
	prevBallX := -1
	sawBallMove := false
	sawVisibleFrame := false

	for frame := 0; frame < 180; frame++ {
		var buttons uint16
		switch {
		case frame >= 20 && frame < 50:
			buttons = 0x01 // Up
		case frame >= 70 && frame < 100:
			buttons = 0x02 // Down
		default:
			buttons = 0
		}
		emu.SetInputButtons(buttons)

		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame failed at frame %d: %v", frame, err)
		}

		leftY := spriteY(&emu.PPU.OAM, 0)
		ballX := spriteX(&emu.PPU.OAM, 2)
		if initialLeftY < 0 {
			initialLeftY = leftY
		}
		if leftY < minLeftY {
			minLeftY = leftY
		}
		if leftY > maxLeftY {
			maxLeftY = leftY
		}

		if prevBallX >= 0 && ballX != prevBallX {
			sawBallMove = true
		}
		prevBallX = ballX

		if !sawVisibleFrame && countNonBlack(emu.GetOutputBuffer(), 120) >= 120 {
			sawVisibleFrame = true
		}
	}

	if !sawVisibleFrame {
		t.Fatalf("tutorial_pong ROM produced no visible rendered pixels in smoke run")
	}
	if !sawBallMove {
		t.Fatalf("tutorial_pong ROM ball sprite did not move during smoke run")
	}
	if minLeftY >= initialLeftY {
		t.Fatalf("left paddle did not move up during scripted input (initial=%d min=%d)", initialLeftY, minLeftY)
	}
	if maxLeftY <= minLeftY {
		t.Fatalf("left paddle did not move back down during scripted input (min=%d max=%d)", minLeftY, maxLeftY)
	}
}
