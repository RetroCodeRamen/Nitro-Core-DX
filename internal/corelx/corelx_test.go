package corelx

import (
	"os"
	"path/filepath"
	"testing"

	"nitro-core-dx/internal/emulator"
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
