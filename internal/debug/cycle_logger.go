package debug

import (
	"fmt"
	"os"
	"sync"
)

// OAMReader interface for reading OAM data (to avoid import cycles)
type OAMReader interface {
	ReadOAM(offset uint8) uint8
}

// MemoryReader interface for reading memory (to avoid import cycles)
type MemoryReader interface {
	Read8(bank uint8, offset uint16) uint8
}

// PPUStateReader interface for reading PPU state (to avoid import cycles)
type PPUStateReader interface {
	GetScanline() int
	GetDot() int
	GetVBlankFlag() bool
	GetFrameCounter() uint16
}

// APUStateReader interface for reading APU state (to avoid import cycles)
type APUStateReader interface {
	GetChannelState(channel int) (enabled bool, frequency uint16, volume uint8, waveform uint8, duration uint16)
	GetMasterVolume() uint8
}

// CPUStateSnapshot represents CPU state for logging (to avoid import cycles)
type CPUStateSnapshot struct {
	R0, R1, R2, R3, R4, R5, R6, R7 uint16
	PCBank                          uint8
	PCOffset                        uint16
	PBR                             uint8
	DBR                             uint8
	SP                              uint16
	Flags                           uint8
	Cycles                          uint32
}

// CycleLogger logs CPU register and memory state for each clock cycle
// This is useful for debugging timing-sensitive issues
type CycleLogger struct {
	file        *os.File
	maxCycles   uint64
	startCycle  uint64 // Start logging after this many cycles
	currentCycle uint64
	totalCycles  uint64 // Total cycles since creation (for offset calculation)
	enabled     bool
	mu          sync.Mutex
	
	// Interfaces for reading memory and OAM state
	bus MemoryReader
	oam OAMReader
	ppu PPUStateReader
	apu APUStateReader
}

// NewCycleLogger creates a new cycle logger
// maxCycles: maximum number of cycles to log (0 = unlimited, but use with caution)
// startCycle: start logging after this many cycles (0 = start immediately)
func NewCycleLogger(filename string, maxCycles uint64, startCycle uint64, bus MemoryReader, oam OAMReader, ppu PPUStateReader, apu APUStateReader) (*CycleLogger, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create cycle log file: %w", err)
	}

	logger := &CycleLogger{
		file:        file,
		maxCycles:   maxCycles,
		startCycle:  startCycle,
		currentCycle: 0,
		totalCycles: 0,
		enabled:     true,
		bus:         bus,
		oam:         oam,
		ppu:         ppu,
		apu:         apu,
	}

	// Write header
	fmt.Fprintf(file, "Cycle-by-Cycle Debug Log\n")
	fmt.Fprintf(file, "========================\n\n")
	if startCycle > 0 {
		fmt.Fprintf(file, "Start cycle offset: %d\n", startCycle)
	}
	if maxCycles > 0 {
		fmt.Fprintf(file, "Max cycles to log: %d\n", maxCycles)
	}
	fmt.Fprintf(file, "\nFormat: Cycle | PC | Registers (R0-R7) | SP | PBR | DBR | Flags | PPU State | APU State | Key Memory\n")
	fmt.Fprintf(file, "PPU State: Scanline | Dot | VBlank | FrameCounter\n")
	fmt.Fprintf(file, "APU State: Ch0-3 (Enabled/Freq/Vol/Wave/Dur)\n")
	fmt.Fprintf(file, "Key Memory: VBlank(0x803E) | OAM_ADDR(0x8014) | OAM_DATA(0x8015) | OAM[0-5] (sprite 0)\n\n")

	return logger, nil
}

// LogCycle logs the CPU state and key memory locations for one cycle
func (c *CycleLogger) LogCycle(cpuState *CPUStateSnapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.enabled {
		return
	}

	c.totalCycles++

	// Check if we should start logging (offset)
	if c.totalCycles < c.startCycle {
		return
	}

	// Check if we've logged enough cycles
	if c.maxCycles > 0 && c.currentCycle >= c.maxCycles {
		c.enabled = false
		return
	}

	c.currentCycle++

	// Read key memory locations
	vblankFlag := uint8(0)
	oamAddr := uint8(0)
	oamData := uint8(0)
	oamSprite0 := [6]uint8{0, 0, 0, 0, 0, 0}

	if c.bus != nil {
		// VBlank flag at 0x803E (PPU register 0x3E)
		vblankFlag = c.bus.Read8(0, 0x803E)
		
		// OAM registers
		oamAddr = c.bus.Read8(0, 0x8014) // OAM_ADDR
		oamData = c.bus.Read8(0, 0x8015) // OAM_DATA
		
		// Read sprite 0 data from OAM
		if c.oam != nil {
			// Read sprite 0 data (6 bytes: X_low, X_high, Y, Tile, Attr, Ctrl)
			for i := 0; i < 6; i++ {
				oamSprite0[i] = c.oam.ReadOAM(uint8(i))
			}
		}
	}

	// Read PPU state
	ppuScanline := -1
	ppuDot := -1
	ppuVBlank := false
	ppuFrameCounter := uint16(0)
	if c.ppu != nil {
		ppuScanline = c.ppu.GetScanline()
		ppuDot = c.ppu.GetDot()
		ppuVBlank = c.ppu.GetVBlankFlag()
		ppuFrameCounter = c.ppu.GetFrameCounter()
	}

	// Read APU state
	apuChannels := [4]struct {
		enabled   bool
		frequency uint16
		volume    uint8
		waveform  uint8
		duration  uint16
	}{}
	apuMasterVol := uint8(0)
	if c.apu != nil {
		apuMasterVol = c.apu.GetMasterVolume()
		for i := 0; i < 4; i++ {
			apuChannels[i].enabled, apuChannels[i].frequency, apuChannels[i].volume, apuChannels[i].waveform, apuChannels[i].duration = c.apu.GetChannelState(i)
		}
	}

	// Format register state
	fmt.Fprintf(c.file, "Cycle %6d | PC %02X:%04X | ", c.totalCycles, cpuState.PCBank, cpuState.PCOffset)
	fmt.Fprintf(c.file, "R0:%04X R1:%04X R2:%04X R3:%04X R4:%04X R5:%04X R6:%04X R7:%04X | ",
		cpuState.R0, cpuState.R1, cpuState.R2, cpuState.R3,
		cpuState.R4, cpuState.R5, cpuState.R6, cpuState.R7)
	fmt.Fprintf(c.file, "SP:%04X | PBR:%02X | DBR:%02X | Flags:%02X (Z:%d N:%d C:%d V:%d I:%d D:%d) | ",
		cpuState.SP, cpuState.PBR, cpuState.DBR, cpuState.Flags,
		(cpuState.Flags>>0)&1, (cpuState.Flags>>1)&1, (cpuState.Flags>>2)&1,
		(cpuState.Flags>>3)&1, (cpuState.Flags>>4)&1, (cpuState.Flags>>5)&1)
	
	// PPU state
	fmt.Fprintf(c.file, "PPU:SL:%03d Dot:%03d VB:%v FC:%04d | ",
		ppuScanline, ppuDot, ppuVBlank, ppuFrameCounter)
	
	// APU state (compact format)
	fmt.Fprintf(c.file, "APU:MV:%02X ", apuMasterVol)
	for i := 0; i < 4; i++ {
		if apuChannels[i].enabled {
			fmt.Fprintf(c.file, "Ch%d:E/%04X/%02X/W%d/D%03d ", i, apuChannels[i].frequency, apuChannels[i].volume, apuChannels[i].waveform, apuChannels[i].duration)
		} else {
			fmt.Fprintf(c.file, "Ch%d:--- ", i)
		}
	}
	
	// Key memory
	fmt.Fprintf(c.file, "| VBlank:%02X | OAM_ADDR:%02X | OAM_DATA:%02X | OAM[0-5]:%02X %02X %02X %02X %02X %02X\n",
		vblankFlag, oamAddr, oamData,
		oamSprite0[0], oamSprite0[1], oamSprite0[2], oamSprite0[3], oamSprite0[4], oamSprite0[5])
}

// SetEnabled enables or disables logging
func (c *CycleLogger) SetEnabled(enabled bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = enabled
}

// Toggle toggles logging on/off
func (c *CycleLogger) Toggle() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.enabled = !c.enabled
}

// Close closes the log file
func (c *CycleLogger) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.enabled = false

	if c.file != nil {
		fmt.Fprintf(c.file, "\n\nLog complete. Total cycles logged: %d\n", c.currentCycle)
		err := c.file.Close()
		c.file = nil
		return err
	}
	return nil
}

// IsEnabled returns whether logging is enabled
func (c *CycleLogger) IsEnabled() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.enabled && (c.maxCycles == 0 || c.currentCycle < c.maxCycles)
}

// GetStatus returns the current logging status
func (c *CycleLogger) GetStatus() (enabled bool, currentCycle uint64, totalCycles uint64, maxCycles uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.enabled, c.currentCycle, c.totalCycles, c.maxCycles
}
