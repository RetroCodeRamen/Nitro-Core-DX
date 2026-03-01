package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: verify_bytecode <rom.rom>")
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
		fmt.Fprintf(os.Stderr, "Invalid ROM magic: 0x%08X\n", magic)
		os.Exit(1)
	}

	romSize := binary.LittleEndian.Uint32(romData[6:10])
	entryBank := binary.LittleEndian.Uint16(romData[10:12])
	entryOffset := binary.LittleEndian.Uint16(romData[12:14])

	fmt.Printf("ROM Analysis:\n")
	fmt.Printf("  Size: %d bytes\n", romSize)
	fmt.Printf("  Entry: Bank %d, Offset 0x%04X\n", entryBank, entryOffset)
	fmt.Printf("  Entry PC: 0x%02X:%04X\n", entryBank, entryOffset)
	fmt.Println()

	// Analyze code
	codeStart := 32
	codeEnd := codeStart + int(romSize)
	code := romData[codeStart:codeEnd]

	fmt.Printf("Code Analysis (ROM offset = file offset - 32):\n")
	fmt.Printf("  Code range: 0x%04X - 0x%04X (ROM offsets)\n", 0, len(code))
	fmt.Println()

	// Find all JMP instructions
	fmt.Println("JMP Instructions:")
	romBase := uint16(0x8000) // ROM code starts at 0x8000 in bank
	for i := 0; i < len(code)-3; i += 2 {
		inst := binary.LittleEndian.Uint16(code[i:])
		if (inst >> 12) == 0xD { // JMP opcode
			offsetBytes := binary.LittleEndian.Uint16(code[i+2:])
			var offset int16
			if offsetBytes > 0x7FFF {
				offset = int16(int32(offsetBytes) - 0x10000)
			} else {
				offset = int16(offsetBytes)
			}

			// Calculate what CPU will do
			romOffset := uint16(i)
			pcAtJMP := romBase + romOffset
			pcAfterInst := pcAtJMP + 2
			pcAfterOffset := pcAfterInst + 2
			targetPC := uint32(pcAfterOffset) + uint32(int32(offset))
			targetROMOffset := targetPC - uint32(romBase)

			fmt.Printf("  JMP at ROM offset 0x%04X (PC 0x%04X):\n", romOffset, pcAtJMP)
			fmt.Printf("    Instruction: 0x%04X\n", inst)
			fmt.Printf("    Offset: %d (0x%04X)\n", offset, uint16(offset))
			fmt.Printf("    PC after instruction: 0x%04X\n", pcAfterInst)
			fmt.Printf("    PC after offset: 0x%04X\n", pcAfterOffset)
			fmt.Printf("    Target PC: 0x%05X\n", targetPC)
			fmt.Printf("    Target ROM offset: 0x%04X\n", targetROMOffset)
			if targetPC < 0x8000 {
				fmt.Printf("    ⚠️  WARNING: Target < 0x8000 (invalid!)\n")
			}
			if targetROMOffset >= uint32(len(code)) {
				fmt.Printf("    ⚠️  WARNING: Target beyond code end\n")
			}
			fmt.Println()
		}
	}

	// Find all branch instructions
	fmt.Println("Branch Instructions (BEQ/BNE/BGE):")
	for i := 0; i < len(code)-3; i += 2 {
		inst := binary.LittleEndian.Uint16(code[i:])
		opcode := inst >> 12
		if opcode == 0xC {
			mode := uint8((inst >> 8) & 0xF)
			if mode >= 0x1 && mode <= 0x6 { // Branch instruction
				branchNames := map[uint8]string{
					0x1: "BEQ",
					0x2: "BNE",
					0x3: "BGT",
					0x4: "BLT",
					0x5: "BGE",
					0x6: "BLE",
				}
				branchName := branchNames[mode]
				if branchName == "" {
					continue
				}

				offsetBytes := binary.LittleEndian.Uint16(code[i+2:])
				var offset int16
				if offsetBytes > 0x7FFF {
					offset = int16(int32(offsetBytes) - 0x10000)
				} else {
					offset = int16(offsetBytes)
				}

				romOffset := uint16(i)
				pcAtBranch := romBase + romOffset
				pcAfterInst := pcAtBranch + 2
				pcAfterOffset := pcAfterInst + 2
				targetPC := uint32(pcAfterOffset) + uint32(int32(offset))
				targetROMOffset := targetPC - uint32(romBase)

				fmt.Printf("  %s at ROM offset 0x%04X (PC 0x%04X):\n", branchName, romOffset, pcAtBranch)
				fmt.Printf("    Offset: %d (0x%04X)\n", offset, uint16(offset))
				fmt.Printf("    Target PC: 0x%05X (ROM offset 0x%04X)\n", targetPC, targetROMOffset)
				if targetPC < 0x8000 {
					fmt.Printf("    ⚠️  WARNING: Target < 0x8000 (invalid!)\n")
				}
				if targetROMOffset >= uint32(len(code)) {
					fmt.Printf("    ⚠️  WARNING: Target beyond code end\n")
				}
				fmt.Println()
			}
		}
	}
}
