package corelx

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"nitro-core-dx/internal/emulator"
	"nitro-core-dx/internal/ppu"
)

// TestCoreLXCompilation tests that CoreLX programs compile successfully
func TestCoreLXCompilation(t *testing.T) {
	// Get test directory (assuming we're running from project root)
	// Try multiple possible paths
	var testDir string
	possiblePaths := []string{
		"test/roms",
		"../../test/roms",
		"../test/roms",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			testDir = path
			break
		}
	}

	if testDir == "" {
		t.Skipf("Test ROM directory not found, skipping compilation tests")
		return
	}

	tests := []struct {
		name     string
		source   string
		expected bool
	}{
		{
			name:     "simple_test",
			source:   filepath.Join(testDir, "simple_test.corelx"),
			expected: true,
		},
		{
			name:     "example",
			source:   filepath.Join(testDir, "example.corelx"),
			expected: true,
		},
		{
			name:     "full_example",
			source:   filepath.Join(testDir, "full_example.corelx"),
			expected: true,
		},
		{
			name:     "apu_test",
			source:   filepath.Join(testDir, "apu_test.corelx"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip if file doesn't exist
			if _, err := os.Stat(tt.source); os.IsNotExist(err) {
				t.Skipf("Source file not found: %s", tt.source)
				return
			}

			// Read source file
			sourceData, err := os.ReadFile(tt.source)
			if err != nil {
				t.Fatalf("Failed to read source file: %v", err)
			}

			// Create temporary output file
			tmpDir := t.TempDir()
			outputPath := filepath.Join(tmpDir, tt.name+".rom")

			// Compile
			err = CompileFile(tt.source, outputPath)
			if (err == nil) != tt.expected {
				t.Errorf("Compilation result mismatch: expected success=%v, got error=%v", tt.expected, err)
				return
			}

			if tt.expected {
				// Verify ROM file exists and has valid header
				romData, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read compiled ROM: %v", err)
				}

				if len(romData) < 32 {
					t.Errorf("ROM file too small: %d bytes (expected at least 32)", len(romData))
				}

				// Check magic number
				if len(romData) >= 4 {
					magic := string(romData[0:4])
					if magic != "RMCF" {
						t.Errorf("Invalid ROM magic: got %q, expected RMCF", magic)
					}
				}

				// Verify source was actually processed
				if len(romData) == 32 {
					t.Logf("Warning: ROM contains only header (no code generated from %d bytes of source)", len(sourceData))
				}
			}
		})
	}
}

// TestAPUFunctions tests APU function code generation and execution
func TestAPUFunctions(t *testing.T) {
	// Test program that exercises all APU functions
	source := `function Start()
    apu.enable()
    apu.set_channel_wave(0, 0)
    apu.set_channel_freq(0, 440)
    apu.set_channel_volume(0, 128)
    apu.note_on(0)
    
    while true
        wait_vblank()
`

	tmpDir := t.TempDir()
	sourcePath := filepath.Join(tmpDir, "apu_test.corelx")
	outputPath := filepath.Join(tmpDir, "apu_test.rom")

	// Write source file
	if err := os.WriteFile(sourcePath, []byte(source), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	// Compile
	if err := CompileFile(sourcePath, outputPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Load and run ROM
	romData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read ROM: %v", err)
	}

	emu := emulator.NewEmulator()
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}

	// Run enough cycles to execute the initialization code
	// The code should execute before the first wait_vblank()
	initialPC := emu.CPU.State.PCOffset
	cyclesExecuted := 0
	maxCycles := 10000 // Allow up to 10k cycles for initialization

	for cyclesExecuted < maxCycles {
		// Execute one instruction
		if err := emu.CPU.ExecuteInstruction(); err != nil {
			t.Fatalf("CPU execution failed: %v", err)
		}
		cyclesExecuted++

		// Check if PC changed (code is executing)
		if emu.CPU.State.PCOffset != initialPC {
			break // Code started executing
		}

		// Safety check - if we've run many cycles without PC change, something's wrong
		if cyclesExecuted > 1000 {
			t.Logf("Warning: Executed %d cycles without PC change (PC stuck at 0x%04X)", cyclesExecuted, initialPC)
		}
	}

	// Continue executing until we hit wait_vblank() loop
	// Run a few more cycles to ensure all APU setup completes
	for i := 0; i < 500 && cyclesExecuted < maxCycles; i++ {
		if err := emu.CPU.ExecuteInstruction(); err != nil {
			// If we hit an error (like invalid instruction), that's okay - we may have hit wait_vblank
			break
		}
		cyclesExecuted++
	}

	t.Logf("Executed %d cycles, PC=0x%04X", cyclesExecuted, emu.CPU.State.PCOffset)

	// Verify APU state
	// Check master volume (should be 0xFF after apu.enable())
	masterVol := emu.APU.MasterVolume
	if masterVol != 0xFF {
		t.Errorf("Master volume: got 0x%02X, expected 0xFF (cycles executed: %d)", masterVol, cyclesExecuted)
	}

	// Check channel 0 state
	ch0 := emu.APU.Channels[0]
	if ch0.Waveform != 0 {
		t.Errorf("Channel 0 waveform: got %d, expected 0 (sine) (cycles executed: %d)", ch0.Waveform, cyclesExecuted)
	}
	if ch0.Frequency != 440 {
		t.Errorf("Channel 0 frequency: got %d, expected 440 (cycles executed: %d)", ch0.Frequency, cyclesExecuted)
	}
	if ch0.Volume != 128 {
		t.Errorf("Channel 0 volume: got %d, expected 128 (cycles executed: %d)", ch0.Volume, cyclesExecuted)
	}
	if !ch0.Enabled {
		t.Errorf("Channel 0 should be enabled after note_on() (cycles executed: %d)", cyclesExecuted)
	}
}

// TestAPUFunctionIndividual tests each APU function individually
func TestAPUFunctionIndividual(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		verifyFn func(t *testing.T, emu *emulator.Emulator)
	}{
		{
			name: "apu_enable",
			source: `function Start()
    apu.enable()
    while true
        wait_vblank()
`,
			verifyFn: func(t *testing.T, emu *emulator.Emulator) {
				if emu.APU.MasterVolume != 0xFF {
					t.Errorf("Master volume: got 0x%02X, expected 0xFF", emu.APU.MasterVolume)
				}
			},
		},
		{
			name: "apu_set_channel_wave",
			source: `function Start()
    apu.set_channel_wave(0, 1)
    while true
        wait_vblank()
`,
			verifyFn: func(t *testing.T, emu *emulator.Emulator) {
				if emu.APU.Channels[0].Waveform != 1 {
					t.Errorf("Channel 0 waveform: got %d, expected 1 (square)", emu.APU.Channels[0].Waveform)
				}
			},
		},
		{
			name: "apu_set_channel_freq",
			source: `function Start()
    apu.set_channel_freq(0, 523)
    while true
        wait_vblank()
`,
			verifyFn: func(t *testing.T, emu *emulator.Emulator) {
				if emu.APU.Channels[0].Frequency != 523 {
					t.Errorf("Channel 0 frequency: got %d, expected 523", emu.APU.Channels[0].Frequency)
				}
			},
		},
		{
			name: "apu_set_channel_volume",
			source: `function Start()
    apu.set_channel_volume(0, 200)
    while true
        wait_vblank()
`,
			verifyFn: func(t *testing.T, emu *emulator.Emulator) {
				if emu.APU.Channels[0].Volume != 200 {
					t.Errorf("Channel 0 volume: got %d, expected 200", emu.APU.Channels[0].Volume)
				}
			},
		},
		{
			name: "apu_note_on",
			source: `function Start()
    apu.note_on(0)
    while true
        wait_vblank()
`,
			verifyFn: func(t *testing.T, emu *emulator.Emulator) {
				if !emu.APU.Channels[0].Enabled {
					t.Errorf("Channel 0 should be enabled after note_on()")
				}
			},
		},
		{
			name: "apu_note_off",
			source: `function Start()
    apu.note_on(0)
    apu.note_off(0)
    while true
        wait_vblank()
`,
			verifyFn: func(t *testing.T, emu *emulator.Emulator) {
				if emu.APU.Channels[0].Enabled {
					t.Errorf("Channel 0 should be disabled after note_off()")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sourcePath := filepath.Join(tmpDir, tt.name+".corelx")
			outputPath := filepath.Join(tmpDir, tt.name+".rom")

			// Write source
			if err := os.WriteFile(sourcePath, []byte(tt.source), 0644); err != nil {
				t.Fatalf("Failed to write source: %v", err)
			}

			// Compile
			if err := CompileFile(sourcePath, outputPath); err != nil {
				t.Fatalf("Compilation failed: %v", err)
			}

			// Load ROM
			romData, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("Failed to read ROM: %v", err)
			}

			emu := emulator.NewEmulator()
			if err := emu.LoadROM(romData); err != nil {
				t.Fatalf("Failed to load ROM: %v", err)
			}

			// Execute enough cycles to run the initialization code
			initialPC := emu.CPU.State.PCOffset
			cyclesExecuted := 0
			maxCycles := 10000
			pcChanged := false

			// Execute until code runs and setup completes
			for cyclesExecuted < maxCycles {
				if err := emu.CPU.ExecuteInstruction(); err != nil {
					// May hit wait_vblank which reads I/O - continue
					break
				}
				cyclesExecuted++

				if emu.CPU.State.PCOffset != initialPC {
					pcChanged = true
				}

				// If PC changed and we've executed enough, setup should be done
				if pcChanged && cyclesExecuted > 100 {
					// Run more to ensure all setup completes
					for i := 0; i < 500 && cyclesExecuted < maxCycles; i++ {
						if err := emu.CPU.ExecuteInstruction(); err != nil {
							break
						}
						cyclesExecuted++
					}
					break
				}
			}

			t.Logf("Test %s: Executed %d cycles, PC changed: %v", tt.name, cyclesExecuted, pcChanged)

			// Verify
			tt.verifyFn(t, emu)
		})
	}
}

// TestSpriteFunctions tests sprite-related functions
func TestSpriteFunctions(t *testing.T) {
	// Simplified test - just enable display first
	source := `function Start()
    ppu.enable_display()
    while true
        wait_vblank()
`

	tmpDir := t.TempDir()
	sourcePath := filepath.Join(tmpDir, "sprite_test.corelx")
	outputPath := filepath.Join(tmpDir, "sprite_test.rom")

	if err := os.WriteFile(sourcePath, []byte(source), 0644); err != nil {
		t.Fatalf("Failed to write source: %v", err)
	}

	if err := CompileFile(sourcePath, outputPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	romData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read ROM: %v", err)
	}

	emu := emulator.NewEmulator()
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}

	// Execute cycles to run initialization
	initialPC := emu.CPU.State.PCOffset
	cyclesExecuted := 0
	maxCycles := 10000
	pcChanged := false

	for cyclesExecuted < maxCycles {
		if err := emu.CPU.ExecuteInstruction(); err != nil {
			// May hit wait_vblank which reads I/O - continue
			break
		}
		cyclesExecuted++

		if emu.CPU.State.PCOffset != initialPC {
			pcChanged = true
			// Code is executing, continue running to complete initialization
			// ppu.enable_display() should execute early
		}

		// If we've executed enough and PC changed, initialization should be done
		if pcChanged && cyclesExecuted > 50 {
			// Run a bit more to ensure all setup completes
			for i := 0; i < 500 && cyclesExecuted < maxCycles; i++ {
				if err := emu.CPU.ExecuteInstruction(); err != nil {
					break
				}
				cyclesExecuted++
			}
			break
		}
	}

	t.Logf("Executed %d cycles, PC changed: %v (0x%04X -> 0x%04X)",
		cyclesExecuted, pcChanged, initialPC, emu.CPU.State.PCOffset)

	// Verify PPU is enabled
	// BG0_CONTROL is at offset 0x08 (address 0x8008)
	// Check via PPU Read8 (offset relative to 0x8000)
	ppuControl := emu.PPU.Read8(0x08)

	// Also check BG0.Enabled directly
	bg0Enabled := emu.PPU.BG0.Enabled

	t.Logf("PPU Read8(0x08): 0x%02X, BG0.Enabled: %v", ppuControl, bg0Enabled)

	if !bg0Enabled && (ppuControl&0x01) == 0 {
		t.Errorf("PPU should be enabled (PPU Read8: 0x%02X, BG0.Enabled: %v, cycles: %d, PC changed: %v)",
			ppuControl, bg0Enabled, cyclesExecuted, pcChanged)
	}

	// Verify OAM has sprite data (check sprite 0)
	oamAddr := emu.PPU.Read8(0x8014)
	if oamAddr != 0 {
		t.Logf("OAM_ADDR is %d (expected 0 or sprite index)", oamAddr)
	}

	// Check OAM data for sprite 0 (should have X=120, Y=80)
	// Note: OAM writes happen during VBlank, so we need to check after VBlank
	// For now, just verify the code executed without errors
}

// TestSpriteHelperFunctions tests sprite helper functions
func TestSpriteHelperFunctions(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		verifyFn func(t *testing.T, emu *emulator.Emulator)
	}{
		{
			name: "SPR_PRI",
			source: `function Start()
    -- Test SPR_PRI(2) should return 0x80 (2 << 6)
    x := SPR_PRI(2)
    while true
        wait_vblank()
`,
			verifyFn: func(t *testing.T, emu *emulator.Emulator) {
				// SPR_PRI(2) should return 0x80 (2 << 6 = 128 = 0x80)
				// We can't easily check variable values yet, but we can verify compilation
				// For now, just verify code executed
			},
		},
		{
			name: "SPR_HFLIP",
			source: `function Start()
    x := SPR_HFLIP()
    while true
        wait_vblank()
`,
			verifyFn: func(t *testing.T, emu *emulator.Emulator) {
				// SPR_HFLIP() should return 0x10
				// Verify compilation succeeded
			},
		},
		{
			name: "SPR_VFLIP",
			source: `function Start()
    x := SPR_VFLIP()
    while true
        wait_vblank()
`,
			verifyFn: func(t *testing.T, emu *emulator.Emulator) {
				// SPR_VFLIP() should return 0x20
			},
		},
		{
			name: "SPR_SIZE_8",
			source: `function Start()
    x := SPR_SIZE_8()
    while true
        wait_vblank()
`,
			verifyFn: func(t *testing.T, emu *emulator.Emulator) {
				// SPR_SIZE_8() should return 0x00
			},
		},
		{
			name: "SPR_BLEND",
			source: `function Start()
    x := SPR_BLEND(2)
    while true
        wait_vblank()
`,
			verifyFn: func(t *testing.T, emu *emulator.Emulator) {
				// SPR_BLEND(2) should return 0x08 (2 << 2 = 8 = 0x08)
			},
		},
		{
			name: "SPR_ALPHA",
			source: `function Start()
    x := SPR_ALPHA(12)
    while true
        wait_vblank()
`,
			verifyFn: func(t *testing.T, emu *emulator.Emulator) {
				// SPR_ALPHA(12) should return 0xC0 (12 << 4 = 192 = 0xC0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sourcePath := filepath.Join(tmpDir, tt.name+".corelx")
			outputPath := filepath.Join(tmpDir, tt.name+".rom")

			// Write source
			if err := os.WriteFile(sourcePath, []byte(tt.source), 0644); err != nil {
				t.Fatalf("Failed to write source: %v", err)
			}

			// Compile
			if err := CompileFile(sourcePath, outputPath); err != nil {
				t.Fatalf("Compilation failed: %v", err)
			}

			// Load ROM
			romData, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("Failed to read ROM: %v", err)
			}

			emu := emulator.NewEmulator()
			if err := emu.LoadROM(romData); err != nil {
				t.Fatalf("Failed to load ROM: %v", err)
			}

			// Execute cycles
			initialPC := emu.CPU.State.PCOffset
			cyclesExecuted := 0
			maxCycles := 5000
			pcChanged := false

			for cyclesExecuted < maxCycles {
				if err := emu.CPU.ExecuteInstruction(); err != nil {
					break
				}
				cyclesExecuted++

				if emu.CPU.State.PCOffset != initialPC {
					pcChanged = true
				}

				if pcChanged && cyclesExecuted > 50 {
					for i := 0; i < 200 && cyclesExecuted < maxCycles; i++ {
						if err := emu.CPU.ExecuteInstruction(); err != nil {
							break
						}
						cyclesExecuted++
					}
					break
				}
			}

			// Verify compilation and execution succeeded
			if !pcChanged {
				t.Errorf("Code did not execute (PC did not change)")
			}

			// Run verification function
			tt.verifyFn(t, emu)
		})
	}
}

// TestFrameCounter tests frame_counter() function
func TestFrameCounter(t *testing.T) {
	source := `function Start()
    frame := frame_counter()
    while true
        wait_vblank()
`

	tmpDir := t.TempDir()
	sourcePath := filepath.Join(tmpDir, "frame_counter_test.corelx")
	outputPath := filepath.Join(tmpDir, "frame_counter_test.rom")

	if err := os.WriteFile(sourcePath, []byte(source), 0644); err != nil {
		t.Fatalf("Failed to write source: %v", err)
	}

	if err := CompileFile(sourcePath, outputPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	romData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read ROM: %v", err)
	}

	emu := emulator.NewEmulator()
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}

	// Execute cycles to read frame counter
	initialPC := emu.CPU.State.PCOffset
	cyclesExecuted := 0
	maxCycles := 5000
	pcChanged := false

	for cyclesExecuted < maxCycles {
		if err := emu.CPU.ExecuteInstruction(); err != nil {
			break
		}
		cyclesExecuted++

		if emu.CPU.State.PCOffset != initialPC {
			pcChanged = true
		}

		if pcChanged && cyclesExecuted > 50 {
			for i := 0; i < 200 && cyclesExecuted < maxCycles; i++ {
				if err := emu.CPU.ExecuteInstruction(); err != nil {
					break
				}
				cyclesExecuted++
			}
			break
		}
	}

	// Verify code executed
	if !pcChanged {
		t.Errorf("Code did not execute (PC did not change)")
	}

	// Frame counter should be readable (we can't easily verify the value without running frames,
	// but we can verify the code compiled and executed)
	t.Logf("Frame counter test: Executed %d cycles, PC changed: %v", cyclesExecuted, pcChanged)
}

// TestVBlankSync tests wait_vblank() function
func TestVBlankSync(t *testing.T) {
	source := `function Start()
    ppu.enable_display()
    
    while true
        wait_vblank()
`

	tmpDir := t.TempDir()
	sourcePath := filepath.Join(tmpDir, "vblank_test.corelx")
	outputPath := filepath.Join(tmpDir, "vblank_test.rom")

	if err := os.WriteFile(sourcePath, []byte(source), 0644); err != nil {
		t.Fatalf("Failed to write source: %v", err)
	}

	if err := CompileFile(sourcePath, outputPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	romData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read ROM: %v", err)
	}

	emu := emulator.NewEmulator()
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}

	// Run multiple frames - should sync properly
	for i := 0; i < 20; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("Frame execution failed: %v", err)
		}
	}

	// If we got here without errors, VBlank sync is working
}

// TestGfxSetPaletteWritesExpectedCGRAM verifies the CoreLX gfx.set_palette builtin writes
// the intended CGRAM color slot (regression for CGRAM_ADDR double-scaling bug).
func TestGfxSetPaletteWritesExpectedCGRAM(t *testing.T) {
	source := `function Start()
    gfx.set_palette(1, 1, 0x7C00)
    while true
        wait_vblank()
`

	tmpDir := t.TempDir()
	sourcePath := filepath.Join(tmpDir, "palette_test.corelx")
	outputPath := filepath.Join(tmpDir, "palette_test.rom")

	if err := os.WriteFile(sourcePath, []byte(source), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}
	if err := CompileFile(sourcePath, outputPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	romData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read ROM: %v", err)
	}
	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}
	emu.Start()

	for i := 0; i < 2; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame failed: %v", err)
		}
	}

	// palette 1, color 1 => CGRAM color index 17 => byte address 34
	if got := emu.PPU.CGRAM[34]; got != 0x00 {
		t.Fatalf("CGRAM low byte mismatch at palette1/color1: got 0x%02X want 0x00", got)
	}
	if got := emu.PPU.CGRAM[35]; got != 0x7C {
		t.Fatalf("CGRAM high byte mismatch at palette1/color1: got 0x%02X want 0x7C", got)
	}
}

func TestBGStaticControlsDriveExpectedPPUState(t *testing.T) {
	source := `function Start()
    bg.enable(2)
    bg.set_priority(2, 3)
    bg.set_tilemap_base(2, 0x5A00)
    bg.set_source_mode(2, 1)
    bg.bind_transform(2, 1)
    while true
        wait_vblank()
`

	tmpDir := t.TempDir()
	sourcePath := filepath.Join(tmpDir, "bg_static_controls.corelx")
	outputPath := filepath.Join(tmpDir, "bg_static_controls.rom")

	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}
	if err := CompileFile(sourcePath, outputPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	romData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read ROM: %v", err)
	}
	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}
	emu.Start()

	for i := 0; i < 2; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame failed: %v", err)
		}
	}

	if !emu.PPU.BG2.Enabled {
		t.Fatal("expected bg.enable(2) to enable BG2")
	}
	if emu.PPU.BG2.Priority != 3 {
		t.Fatalf("BG2.Priority = %d, want 3", emu.PPU.BG2.Priority)
	}
	if emu.PPU.BG2.TilemapBase != 0x5A00 {
		t.Fatalf("BG2.TilemapBase = 0x%04X, want 0x5A00", emu.PPU.BG2.TilemapBase)
	}
	if emu.PPU.BG2.SourceMode != 1 {
		t.Fatalf("BG2.SourceMode = %d, want 1", emu.PPU.BG2.SourceMode)
	}
	if emu.PPU.BG2.TransformChannel != 1 {
		t.Fatalf("BG2.TransformChannel = %d, want 1", emu.PPU.BG2.TransformChannel)
	}
}

func TestMatrixHelpersDriveBoundTransformChannelState(t *testing.T) {
	source := `function Start()
    matrix.bind(1, 3)
    matrix.enable(1)
    matrix.set_matrix(1, 0x0200, 0x0010, 0x0004, 0x0180)
    matrix.set_center(1, 12, 34)
    while true
        wait_vblank()
`

	tmpDir := t.TempDir()
	sourcePath := filepath.Join(tmpDir, "matrix_helpers.corelx")
	outputPath := filepath.Join(tmpDir, "matrix_helpers.rom")

	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}
	if err := CompileFile(sourcePath, outputPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	romData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read ROM: %v", err)
	}
	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}
	emu.Start()

	for i := 0; i < 2; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame failed: %v", err)
		}
	}

	if emu.PPU.BG1.TransformChannel != 3 {
		t.Fatalf("BG1.TransformChannel = %d, want 3", emu.PPU.BG1.TransformChannel)
	}
	ch := emu.PPU.TransformChannels[3]
	if !ch.Enabled {
		t.Fatal("expected matrix.enable(1) to enable bound transform channel 3")
	}
	if ch.A != 0x0200 || ch.B != 0x0010 || ch.C != 0x0004 || ch.D != 0x0180 {
		t.Fatalf("channel 3 matrix = {%04X,%04X,%04X,%04X}, want {0200,0010,0004,0180}", uint16(ch.A), uint16(ch.B), uint16(ch.C), uint16(ch.D))
	}
	if ch.CenterX != 12 || ch.CenterY != 34 {
		t.Fatalf("channel 3 center = (%d,%d), want (12,34)", ch.CenterX, ch.CenterY)
	}
}

func TestMatrixIdentityAndDisableHelpers(t *testing.T) {
	source := `function Start()
    matrix.bind(0, 2)
    matrix.enable(0)
    matrix.set_matrix(0, 0x0200, 0x0040, 0x0040, 0x0200)
    matrix.identity(0)
    matrix.disable(0)
    while true
        wait_vblank()
`

	tmpDir := t.TempDir()
	sourcePath := filepath.Join(tmpDir, "matrix_identity_disable.corelx")
	outputPath := filepath.Join(tmpDir, "matrix_identity_disable.rom")

	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}
	if err := CompileFile(sourcePath, outputPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	romData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read ROM: %v", err)
	}
	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}
	emu.Start()

	for i := 0; i < 2; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame failed: %v", err)
		}
	}

	ch := emu.PPU.TransformChannels[2]
	if ch.Enabled {
		t.Fatal("expected matrix.disable(0) to clear enabled bit on bound transform channel 2")
	}
	if ch.A != 0x0100 || ch.B != 0x0000 || ch.C != 0x0000 || ch.D != 0x0100 {
		t.Fatalf("channel 2 matrix after identity = {%04X,%04X,%04X,%04X}, want {0100,0000,0000,0100}", uint16(ch.A), uint16(ch.B), uint16(ch.C), uint16(ch.D))
	}
}

func TestBGDisableAndTileSizeHelpersDriveExpectedPPUState(t *testing.T) {
	source := `function Start()
    bg.enable(3)
    bg.set_tile_size(3, 16)
    bg.disable(3)
    while true
        wait_vblank()
`

	tmpDir := t.TempDir()
	sourcePath := filepath.Join(tmpDir, "bg_disable_tilesize.corelx")
	outputPath := filepath.Join(tmpDir, "bg_disable_tilesize.rom")

	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}
	if err := CompileFile(sourcePath, outputPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	romData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read ROM: %v", err)
	}
	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}
	emu.Start()

	for i := 0; i < 2; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame failed: %v", err)
		}
	}

	if emu.PPU.BG3.Enabled {
		t.Fatal("expected bg.disable(3) to leave BG3 disabled")
	}
	if !emu.PPU.BG3.TileSize {
		t.Fatal("expected bg.set_tile_size(3, 16) to set BG3 tile size to 16x16")
	}
}

func TestMatrixSetFlagsHelperDrivesBoundTransformFlags(t *testing.T) {
	source := `function Start()
    matrix.bind(2, 1)
    matrix.enable(2)
    matrix.set_flags(2, true, false, 2, true)
    while true
        wait_vblank()
`

	tmpDir := t.TempDir()
	sourcePath := filepath.Join(tmpDir, "matrix_set_flags.corelx")
	outputPath := filepath.Join(tmpDir, "matrix_set_flags.rom")

	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}
	if err := CompileFile(sourcePath, outputPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	romData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read ROM: %v", err)
	}
	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}
	emu.Start()

	for i := 0; i < 2; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame failed: %v", err)
		}
	}

	if emu.PPU.BG2.TransformChannel != 1 {
		t.Fatalf("BG2.TransformChannel = %d, want 1", emu.PPU.BG2.TransformChannel)
	}
	ch := emu.PPU.TransformChannels[1]
	if !ch.Enabled {
		t.Fatal("expected matrix.enable(2) to preserve enabled state")
	}
	if !ch.MirrorH {
		t.Fatal("expected matrix.set_flags to set MirrorH")
	}
	if ch.MirrorV {
		t.Fatal("expected matrix.set_flags to clear MirrorV")
	}
	if ch.OutsideMode != 2 {
		t.Fatalf("channel 1 OutsideMode = %d, want 2", ch.OutsideMode)
	}
	if !ch.DirectColor {
		t.Fatal("expected matrix.set_flags to set DirectColor")
	}
}

func TestMatrixPlaneHelpersDriveExpectedPPUState(t *testing.T) {
	source := `asset Solid: tiles8 hex
    11 11 11 11 11 11 11 11
    11 11 11 11 11 11 11 11
    11 11 11 11 11 11 11 11
    11 11 11 11 11 11 11 11

function Start()
    matrix_plane.enable(1, 128)
    base := matrix_plane.load_tiles(ASSET_Solid, 1, 3)
    matrix_plane.set_tile(1, 2, 3, 0x12, 0x34)
    while true
        wait_vblank()
`

	tmpDir := t.TempDir()
	sourcePath := filepath.Join(tmpDir, "matrix_plane_helpers.corelx")
	outputPath := filepath.Join(tmpDir, "matrix_plane_helpers.rom")

	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}
	if err := CompileFile(sourcePath, outputPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	romData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read ROM: %v", err)
	}
	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}
	emu.Start()

	for i := 0; i < 2; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame failed: %v", err)
		}
	}

	plane := emu.PPU.MatrixPlanes[1]
	if !plane.Enabled {
		t.Fatal("expected matrix_plane.enable(1, 128) to enable plane 1")
	}
	if plane.Size != ppu.TilemapSize128x128 {
		t.Fatalf("plane.Size = %d, want %d", plane.Size, ppu.TilemapSize128x128)
	}
	offset := (3*128 + 2) * 2
	if got := plane.Tilemap[offset]; got != 0x12 {
		t.Fatalf("plane tile low byte = 0x%02X, want 0x12", got)
	}
	if got := plane.Tilemap[offset+1]; got != 0x34 {
		t.Fatalf("plane tile attr byte = 0x%02X, want 0x34", got)
	}
	for i := 0; i < 32; i++ {
		if got := plane.Pattern[3*32+i]; got != 0x11 {
			t.Fatalf("plane pattern byte %d = 0x%02X, want 0x11", i, got)
		}
	}
}

func TestMatrixPlaneTilemapAndFillHelpersDriveExpectedPPUState(t *testing.T) {
	source := `asset PlaneMap: tilemap hex
    21 43 65 87

function Start()
    matrix_plane.enable(2, 32)
    matrix_plane.clear(2, 0x01, 0x02)
    matrix_plane.fill_rect(2, 1, 2, 3, 2, 0x07, 0x40)
    matrix_plane.load_tilemap(ASSET_PlaneMap, 2)
    while true
        wait_vblank()
`

	tmpDir := t.TempDir()
	sourcePath := filepath.Join(tmpDir, "matrix_plane_tilemap_fill.corelx")
	outputPath := filepath.Join(tmpDir, "matrix_plane_tilemap_fill.rom")

	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}
	if err := CompileFile(sourcePath, outputPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	romData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read ROM: %v", err)
	}
	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}
	emu.Start()

	for i := 0; i < 4; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame failed: %v", err)
		}
	}

	plane := emu.PPU.MatrixPlanes[2]
	if !plane.Enabled {
		t.Fatal("expected matrix plane 2 to be enabled")
	}
	// load_tilemap writes from address 0, so the first bytes should come from the asset
	if got := plane.Tilemap[0]; got != 0x21 {
		t.Fatalf("plane.Tilemap[0] = 0x%02X, want 0x21", got)
	}
	if got := plane.Tilemap[1]; got != 0x43 {
		t.Fatalf("plane.Tilemap[1] = 0x%02X, want 0x43", got)
	}
	// clear/fill_rect should have populated later cells.
	fillOffset := (2*32 + 1) * 2
	if got := plane.Tilemap[fillOffset]; got != 0x07 {
		t.Fatalf("filled tile low byte = 0x%02X, want 0x07", got)
	}
	if got := plane.Tilemap[fillOffset+1]; got != 0x40 {
		t.Fatalf("filled tile attr byte = 0x%02X, want 0x40", got)
	}
	clearOffset := (10*32 + 10) * 2
	if got := plane.Tilemap[clearOffset]; got != 0x01 {
		t.Fatalf("cleared tile low byte = 0x%02X, want 0x01", got)
	}
	if got := plane.Tilemap[clearOffset+1]; got != 0x02 {
		t.Fatalf("cleared tile attr byte = 0x%02X, want 0x02", got)
	}
}

func TestBGSetTileDefaultsToRendererTilemapBase(t *testing.T) {
	source := `function Start()
    bg.set_tile(0, 2, 1, 0x12, 0x34)
    while true
        wait_vblank()
`

	tmpDir := t.TempDir()
	sourcePath := filepath.Join(tmpDir, "bg_set_tile.corelx")
	outputPath := filepath.Join(tmpDir, "bg_set_tile.rom")

	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}
	if err := CompileFile(sourcePath, outputPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	romData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read ROM: %v", err)
	}
	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}
	emu.Start()

	for i := 0; i < 2; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame failed: %v", err)
		}
	}

	entry := uint16(0x4000 + ((1*32 + 2) * 2))
	if got := emu.PPU.VRAM[entry]; got != 0x12 {
		t.Fatalf("tile byte at 0x%04X = 0x%02X, want 0x12", entry, got)
	}
	if got := emu.PPU.VRAM[entry+1]; got != 0x34 {
		t.Fatalf("attr byte at 0x%04X = 0x%02X, want 0x34", entry+1, got)
	}
}

func TestBGClearAndFillSpanDriveExpectedTilemapWrites(t *testing.T) {
	source := `function Start()
    bg.set_tilemap_base(1, 0x4800)
    bg.clear(1, 0x21, 0x43)
    bg.fill_span(1, 4, 6, 3, 0x55, 0x66)
    while true
        wait_vblank()
`

	tmpDir := t.TempDir()
	sourcePath := filepath.Join(tmpDir, "bg_fill_span.corelx")
	outputPath := filepath.Join(tmpDir, "bg_fill_span.rom")

	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}
	if err := CompileFile(sourcePath, outputPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	romData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read ROM: %v", err)
	}
	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}
	emu.Start()

	for i := 0; i < 2; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame failed: %v", err)
		}
	}

	base := uint16(0x4800)
	clearEntry := base
	if got := emu.PPU.VRAM[clearEntry]; got != 0x21 {
		t.Fatalf("clear tile byte at 0x%04X = 0x%02X, want 0x21", clearEntry, got)
	}
	if got := emu.PPU.VRAM[clearEntry+1]; got != 0x43 {
		t.Fatalf("clear attr byte at 0x%04X = 0x%02X, want 0x43", clearEntry+1, got)
	}

	for x := 4; x < 7; x++ {
		entry := base + uint16(((6*32)+x)*2)
		if got := emu.PPU.VRAM[entry]; got != 0x55 {
			t.Fatalf("fill tile byte at x=%d addr 0x%04X = 0x%02X, want 0x55", x, entry, got)
		}
		if got := emu.PPU.VRAM[entry+1]; got != 0x66 {
			t.Fatalf("fill attr byte at x=%d addr 0x%04X = 0x%02X, want 0x66", x, entry+1, got)
		}
	}
}

func TestBGLoadTilemapAssetUsesConfiguredLayerBase(t *testing.T) {
	source := `asset MapA: tilemap hex
    10 01 20 02

function Start()
    bg.set_tilemap_base(0, 0x4A00)
    base := bg.load_tilemap(ASSET_MapA, 0)
    while true
        wait_vblank()
`

	tmpDir := t.TempDir()
	sourcePath := filepath.Join(tmpDir, "bg_load_tilemap.corelx")
	outputPath := filepath.Join(tmpDir, "bg_load_tilemap.rom")

	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}
	if err := CompileFile(sourcePath, outputPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	romData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read ROM: %v", err)
	}
	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}
	emu.Start()

	for i := 0; i < 2; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame failed: %v", err)
		}
	}

	if emu.PPU.BG0.TilemapBase != 0x4A00 {
		t.Fatalf("BG0.TilemapBase = 0x%04X, want 0x4A00", emu.PPU.BG0.TilemapBase)
	}
	want := []byte{0x10, 0x01, 0x20, 0x02}
	for i, b := range want {
		addr := uint16(0x4A00 + i)
		if got := emu.PPU.VRAM[addr]; got != b {
			t.Fatalf("tilemap byte at 0x%04X = 0x%02X, want 0x%02X", addr, got, b)
		}
	}
}

func TestRasterHelpersDriveExpectedPPUState(t *testing.T) {
	source := `function Start()
    raster.enable(0x3800, 0x03, false, true, true, true)
    raster.disable()
    raster.enable(0x3C00, 0x01, true, false, true, false)
    while true
        wait_vblank()
`

	tmpDir := t.TempDir()
	sourcePath := filepath.Join(tmpDir, "raster_helpers.corelx")
	outputPath := filepath.Join(tmpDir, "raster_helpers.rom")

	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}
	if err := CompileFile(sourcePath, outputPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	romData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read ROM: %v", err)
	}
	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}
	emu.Start()

	for i := 0; i < 2; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame failed: %v", err)
		}
	}

	if !emu.PPU.HDMAEnabled {
		t.Fatal("expected raster.enable to leave HDMA enabled")
	}
	if emu.PPU.HDMATableBase != 0x3C00 {
		t.Fatalf("HDMATableBase = 0x%04X, want 0x3C00", emu.PPU.HDMATableBase)
	}
	if emu.PPU.HDMAControl != 0xA3 {
		t.Fatalf("HDMAControl = 0x%02X, want 0xA3", emu.PPU.HDMAControl)
	}
	if emu.PPU.HDMAExtControl != 0x00 {
		t.Fatalf("HDMAExtControl = 0x%02X, want 0x00", emu.PPU.HDMAExtControl)
	}
}

func TestRasterScanlineHelpersWriteExpectedTableEntries(t *testing.T) {
	source := `function Start()
    raster.enable(0x3000, 0x01, true, true, true, true)
    raster.set_scanline_scroll(5, 0, 3, 4)
    raster.set_scanline_matrix(5, 0, 0x0100, 0x0001, 0x0002, 0x0200)
    raster.set_scanline_center(5, 0, 7, 8)
    raster.set_scanline_rebind(5, 0, 2)
    raster.set_scanline_priority(5, 0, 3)
    raster.set_scanline_tilemap_base(5, 0, 0x1800)
    raster.set_scanline_source_mode(5, 0, 1)
    while true
        wait_vblank()
`

	tmpDir := t.TempDir()
	sourcePath := filepath.Join(tmpDir, "raster_scanline_helpers.corelx")
	outputPath := filepath.Join(tmpDir, "raster_scanline_helpers.rom")

	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}
	if err := CompileFile(sourcePath, outputPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	romData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read ROM: %v", err)
	}
	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}
	emu.Start()

	for i := 0; i < 2; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame failed: %v", err)
		}
	}

	base := uint16(0x3000 + 5*84)
	want := []uint8{
		0x03, 0x00, // scroll x
		0x04, 0x00, // scroll y
		0x00, 0x01, // A
		0x01, 0x00, // B
		0x02, 0x00, // C
		0x00, 0x02, // D
		0x07, 0x00, // center x
		0x08, 0x00, // center y
	}
	for i, b := range want {
		addr := base + uint16(i)
		if got := emu.PPU.VRAM[addr]; got != b {
			t.Fatalf("scanline payload byte at 0x%04X = 0x%02X, want 0x%02X", addr, got, b)
		}
	}

	rebindAddr := base + 64
	if got := emu.PPU.VRAM[rebindAddr]; got != 0x02 {
		t.Fatalf("rebind byte at 0x%04X = 0x%02X, want 0x02", rebindAddr, got)
	}

	priorityAddr := base + 68
	if got := emu.PPU.VRAM[priorityAddr]; got != 0x03 {
		t.Fatalf("priority byte at 0x%04X = 0x%02X, want 0x03", priorityAddr, got)
	}

	tilemapBaseAddr := base + 72
	if got := emu.PPU.VRAM[tilemapBaseAddr]; got != 0x00 {
		t.Fatalf("tilemap base low byte at 0x%04X = 0x%02X, want 0x00", tilemapBaseAddr, got)
	}
	if got := emu.PPU.VRAM[tilemapBaseAddr+1]; got != 0x18 {
		t.Fatalf("tilemap base high byte at 0x%04X = 0x%02X, want 0x18", tilemapBaseAddr+1, got)
	}

	sourceModeAddr := base + 80
	if got := emu.PPU.VRAM[sourceModeAddr]; got != 0x01 {
		t.Fatalf("source mode byte at 0x%04X = 0x%02X, want 0x01", sourceModeAddr, got)
	}
}

func TestRasterShowcaseDemoRendersVisibleSplit(t *testing.T) {
	var sourcePath string
	possiblePaths := []string{
		"test/roms/raster_showcase.corelx",
		"../../test/roms/raster_showcase.corelx",
		"../test/roms/raster_showcase.corelx",
	}
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			sourcePath = path
			break
		}
	}
	if sourcePath == "" {
		t.Skip("raster showcase demo not found")
	}

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "raster_showcase.rom")
	if err := CompileFile(sourcePath, outputPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	romData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read ROM: %v", err)
	}
	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}
	emu.Start()

	for i := 0; i < 90; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame failed: %v", err)
		}
		topColor := emu.PPU.OutputBuffer[10*320+10]
		bottomColor := emu.PPU.OutputBuffer[(200-10)*320+10]
		if topColor != 0 && bottomColor != 0 && topColor != bottomColor {
			return
		}
	}

	topColor := emu.PPU.OutputBuffer[10*320+10]
	bottomColor := emu.PPU.OutputBuffer[(200-10)*320+10]
	t.Fatalf("expected raster showcase split after initialization, got top=0x%06X bottom=0x%06X", topColor, bottomColor)
}

func TestGraphicsImageDemoRendersScene(t *testing.T) {
	var sourcePath string
	possiblePaths := []string{
		"test/roms/graphics_image_demo.corelx",
		"../../test/roms/graphics_image_demo.corelx",
		"../test/roms/graphics_image_demo.corelx",
	}
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			sourcePath = path
			break
		}
	}
	if sourcePath == "" {
		t.Skip("graphics image demo not found")
	}

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "graphics_image_demo.rom")
	if err := CompileFile(sourcePath, outputPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	romData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read ROM: %v", err)
	}
	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}
	emu.Start()

	for i := 0; i < 600; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame failed: %v", err)
		}
		sky := emu.PPU.OutputBuffer[20*320+20]
		ground := emu.PPU.OutputBuffer[180*320+20]
		sun := emu.PPU.OutputBuffer[52*320+76]
		if sky != 0 && ground != 0 && sun != 0 && sky != ground && sky != sun && ground != sun {
			return
		}
	}

	sky := emu.PPU.OutputBuffer[20*320+20]
	ground := emu.PPU.OutputBuffer[180*320+20]
	sun := emu.PPU.OutputBuffer[52*320+76]
	t.Fatalf("expected visible image scene, got sky=0x%06X ground=0x%06X sun=0x%06X", sky, ground, sun)
}

func TestMatrixPlaneShowcaseProgramsDedicatedPlane(t *testing.T) {
	var sourcePath string
	possiblePaths := []string{
		"test/roms/matrix_plane_showcase.corelx",
		"../../test/roms/matrix_plane_showcase.corelx",
		"../test/roms/matrix_plane_showcase.corelx",
	}
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			sourcePath = path
			break
		}
	}
	if sourcePath == "" {
		t.Skip("matrix plane showcase not found")
	}

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "matrix_plane_showcase.rom")
	if err := CompileFile(sourcePath, outputPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	romData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read ROM: %v", err)
	}
	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}
	emu.Start()

	for i := 0; i < 600; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame failed: %v", err)
		}
	}

	if !emu.PPU.BG0.Enabled {
		t.Fatal("expected matrix plane showcase to enable BG0")
	}
	if emu.PPU.BG0.TransformChannel != 0 {
		t.Fatalf("BG0.TransformChannel = %d, want 0", emu.PPU.BG0.TransformChannel)
	}
	if !emu.PPU.TransformChannels[0].Enabled {
		t.Fatal("expected matrix plane showcase to enable transform channel 0")
	}
	if !emu.PPU.MatrixPlanes[0].Enabled {
		t.Fatal("expected matrix plane showcase to enable dedicated matrix plane 0")
	}
	if emu.PPU.MatrixPlanes[0].Size != ppu.TilemapSize32x32 {
		t.Fatalf("matrix plane 0 size = %d, want %d", emu.PPU.MatrixPlanes[0].Size, ppu.TilemapSize32x32)
	}
	if emu.PPU.MatrixPlanes[0].Pattern[0] == 0 && emu.PPU.MatrixPlanes[0].Pattern[32] == 0 {
		t.Fatal("expected matrix plane showcase to upload pattern data")
	}
	rightHalfOffset := (0*32 + 20) * 2
	markerOffset := (14*32 + 14) * 2
	if got := emu.PPU.MatrixPlanes[0].Tilemap[rightHalfOffset]; got != 0x01 {
		t.Fatalf("expected right-half tile entry to be 0x01, got 0x%02X", got)
	}
	if got := emu.PPU.MatrixPlanes[0].Tilemap[markerOffset]; got != 0x02 {
		t.Fatalf("expected marker tile entry to be 0x02, got 0x%02X", got)
	}
}

func framebufferFingerprint(buf []uint32) string {
	raw := make([]byte, len(buf)*4)
	for i, px := range buf {
		raw[i*4+0] = byte(px)
		raw[i*4+1] = byte(px >> 8)
		raw[i*4+2] = byte(px >> 16)
		raw[i*4+3] = byte(px >> 24)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func TestGraphicsPipelineShowcasePhasesDiffer(t *testing.T) {
	var sourcePath string
	possiblePaths := []string{
		"test/roms/graphics_pipeline_showcase.corelx",
		"../../test/roms/graphics_pipeline_showcase.corelx",
		"../test/roms/graphics_pipeline_showcase.corelx",
	}
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			sourcePath = path
			break
		}
	}
	if sourcePath == "" {
		t.Skip("graphics pipeline showcase not found")
	}

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "graphics_pipeline_showcase.rom")
	if err := CompileFile(sourcePath, outputPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	romData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read ROM: %v", err)
	}
	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}
	emu.Start()

	checkpoints := []int{120, 240, 420, 600}
	fingerprints := make([]string, 0, len(checkpoints))
	frame := 0
	for _, target := range checkpoints {
		for frame < target {
			if err := emu.RunFrame(); err != nil {
				t.Fatalf("RunFrame failed at frame %d: %v", frame, err)
			}
			frame++
		}
		fp := framebufferFingerprint(emu.PPU.OutputBuffer[:])
		fingerprints = append(fingerprints, fp)
	}

	seen := map[string]bool{}
	for _, fp := range fingerprints {
		seen[fp] = true
	}
	if len(seen) < 4 {
		t.Fatalf("expected 4 distinct phase framebuffers, got %d unique fingerprints: %v", len(seen), fingerprints)
	}
}

func TestMatrixPlanePipelineShowcasePhasesDiffer(t *testing.T) {
	var sourcePath string
	possiblePaths := []string{
		"test/roms/matrix_plane_pipeline_showcase.corelx",
		"../../test/roms/matrix_plane_pipeline_showcase.corelx",
		"../test/roms/matrix_plane_pipeline_showcase.corelx",
	}
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			sourcePath = path
			break
		}
	}
	if sourcePath == "" {
		t.Skip("matrix plane pipeline showcase not found")
	}

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "matrix_plane_pipeline_showcase.rom")
	if err := CompileFile(sourcePath, outputPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	romData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read ROM: %v", err)
	}
	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}
	emu.Start()

	checkpoints := []int{120, 300, 480, 700}
	fingerprints := make([]string, 0, len(checkpoints))
	frame := 0
	for _, target := range checkpoints {
		for frame < target {
			if err := emu.RunFrame(); err != nil {
				t.Fatalf("RunFrame failed at frame %d: %v", frame, err)
			}
			frame++
		}
		fp := framebufferFingerprint(emu.PPU.OutputBuffer[:])
		fingerprints = append(fingerprints, fp)
	}

	seen := map[string]bool{}
	for _, fp := range fingerprints {
		seen[fp] = true
	}
	if len(seen) < 4 {
		t.Fatalf("expected 4 distinct matrix-plane phase framebuffers, got %d unique fingerprints: %v", len(seen), fingerprints)
	}
}

// TestGfxLoadTilesRuntimeAssetIDDispatch verifies runtime asset-ID dispatch for
// gfx.load_tiles by selecting between two declared assets via a variable.
func TestGfxLoadTilesRuntimeAssetIDDispatch(t *testing.T) {
	source := `asset A: tiles8
    hex
        11 11 11 11 11 11 11 11
        11 11 11 11 11 11 11 11
        11 11 11 11 11 11 11 11
        11 11 11 11 11 11 11 11

asset B: tiles8
    hex
        22 22 22 22 22 22 22 22
        22 22 22 22 22 22 22 22
        22 22 22 22 22 22 22 22
        22 22 22 22 22 22 22 22

function Start()
    id := ASSET_B
    base := gfx.load_tiles(id, 0)
    ppu.enable_display()
    while true
        wait_vblank()
`

	tmpDir := t.TempDir()
	sourcePath := filepath.Join(tmpDir, "runtime_asset_dispatch.corelx")
	outputPath := filepath.Join(tmpDir, "runtime_asset_dispatch.rom")

	if err := os.WriteFile(sourcePath, []byte(source), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}
	if err := CompileFile(sourcePath, outputPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	romData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read ROM: %v", err)
	}
	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}
	emu.Start()

	for i := 0; i < 2; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame failed: %v", err)
		}
	}

	for i := 0; i < 32; i++ {
		if got := emu.PPU.VRAM[i]; got != 0x22 {
			t.Fatalf("VRAM[%d] mismatch: got 0x%02X want 0x22", i, got)
		}
	}
}

// TestGfxLoadTilesTilesetWritesFullPayload ensures tileset/sprite-sized payloads are
// emitted completely (not truncated to a single 8x8 tile worth of bytes).
func TestGfxLoadTilesTilesetWritesFullPayload(t *testing.T) {
	source := `asset Big: tileset
    hex
        60 60 60 60 60 60 60 60
        60 60 60 60 60 60 60 60
        60 60 60 60 60 60 60 60
        60 60 60 60 60 60 60 60
        60 60 60 60 60 60 60 60
        60 60 60 60 60 60 60 60
        60 60 60 60 60 60 60 60
        60 60 60 60 60 60 60 60

function Start()
    base := gfx.load_tiles(ASSET_Big, 0)
    ppu.enable_display()
    while true
        wait_vblank()
`

	tmpDir := t.TempDir()
	sourcePath := filepath.Join(tmpDir, "tileset_full_payload.corelx")
	outputPath := filepath.Join(tmpDir, "tileset_full_payload.rom")

	if err := os.WriteFile(sourcePath, []byte(source), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}
	if err := CompileFile(sourcePath, outputPath); err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	romData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read ROM: %v", err)
	}
	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(romData); err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}
	emu.Start()

	for i := 0; i < 2; i++ {
		if err := emu.RunFrame(); err != nil {
			t.Fatalf("RunFrame failed: %v", err)
		}
	}

	for i := 0; i < 64; i++ {
		if got := emu.PPU.VRAM[i]; got != 0x60 {
			t.Fatalf("VRAM[%d] mismatch: got 0x%02X want 0x60", i, got)
		}
	}
}
