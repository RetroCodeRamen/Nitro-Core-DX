package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

// Test CPU execution by simulating instruction execution
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: test_cpu_execution <rom.rom>")
		os.Exit(1)
	}

	romData, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading ROM: %v\n", err)
		os.Exit(1)
	}

	if len(romData) < 32 {
		fmt.Fprintf(os.Stderr, "ROM too small\n")
		os.Exit(1)
	}

	// Parse header
	magic := binary.LittleEndian.Uint32(romData[0:4])
	if magic != 0x46434D52 {
		fmt.Fprintf(os.Stderr, "Invalid ROM magic\n")
		os.Exit(1)
	}

	entryBank := binary.LittleEndian.Uint16(romData[10:12])
	entryOffset := binary.LittleEndian.Uint16(romData[12:14])

	fmt.Printf("CPU Execution Test\n")
	fmt.Printf("==================\n\n")
	fmt.Printf("Entry Point: Bank %d, Offset 0x%04X\n", entryBank, entryOffset)
	fmt.Printf("Entry PC: 0x%02X:%04X\n\n", entryBank, entryOffset)

	// Simulate CPU execution
	codeStart := 32
	code := romData[codeStart:]
	romBase := uint16(0x8000)

	// Simulate PC
	pcBank := uint8(entryBank)
	pcOffset := entryOffset
	maxInstructions := 50 // Limit to prevent infinite loops

	fmt.Printf("Simulating CPU execution (max %d instructions):\n\n", maxInstructions)

	for i := 0; i < maxInstructions; i++ {
		// Calculate ROM offset
		romOffset := int(pcOffset) - int(romBase)
		if romOffset < 0 || romOffset >= len(code)-1 {
			fmt.Printf("⚠️  PC out of bounds: 0x%02X:%04X (ROM offset 0x%04X)\n", pcBank, pcOffset, romOffset)
			break
		}

		// Fetch instruction
		inst := binary.LittleEndian.Uint16(code[romOffset:])
		opcode := (inst >> 12) & 0xF
		mode := (inst >> 8) & 0xF
		reg1 := (inst >> 4) & 0xF
		reg2 := inst & 0xF

		// Display instruction
		fmt.Printf("[%d] PC: 0x%02X:%04X (ROM offset 0x%04X)\n", i+1, pcBank, pcOffset, romOffset)
		fmt.Printf("     Instruction: 0x%04X\n", inst)
		fmt.Printf("     Decoded: opcode=0x%X, mode=0x%X, reg1=%d, reg2=%d\n", opcode, mode, reg1, reg2)

		// Simulate instruction execution
		pcAfterInst := pcOffset + 2

		switch opcode {
		case 0xD: // JMP
			if romOffset+3 >= len(code) {
				fmt.Printf("     ⚠️  JMP: Not enough bytes for offset\n")
				break
			}
			offsetBytes := binary.LittleEndian.Uint16(code[romOffset+2:])
			var offset int16
			if offsetBytes > 0x7FFF {
				offset = int16(int32(offsetBytes) - 0x10000)
			} else {
				offset = int16(offsetBytes)
			}
			pcAfterOffset := pcAfterInst + 2
			newOffset := int32(pcAfterOffset) + int32(offset)
			fmt.Printf("     JMP: offset=%d (0x%04X), PC after inst=0x%04X, PC after offset=0x%04X\n", offset, uint16(offset), pcAfterInst, pcAfterOffset)
			fmt.Printf("     JMP: newOffset = 0x%04X + %d = 0x%05X\n", pcAfterOffset, offset, newOffset)
			if newOffset < 0x8000 {
				fmt.Printf("     ⚠️  ERROR: JMP target < 0x8000 (invalid!)\n")
				break
			}
			if newOffset > 0xFFFF {
				fmt.Printf("     ⚠️  WARNING: JMP target > 0xFFFF, clamping to 0xFFFF\n")
				newOffset = 0xFFFF
			}
			pcOffset = uint16(newOffset)
			pcOffset &^= 1 // Align
			fmt.Printf("     → New PC: 0x%02X:%04X\n", pcBank, pcOffset)

		case 0xC: // CMP and branches
			modeU8 := uint8(mode)
			if modeU8 >= 0x1 && modeU8 <= 0x6 { // Branch
				if romOffset+3 >= len(code) {
					fmt.Printf("     ⚠️  Branch: Not enough bytes for offset\n")
					break
				}
				branchNames := map[uint8]string{
					0x1: "BEQ",
					0x2: "BNE",
					0x3: "BGT",
					0x4: "BLT",
					0x5: "BGE",
					0x6: "BLE",
				}
				branchName := branchNames[modeU8]
				offsetBytes := binary.LittleEndian.Uint16(code[romOffset+2:])
				var offset int16
				if offsetBytes > 0x7FFF {
					offset = int16(int32(offsetBytes) - 0x10000)
				} else {
					offset = int16(offsetBytes)
				}
				pcAfterOffset := pcAfterInst + 2
				newOffset := int32(pcAfterOffset) + int32(offset)
				fmt.Printf("     %s: offset=%d (0x%04X), PC after offset=0x%04X\n", branchName, offset, uint16(offset), pcAfterOffset)
				fmt.Printf("     %s: target = 0x%04X + %d = 0x%05X\n", branchName, pcAfterOffset, offset, newOffset)
				// For simulation, assume branch is taken (worst case for infinite loop)
				if newOffset >= 0x8000 && newOffset <= 0xFFFF {
					pcOffset = uint16(newOffset)
					pcOffset &^= 1
					fmt.Printf("     → Branch taken, New PC: 0x%02X:%04X\n", pcBank, pcOffset)
				} else {
					fmt.Printf("     → Branch not taken (invalid target or simulation)\n")
					pcOffset = pcAfterOffset
				}
			} else {
				// CMP - just advance PC
				if modeU8 == 0 {
					// CMP R1, R2 - no immediate
					pcOffset = pcAfterInst
				} else {
					// CMP R1, #imm - has immediate
					pcOffset = pcAfterInst + 2
				}
			}

		case 0x1: // MOV
			if mode == 1 || mode == 2 { // Has immediate
				pcOffset = pcAfterInst + 2
			} else {
				pcOffset = pcAfterInst
			}

		default:
			// Most instructions just advance PC
			pcOffset = pcAfterInst
		}

		fmt.Println()

		// Check for infinite loop (same PC twice)
		// This is a simple check - in real execution, we'd track more state
	}

	fmt.Println("Simulation complete.")
}
