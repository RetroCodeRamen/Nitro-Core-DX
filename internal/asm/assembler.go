package asm

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"nitro-core-dx/internal/rom"
)

type Options struct {
	EntryBank   uint8
	EntryOffset uint16
	OutputPath  string
}

type Result struct {
	EntryBank   uint8
	EntryOffset uint16
	ROMBytes    []byte
	Words       int
	Labels      map[string]int
}

type statement struct {
	Line     int
	Label    string
	Mnemonic string
	Operands []string
	Dir      string
	Raw      string
}

type Assembler struct {
	source string
	path   string
	opts   Options

	stmts  []statement
	labels map[string]int // code word index
}

func defaultOptions() Options {
	return Options{EntryBank: 1, EntryOffset: 0x8000}
}

func AssembleFile(path string, opts *Options) (*Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return AssembleSource(string(data), path, opts)
}

func AssembleSource(source, path string, opts *Options) (*Result, error) {
	cfg := defaultOptions()
	if opts != nil {
		if opts.EntryBank != 0 {
			cfg.EntryBank = opts.EntryBank
		}
		if opts.EntryOffset != 0 {
			cfg.EntryOffset = opts.EntryOffset
		}
		cfg.OutputPath = opts.OutputPath
	}
	if path == "" {
		path = "<buffer>"
	}
	a := &Assembler{source: source, path: path, opts: cfg, labels: make(map[string]int)}
	if err := a.parse(); err != nil {
		return nil, err
	}
	if err := a.firstPass(); err != nil {
		return nil, err
	}
	res, err := a.secondPass()
	if err != nil {
		return nil, err
	}
	if cfg.OutputPath != "" {
		if err := os.WriteFile(cfg.OutputPath, res.ROMBytes, 0o644); err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (a *Assembler) parse() error {
	s := bufio.NewScanner(strings.NewReader(a.source))
	lineNo := 0
	for s.Scan() {
		lineNo++
		raw := s.Text()
		line := stripComment(raw)
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		st := statement{Line: lineNo, Raw: raw}
		// label-only or label+instruction
		if idx := strings.Index(line, ":"); idx >= 0 {
			left := strings.TrimSpace(line[:idx])
			if left != "" && isIdent(left) {
				st.Label = strings.ToUpper(left)
				line = strings.TrimSpace(line[idx+1:])
			}
		}
		if line == "" {
			a.stmts = append(a.stmts, st)
			continue
		}
		if strings.HasPrefix(line, ".") {
			parts := splitFieldsPreserveRemainder(line)
			st.Dir = strings.ToLower(parts[0])
			if len(parts) > 1 {
				st.Operands = splitOperands(parts[1])
			}
			a.stmts = append(a.stmts, st)
			continue
		}
		parts := splitFieldsPreserveRemainder(line)
		st.Mnemonic = strings.ToUpper(parts[0])
		if len(parts) > 1 {
			st.Operands = splitOperands(parts[1])
		}
		a.stmts = append(a.stmts, st)
	}
	if err := s.Err(); err != nil {
		return err
	}
	return nil
}

func (a *Assembler) firstPass() error {
	pcWords := 0
	for _, st := range a.stmts {
		if st.Label != "" {
			if _, exists := a.labels[st.Label]; exists {
				return a.errf(st.Line, "duplicate label: %s", st.Label)
			}
			a.labels[st.Label] = pcWords
		}
		w, err := a.statementWords(st)
		if err != nil {
			return err
		}
		pcWords += w
	}
	return nil
}

func (a *Assembler) secondPass() (*Result, error) {
	b := rom.NewROMBuilder()
	for _, st := range a.stmts {
		if st.Dir != "" {
			if err := a.emitDirective(b, st); err != nil {
				return nil, err
			}
			continue
		}
		if st.Mnemonic == "" {
			continue
		}
		if err := a.emitInstruction(b, st); err != nil {
			return nil, err
		}
	}
	romBytes, err := b.BuildROMBytes(a.opts.EntryBank, a.opts.EntryOffset)
	if err != nil {
		return nil, err
	}
	labelsCopy := make(map[string]int, len(a.labels))
	for k, v := range a.labels {
		labelsCopy[k] = v
	}
	return &Result{EntryBank: a.opts.EntryBank, EntryOffset: a.opts.EntryOffset, ROMBytes: romBytes, Words: b.GetCodeLength(), Labels: labelsCopy}, nil
}

func (a *Assembler) statementWords(st statement) (int, error) {
	if st.Dir == "" && st.Mnemonic == "" {
		return 0, nil
	}
	if st.Dir != "" {
		switch st.Dir {
		case ".entry":
			return 0, nil
		case ".word":
			if len(st.Operands) != 1 { return 0, a.errf(st.Line, ".word requires 1 operand") }
			return 1, nil
		default:
			return 0, a.errf(st.Line, "unknown directive %s", st.Dir)
		}
	}
	m := strings.ToUpper(st.Mnemonic)
	switch m {
	case "NOP", "RET", "BEQ", "BNE", "BGT", "BLT", "BGE", "BLE", "JMP", "CALL", "PUSH", "POP", "NOT":
		// Branch/JMP/CALL take immediates, handled below
	}
	if isBranchLike(m) || m == "JMP" || m == "CALL" {
		return 2, nil
	}
	if m == "NOP" || m == "RET" || m == "PUSH" || m == "POP" || m == "NOT" {
		return 1, nil
	}
	if needsImmediateWord(st) {
		return 2, nil
	}
	return 1, nil
}

func (a *Assembler) emitDirective(b *rom.ROMBuilder, st statement) error {
	switch st.Dir {
	case ".entry":
		if len(st.Operands) != 2 {
			return a.errf(st.Line, ".entry requires 2 operands: bank, offset")
		}
		bank, err := a.eval(st.Line, st.Operands[0])
		if err != nil { return err }
		off, err := a.eval(st.Line, st.Operands[1])
		if err != nil { return err }
		a.opts.EntryBank = uint8(bank)
		a.opts.EntryOffset = uint16(off)
		return nil
	case ".word":
		v, err := a.eval(st.Line, st.Operands[0])
		if err != nil { return err }
		b.AddImmediate(uint16(v))
		return nil
	default:
		return a.errf(st.Line, "unknown directive %s", st.Dir)
	}
}

func (a *Assembler) emitInstruction(b *rom.ROMBuilder, st statement) error {
	m := strings.ToUpper(st.Mnemonic)
	ops := st.Operands

	switch m {
	case "NOP":
		b.AddInstruction(rom.EncodeNOP())
		return nil
	case "RET":
		b.AddInstruction(rom.EncodeRET())
		return nil
	case "JMP":
		return a.emitPCRel(b, st, rom.EncodeJMP())
	case "CALL":
		return a.emitPCRel(b, st, rom.EncodeCALL())
	case "BEQ":
		return a.emitPCRel(b, st, rom.EncodeBEQ())
	case "BNE":
		return a.emitPCRel(b, st, rom.EncodeBNE())
	case "BGT":
		return a.emitPCRel(b, st, rom.EncodeBGT())
	case "BLT":
		return a.emitPCRel(b, st, rom.EncodeBLT())
	case "BGE":
		return a.emitPCRel(b, st, rom.EncodeBGE())
	case "BLE":
		return a.emitPCRel(b, st, rom.EncodeBLE())
	case "PUSH":
		if len(ops) != 1 { return a.errf(st.Line, "PUSH requires 1 operand") }
		r1, err := parseReg(ops[0]); if err != nil { return a.errf(st.Line, err.Error()) }
		b.AddInstruction(encodeOpcodeModeRegs(0x1, 4, r1, 0))
		return nil
	case "POP":
		if len(ops) != 1 { return a.errf(st.Line, "POP requires 1 operand") }
		r1, err := parseReg(ops[0]); if err != nil { return a.errf(st.Line, err.Error()) }
		b.AddInstruction(encodeOpcodeModeRegs(0x1, 5, r1, 0))
		return nil
	case "NOT":
		if len(ops) != 1 { return a.errf(st.Line, "NOT requires 1 operand") }
		r1, err := parseReg(ops[0]); if err != nil { return a.errf(st.Line, err.Error()) }
		b.AddInstruction(encodeOpcodeModeRegs(0x9, 0, r1, 0))
		return nil
	}

	// MOV and MOV.B forms
	if m == "MOV" || m == "MOV.B" {
		return a.emitMOVLike(b, st, m == "MOV.B")
	}

	// ALU/CMP/shift
	opcodeMap := map[string]uint8{
		"ADD": 0x2, "SUB": 0x3, "MUL": 0x4, "DIV": 0x5,
		"AND": 0x6, "OR": 0x7, "XOR": 0x8,
		"SHL": 0xA, "SHR": 0xB,
		"CMP": 0xC,
	}
	opcode, ok := opcodeMap[m]
	if !ok {
		return a.errf(st.Line, "unknown mnemonic %s", st.Mnemonic)
	}
	if len(ops) != 2 {
		return a.errf(st.Line, "%s requires 2 operands", m)
	}
	r1, err := parseReg(ops[0])
	if err != nil { return a.errf(st.Line, err.Error()) }
	if imm, ok, err := parseImmediateOperand(ops[1], a, st.Line); err != nil {
		return err
	} else if ok {
		mode := uint8(1)
		reg2 := uint8(0)
		// CMP immediate encoding overlaps BEQ if reg1=R0 and reg2=0. Force a non-zero reg2 tag.
		if m == "CMP" {
			reg2 = 0xF
		}
		b.AddInstruction(encodeOpcodeModeRegs(opcode, mode, r1, reg2))
		b.AddImmediate(uint16(imm))
		return nil
	}
	r2, err := parseReg(ops[1])
	if err != nil { return a.errf(st.Line, err.Error()) }
	b.AddInstruction(encodeOpcodeModeRegs(opcode, 0, r1, r2))
	return nil
}

func (a *Assembler) emitMOVLike(b *rom.ROMBuilder, st statement, byteMode bool) error {
	ops := st.Operands
	if len(ops) != 2 {
		return a.errf(st.Line, "%s requires 2 operands", st.Mnemonic)
	}
	left, right := strings.TrimSpace(ops[0]), strings.TrimSpace(ops[1])

	// PUSH/POP are separate mnemonics; here support all MOV modes explicitly.
	if isMemRef(left) {
		r1, err := parseMemReg(left)
		if err != nil { return a.errf(st.Line, err.Error()) }
		r2, err := parseReg(right)
		if err != nil { return a.errf(st.Line, err.Error()) }
		mode := uint8(3)
		if byteMode { mode = 7 }
		b.AddInstruction(rom.EncodeMOV(mode, r1, r2))
		return nil
	}
	// left is register target
	r1, err := parseReg(left)
	if err != nil { return a.errf(st.Line, err.Error()) }
	if imm, ok, err := parseImmediateOperand(right, a, st.Line); err != nil {
		return err
	} else if ok {
		if byteMode {
			return a.errf(st.Line, "MOV.B does not support immediate form")
		}
		b.AddInstruction(rom.EncodeMOV(1, r1, 0))
		b.AddImmediate(uint16(imm))
		return nil
	}
	if isMemRef(right) {
		r2, err := parseMemReg(right)
		if err != nil { return a.errf(st.Line, err.Error()) }
		mode := uint8(2)
		if byteMode { mode = 6 }
		b.AddInstruction(rom.EncodeMOV(mode, r1, r2))
		return nil
	}
	r2, err := parseReg(right)
	if err != nil { return a.errf(st.Line, err.Error()) }
	if byteMode {
		return a.errf(st.Line, "MOV.B register-to-register form is not a valid CPU mode")
	}
	b.AddInstruction(rom.EncodeMOV(0, r1, r2))
	return nil
}

func (a *Assembler) emitPCRel(b *rom.ROMBuilder, st statement, op uint16) error {
	if len(st.Operands) != 1 {
		return a.errf(st.Line, "%s requires 1 operand", st.Mnemonic)
	}
	b.AddInstruction(op)
	offsetWordIndex := b.GetCodeLength()
	targetExpr := st.Operands[0]
	if target, ok := a.labels[strings.ToUpper(targetExpr)]; ok {
		off := rom.CalculateBranchOffset(uint16(offsetWordIndex*2), uint16(target*2))
		b.AddImmediate(uint16(off))
		return nil
	}
	// Allow explicit numeric relative offset (raw immediate)
	v, err := a.eval(st.Line, targetExpr)
	if err != nil {
		return err
	}
	b.AddImmediate(uint16(v))
	return nil
}

func (a *Assembler) eval(line int, expr string) (int64, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return 0, a.errf(line, "empty expression")
	}
	if v, ok := a.labels[strings.ToUpper(expr)]; ok {
		return int64(v * 2), nil
	}
	if strings.HasPrefix(expr, "#") {
		expr = strings.TrimSpace(expr[1:])
	}
	if strings.HasPrefix(expr, "$") {
		expr = "0x" + expr[1:]
	}
	if strings.HasPrefix(expr, "0b") || strings.HasPrefix(expr, "0B") {
		v, err := strconv.ParseInt(expr[2:], 2, 64)
		if err != nil { return 0, a.errf(line, "invalid binary literal %q", expr) }
		return v, nil
	}
	v, err := strconv.ParseInt(expr, 0, 64)
	if err != nil {
		return 0, a.errf(line, "invalid number or unknown label %q", expr)
	}
	return v, nil
}

func parseImmediateOperand(op string, a *Assembler, line int) (int64, bool, error) {
	op = strings.TrimSpace(op)
	if !strings.HasPrefix(op, "#") {
		return 0, false, nil
	}
	v, err := a.eval(line, op)
	if err != nil {
		return 0, true, err
	}
	return v, true, nil
}

func parseReg(s string) (uint8, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if len(s) != 2 || s[0] != 'R' || s[1] < '0' || s[1] > '7' {
		return 0, fmt.Errorf("expected register R0-R7, got %q", s)
	}
	return uint8(s[1] - '0'), nil
}

func isMemRef(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]")
}

func parseMemReg(s string) (uint8, error) {
	if !isMemRef(s) { return 0, fmt.Errorf("expected memory operand [Rn], got %q", s) }
	inner := strings.TrimSpace(s[1 : len(s)-1])
	return parseReg(inner)
}

func encodeOpcodeModeRegs(opcode, mode, reg1, reg2 uint8) uint16 {
	return (uint16(opcode) << 12) | (uint16(mode) << 8) | (uint16(reg1) << 4) | uint16(reg2)
}

func (a *Assembler) errf(line int, f string, args ...any) error {
	msg := fmt.Sprintf(f, args...)
	return fmt.Errorf("%s:%d: %s", filepath.Base(a.path), line, msg)
}

func stripComment(line string) string {
	cut := len(line)
	if i := strings.Index(line, ";"); i >= 0 && i < cut { cut = i }
	if i := strings.Index(line, "--"); i >= 0 && i < cut { cut = i }
	return line[:cut]
}

func splitOperands(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" { out = append(out, p) }
	}
	return out
}

func splitFieldsPreserveRemainder(line string) []string {
	line = strings.TrimSpace(line)
	if line == "" { return nil }
	for i, r := range line {
		if r == ' ' || r == '\t' {
			return []string{line[:i], strings.TrimSpace(line[i+1:])}
		}
	}
	return []string{line}
}

func isIdent(s string) bool {
	if s == "" { return false }
	for i, r := range s {
		if i == 0 {
			if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '_' || r == '.') {
				return false
			}
		} else {
			if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '.') {
				return false
			}
		}
	}
	return true
}

func isBranchLike(m string) bool {
	switch strings.ToUpper(m) {
	case "BEQ", "BNE", "BGT", "BLT", "BGE", "BLE":
		return true
	default:
		return false
	}
}

func needsImmediateWord(st statement) bool {
	m := strings.ToUpper(st.Mnemonic)
	ops := st.Operands
	if len(ops) == 0 { return false }
	switch m {
	case "MOV":
		return len(ops) == 2 && strings.HasPrefix(strings.TrimSpace(ops[1]), "#")
	case "MOV.B":
		return false
	case "ADD", "SUB", "MUL", "DIV", "AND", "OR", "XOR", "CMP", "SHL", "SHR":
		return len(ops) == 2 && strings.HasPrefix(strings.TrimSpace(ops[1]), "#")
	default:
		return false
	}
}
