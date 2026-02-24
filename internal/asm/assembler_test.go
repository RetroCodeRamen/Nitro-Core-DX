package asm

import (
	"encoding/binary"
	"testing"

	"nitro-core-dx/internal/emulator"
)

func TestAssembleAllSupportedMnemonics(t *testing.T) {
	src := `
.entry 1, 0x8000
start:
    NOP
    MOV R0, #0x1234
    MOV R1, R0
    MOV R2, [R1]
    MOV [R1], R2
    MOV.B R3, [R1]
    MOV.B [R1], R3
    PUSH R0
    POP R4
    ADD R0, #1
    ADD R0, R1
    SUB R0, #1
    SUB R0, R1
    MUL R0, #2
    MUL R0, R1
    DIV R0, #2
    DIV R0, R1
    AND R0, #0xFF
    AND R0, R1
    OR R0, #1
    OR R0, R1
    XOR R0, #1
    XOR R0, R1
    NOT R0
    SHL R0, #1
    SHL R0, R1
    SHR R0, #1
    SHR R0, R1
    CMP R0, #0x10
    CMP R0, R1
    BEQ done
    BNE done
    BGT done
    BLT done
    BGE done
    BLE done
    CALL subr
    JMP done
subr:
    RET
done:
    RET
`
	res, err := AssembleSource(src, "all.asm", nil)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}
	if len(res.ROMBytes) <= 32 {
		t.Fatalf("expected ROM bytes > header, got %d", len(res.ROMBytes))
	}
}

func TestAssembleLabelsAndBranchOffsetsRun(t *testing.T) {
	src := `
start:
    MOV R0, #0
loop:
    ADD R0, #1
    CMP R0, #5
    BLT loop
    RET
`
	res, err := AssembleSource(src, "loop.asm", nil)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	emu := emulator.NewEmulator()
	emu.SetFrameLimit(false)
	if err := emu.LoadROM(res.ROMBytes); err != nil {
		t.Fatalf("load rom failed: %v", err)
	}
	emu.Start()
	// execute a handful of instructions manually; RET will error on empty stack, which is fine after loop completes
	for i := 0; i < 32; i++ {
		if err := emu.CPU.ExecuteInstruction(); err != nil {
			break
		}
	}
	if got := emu.CPU.State.R0; got != 5 {
		t.Fatalf("expected R0=5, got %d", got)
	}
}

func TestCMPImmediateR0EncodesDisambiguated(t *testing.T) {
	src := `CMP R0, #1`
	res, err := AssembleSource(src, "cmp.asm", nil)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}
	if len(res.ROMBytes) < 36 { // header + instruction+imm
		t.Fatalf("rom too small")
	}
	inst := binary.LittleEndian.Uint16(res.ROMBytes[32:34])
	mode := (inst >> 8) & 0xF
	reg1 := (inst >> 4) & 0xF
	reg2 := inst & 0xF
	if mode != 1 || reg1 != 0 || reg2 == 0 {
		t.Fatalf("CMP R0,#imm not disambiguated: inst=0x%04X mode=%d reg1=%d reg2=%d", inst, mode, reg1, reg2)
	}
}
