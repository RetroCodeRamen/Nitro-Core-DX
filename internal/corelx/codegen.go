package corelx

import (
	"errors"
	"fmt"
	"strings"

	"nitro-core-dx/internal/rom"
)

// codeSink is the minimal interface codegen needs from its instruction
// stream. *rom.ROMBuilder satisfies it directly (used for pass 1/compact
// output and pass 2/measurement, both single-bank). *bankCursor wraps
// *rom.BankedROMBuilder for pass 3/final multi-bank output, forwarding each
// call with the generator's current bank injected automatically -- every
// one of the ~2000 AddInstruction/AddImmediate/GetCodeLength/SetImmediateAt
// call sites throughout this file stays untouched, since intra-function
// control flow (if/for/while, bounds checks) never spans a bank boundary
// and is always resolved relative to "wherever we currently are." Only
// cross-function call patching needs bank-explicit handling -- see
// setPatchImmediate.
type codeSink interface {
	AddInstruction(instruction uint16)
	AddImmediate(value uint16)
	GetCodeLength() int
	SetImmediateAt(wordIndex int, value uint16)
}

// bankCursor adapts *rom.BankedROMBuilder to codeSink, always operating on
// cg.currentBank -- the single bank codegen is actively emitting into at
// the moment of each call. Bank transitions happen between functions, and
// also mid-function when a function continues into the next ROM bank (see
// maybeSpill). Cross-bank branches inside that function go through a
// same-bank trampoline; see patchPos.
type bankCursor struct {
	b  *rom.BankedROMBuilder
	cg *CodeGenerator
}

func (c *bankCursor) AddInstruction(instruction uint16) {
	c.b.AddInstruction(c.cg.currentBank, instruction)
}
func (c *bankCursor) AddImmediate(value uint16) {
	c.b.AddImmediate(c.cg.currentBank, value)
}
func (c *bankCursor) GetCodeLength() int {
	return c.b.GetCodeLength(c.cg.currentBank)
}
func (c *bankCursor) SetImmediateAt(wordIndex int, value uint16) {
	c.b.SetImmediateAt(c.cg.currentBank, wordIndex, value)
}

// funcAddr is a function's start position: which bank it landed in, plus
// its bank-local word index (matching what GetCodeLength/PC report for that
// bank). In compact/measurement mode (single implicit bank), bank is always 1.
type funcAddr struct {
	bank  uint8
	index int
}

// CodeGenerator generates Nitro Core DX machine code from AST
type CodeGenerator struct {
	program          *Program
	builder          codeSink
	symbols          map[string]*Symbol
	assets           map[string]*AssetDecl
	assetIDs         map[string]uint16
	normalizedAssets map[string]AssetIR
	assetOffsets     map[string]uint16

	// Variable storage tracking
	variables   map[string]*VariableInfo
	varCounter  int
	stackOffset uint16 // Current stack offset for spilled variables

	// Top-level constants and WRAM globals (charter D3).
	consts         map[string]int64
	constFixed     map[string]bool
	imageAssets    map[string]*ImageAsset
	musicAssets    map[string]*MusicAsset
	globals        map[string]*VariableInfo
	memoryMap      []MemoryMapEntry
	structLayouts  map[string]*structLayout // lazily built by structLayoutFor
	layoutVisiting map[string]bool
	layoutErr      error
	varScopes      []varScope

	// Arithmetic helper routines emitted once after user functions when
	// referenced (signed 8.8 multiply needs 32-bit-correct partials).
	needFixmul  bool
	needDivs    bool
	needFixdiv  bool
	needDrawInt bool

	// exprSpill is how many expression temps are live in this function's
	// frame. spillBase is that frame's slot 0. Each function has its own
	// block, so a callee that spills does not overwrite the caller.
	exprSpill int
	spillBase uint16

	// Function call support
	functionAddrs map[string]funcAddr // function name -> (bank, code word index) of function start
	callPatches   []callPatch         // pending CALL offset patches
	globalStack   uint16              // tracks per-function stack base to avoid overlap

	// IRQ/NMI vector fix-up: word positions of the placeholder bank/offset
	// bytes written by emitIRQVectorFix, patched once __irqstub's real
	// (bank, address) is known (see patchIRQVector).
	irqVectorPatchPositions []irqVectorPatch

	// Innermost-enclosing-loop stack, for break/continue target patching.
	loopStack []*loopContext

	// Multi-bank support (see internal/rom/banked_builder.go and the
	// codeSink/bankCursor types above). currentBank is which bank cg.builder
	// (when it's a *bankCursor) is actively emitting into. It changes between
	// functions, and mid-function when maybeSpill continues a body that
	// does not fit in one ROM bank.
	// wideCallMode, when set, makes every function/helper call emit the
	// 5-word far-call form (see emitCallPatch) even for same-bank targets,
	// so call-site sizes are identical across the measurement pass and the
	// final emission pass that uses its bank schedule. bankedBuilder is the
	// concrete builder backing cg.builder in pass 3 (nil otherwise), needed
	// because cross-function patches resolve after cg.currentBank has moved
	// on and must target an explicit bank rather than "wherever we are now"
	// (see setPatchImmediate). bankSchedule maps function name -> assigned
	// bank, computed by pass 2 and consulted by pass 3's emission loop.
	currentBank   uint8
	wideCallMode  bool
	bankedBuilder *rom.BankedROMBuilder
	bankSchedule  map[string]uint8
	fnBankEnds    map[string]uint8
	fnBankEnd     uint8
	currentFn     string
	openBranches  []codePos

	// emitOrder records every function/helper name in the exact order
	// recordFuncAddr saw it emitted. A pass-2 measurement build's emitOrder
	// plus functionAddrs is the sole input to packFunctionBanks -- sizes
	// are derived from consecutive word-index deltas, so this order must
	// exactly match what pass 3 will later emit (guaranteed, since both
	// passes run the same codegen logic over the same AST).
	emitOrder []string
}

// callPatch records a pending CALL that needs its offset patched once the
// target function address is known. In compact mode (wide == false) this is
// exactly today's single relative-offset patch. In wide mode, two immediate
// words are patched: the bank byte at bankPos and the absolute bank-local
// offset at offsetPos -- both resolved against the patch's own recorded
// bank (the bank the call site itself lives in), not cg.currentBank, since
// resolution happens after emission has moved on to later functions/banks.
type callPatch struct {
	offsetPos int    // word index of the offset immediate
	bankPos   int    // word index of the bank immediate (wide mode only)
	bank      uint8  // bank the call site itself was emitted into
	target    string // target function name
	wide      bool   // true if this patch is the 5-word far-call form
}

// loopContext tracks pending break/continue jumps for the innermost
// enclosing while/for loop during codegen. Both are backpatched once their
// target is known, the same way loop-exit branches already are: break to the
// loop's exit point, continue to its "next iteration" point (condition
// re-check for while; the increment step for for, so the loop variable still
// advances).
type loopContext struct {
	breakPatches    []codePos
	continuePatches []codePos
}

// codePos is a bank-local instruction-stream position. Word indexes restart
// at 0 in each ROM bank, so a branch site and its target are only comparable
// together with the bank they were emitted in.
type codePos struct {
	bank uint8
	word int
}

// farJumpWords is the MOV-bank / MOV-offset / JMP-far sequence used to leave
// the current ROM bank. bankSpillSlackWords is the free space pass 3 keeps
// at the end of a bank before continuing a function in the next one. It has
// to cover the largest single statement that is not split (a 64×64
// gfx.load_tiles expands to roughly 6K words) so that statement is only
// started when the rest of the bank can hold it.
const (
	farJumpWords        = 5
	bankSpillSlackWords = 8192
)

const (
	// Top of WRAM stack region used by CPU Push16/Pop16.
	stackTopAddr = uint16(0x1FFF)
	// Reserve top 256 bytes for CALL/RET return stack frames.
	callStackReserveBytes = uint16(0x0100)
	// Compiler-managed locals/params must stay above this floor.
	stackMinAddr = uint16(0x0100)

	// WRAM global region (charter memory model):
	// 0x2000-0x20FF reserved runtime block (compiler/runtime internal state),
	// 0x2100+       auto-allocated globals,
	// 0x7000-0x7FFF user scratch (never compiler-allocated; mem.* safe zone).
	runtimeBlockBase = uint16(0x2000)
	globalsBase      = uint16(0x2100)
	globalsLimit     = uint16(0x6FFF)
	userScratchBase  = uint16(0x7000)
	userScratchTop   = uint16(0x7FFF)
)

// VariableInfo tracks where a variable is stored
type VariableInfo struct {
	Name       string
	Location   VariableLocation
	StackAddr  uint16 // WRAM address of the slot, or of the value itself when Direct
	StructType string // struct name when this value is a struct or an array of structs
	ElemWidth  uint8  // element width in bytes for arrays (1, 2, or a struct size up to 255)
	ArrayLen   int    // 0 = scalar; N>0 = fixed-size array
	VarType    string // declared/inferred storage type name ("int","u8","u16","fixed"); "" = unknown
	Direct     bool   // struct bytes live at StackAddr (a global or a local array)
	ArrayRef   bool   // StackAddr holds the base address of an array parameter
}

// VariableLocation indicates where a variable is stored. Every live
// variable is a WRAM slot; the old register-resident location was never
// assigned.
type VariableLocation int

const (
	VarLocationStack VariableLocation = iota
)

type structMemberInfo struct {
	Offset uint16
	Width  uint8  // 1 or 2 for a scalar field; 0 when the field is a nested struct
	Nested string // nested struct type, empty for a scalar
}

type varScope struct {
	saved map[string]*VariableInfo
	had   map[string]bool
}

// structLayout describes a struct type's field offsets/widths and total size.
type structLayout struct {
	Fields map[string]structMemberInfo
	Size   uint16
}

// structLayoutFor returns the field layout for a struct type name, computing
// and caching it on first use. "Sprite" is a compiler intrinsic (the OAM
// sprite-attribute hardware format) with no source-level declaration; every
// other name is resolved generically from a `type Name = struct` declaration
// in the program.
func (cg *CodeGenerator) structLayoutFor(name string) (*structLayout, bool) {
	if cg.structLayouts == nil {
		cg.structLayouts = map[string]*structLayout{
			"Sprite": {
				Size: 6,
				Fields: map[string]structMemberInfo{
					"x_lo": {Offset: 0, Width: 1},
					"x_hi": {Offset: 1, Width: 1},
					"y":    {Offset: 2, Width: 1},
					"tile": {Offset: 3, Width: 1},
					"attr": {Offset: 4, Width: 1},
					"ctrl": {Offset: 5, Width: 1},
				},
			},
		}
	}
	if layout, ok := cg.structLayouts[name]; ok {
		return layout, true
	}
	for _, td := range cg.program.Types {
		if td.Name != name {
			continue
		}
		st, ok := td.Type.(*StructType)
		if !ok {
			return nil, false
		}
		if cg.layoutVisiting == nil {
			cg.layoutVisiting = map[string]bool{}
		}
		if cg.layoutVisiting[name] {
			cg.layoutErr = fmt.Errorf("struct %s refers to itself", name)
			return nil, false
		}
		cg.layoutVisiting[name] = true
		defer delete(cg.layoutVisiting, name)
		layout := &structLayout{Fields: map[string]structMemberInfo{}}
		offset := uint16(0)
		for _, f := range st.Fields {
			info, size, err := cg.fieldInfo(f.Type)
			if err != nil {
				cg.layoutErr = err
				return nil, false
			}
			info.Offset = offset
			layout.Fields[f.Name] = info
			offset += size
		}
		if offset == 0 || offset%2 != 0 {
			cg.layoutErr = fmt.Errorf("struct %s must have a non-zero even size (got %d bytes)", name, offset)
			return nil, false
		}
		layout.Size = offset
		cg.structLayouts[name] = layout
		return layout, true
	}
	return nil, false
}

// fieldInfo describes one struct field. A nested struct is stored inline.
func (cg *CodeGenerator) fieldInfo(t TypeExpr) (structMemberInfo, uint16, error) {
	if _, ok := t.(*ArrayType); ok {
		return structMemberInfo{}, 0, fmt.Errorf("a struct field cannot be an array")
	}
	named, ok := t.(*NamedType)
	if !ok || named == nil {
		return structMemberInfo{}, 0, fmt.Errorf("unsupported struct field type")
	}
	switch named.Name {
	case "u8", "i8":
		return structMemberInfo{Width: 1}, 1, nil
	case "int", "i16", "u16", "fixed":
		return structMemberInfo{Width: 2}, 2, nil
	default:
		layout, ok := cg.structLayoutFor(named.Name)
		if !ok {
			if cg.layoutErr != nil {
				return structMemberInfo{}, 0, cg.layoutErr
			}
			return structMemberInfo{}, 0, fmt.Errorf("unsupported struct field type %q", named.Name)
		}
		return structMemberInfo{Width: 0, Nested: named.Name}, layout.Size, nil
	}
}

// elemSize is the byte width of one value of typeName. The second result is
// the struct name when typeName is a struct, otherwise "".
func (cg *CodeGenerator) elemSize(typeName string) (uint16, string, error) {
	switch typeName {
	case "u8", "i8":
		return 1, "", nil
	case "int", "i16", "u16", "fixed":
		return 2, "", nil
	default:
		layout, ok := cg.structLayoutFor(typeName)
		if !ok {
			if cg.layoutErr != nil {
				return 0, "", cg.layoutErr
			}
			return 0, "", fmt.Errorf("unsupported type %q (supported: int, u8, u16, fixed, and struct types)", typeName)
		}
		if layout.Size > 255 {
			return 0, "", fmt.Errorf("struct %s is %d bytes; an element cannot exceed 255", typeName, layout.Size)
		}
		return layout.Size, typeName, nil
	}
}

func (cg *CodeGenerator) structTypeNameFromTypeExpr(t TypeExpr) string {
	switch tt := t.(type) {
	case *NamedType:
		if _, ok := cg.structLayoutFor(tt.Name); ok {
			return tt.Name
		}
	}
	return ""
}

var errUnknownBuiltin = errors.New("unknown builtin")

// errCodeOverflowsBank signals that a compact-mode (non-wide-call) Generate
// call produced more code than fits in one ROM bank -- a normal "pass 1
// doesn't fit, fall back to multi-bank" condition (see compileMultiBank in
// compiler.go), not a real compile error. Checked via errors.Is rather than
// surfaced as a diagnostic directly.
var errCodeOverflowsBank = errors.New("code exceeds one ROM bank")

// NewCodeGenerator creates a new code generator. builder may be a
// *rom.ROMBuilder (compact pass 1, or flat wide-call measurement pass 2) or
// left nil and assigned afterward as a *bankCursor wrapping a
// *rom.BankedROMBuilder (pass 3 final multi-bank emission) -- see
// SetBankedBuilder.
func NewCodeGenerator(program *Program, builder codeSink) *CodeGenerator {
	return &CodeGenerator{
		program:          program,
		builder:          builder,
		symbols:          make(map[string]*Symbol),
		assets:           make(map[string]*AssetDecl),
		assetIDs:         make(map[string]uint16),
		normalizedAssets: make(map[string]AssetIR),
		assetOffsets:     make(map[string]uint16),
		variables:        make(map[string]*VariableInfo),
		varCounter:       0,
		stackOffset:      stackTopAddr - callStackReserveBytes, // Below CALL stack reserve
		functionAddrs:    make(map[string]funcAddr),
		callPatches:      nil,
		globalStack:      stackTopAddr - callStackReserveBytes, // Reserve top bytes for CALL/RET stack
		consts:           make(map[string]int64),
		constFixed:       make(map[string]bool),
		globals:          make(map[string]*VariableInfo),
		currentBank:      1,
	}
}

// EnableWideCallMode makes every function/helper call emit the 5-word
// far-call form (see emitCallPatch), even for same-bank targets. Used by
// pass 2 (size measurement) and pass 3 (final multi-bank emission) so
// call-site sizes match between the pass that computes the bank schedule
// and the pass that emits against it.
func (cg *CodeGenerator) EnableWideCallMode() {
	cg.wideCallMode = true
}

// SetBankedBuilder switches cg into pass-3 final multi-bank emission:
// bankedBuilder lets cross-function patches (which resolve after
// cg.currentBank has moved on to later functions) target their own
// recorded bank explicitly; schedule is pass 2's function -> bank
// assignment, consulted by the top-level emission loop to move
// cg.currentBank between functions. Callers must also point cg.builder at
// a *bankCursor wrapping the same *rom.BankedROMBuilder.
func (cg *CodeGenerator) SetBankedBuilder(b *rom.BankedROMBuilder, schedule map[string]uint8) {
	cg.bankedBuilder = b
	cg.bankSchedule = schedule
}

// SetFunctionBankEnds records the inclusive last ROM bank reserved for each
// function that does not fit in a single bank. Pass 3 may continue those
// functions forward up to that bank and no further.
func (cg *CodeGenerator) SetFunctionBankEnds(ends map[string]uint8) {
	cg.fnBankEnds = ends
}

// MemoryMapEntry records one allocated WRAM symbol for the build's memory
// map listing (name -> address), emitted alongside the ROM for the debugger.
type MemoryMapEntry struct {
	Name    string
	Address uint16
	Size    uint16
	Kind    string // "global" | "global(pin)" | "runtime"
}

// MemoryMap returns the WRAM allocation listing for this build.
func (cg *CodeGenerator) MemoryMap() []MemoryMapEntry { return cg.memoryMap }

// allocateGlobals folds constants, assigns WRAM addresses to globals
// (auto-allocated from globalsBase; 'at' pins validated for overlap), and
// records the memory map.
func (cg *CodeGenerator) allocateGlobals() error {
	consts, constFixed, err := foldProgramConstsTyped(cg.program)
	if err != nil {
		return err
	}
	cg.consts = consts
	cg.constFixed = constFixed
	// Predefined button-mask constants for the input builtins (bit position
	// per internal/input button layout: UP=0..Z=11).
	for name, mask := range buttonMasks {
		if _, used := cg.consts[name]; !used {
			cg.consts[name] = mask
		}
	}

	cg.memoryMap = append(cg.memoryMap, MemoryMapEntry{
		Name: "__runtime", Address: runtimeBlockBase, Size: globalsBase - runtimeBlockBase, Kind: "runtime",
	})

	type span struct {
		lo, hi uint16
		name   string
	}
	pinned := []span{}
	cursor := globalsBase

	for _, g := range cg.program.Globals {
		if _, dup := cg.globals[g.Name]; dup {
			return fmt.Errorf("line %d: duplicate global %q", g.Position.Line, g.Name)
		}
		if _, isConst := cg.consts[g.Name]; isConst {
			return fmt.Errorf("line %d: global %q collides with a constant of the same name", g.Position.Line, g.Name)
		}
		if g.ArrayLen == 0 && g.ArrayLenName != "" {
			v, ok := cg.consts[g.ArrayLenName]
			if !ok || v <= 0 || v > 0x4000 {
				return fmt.Errorf("line %d: array length %s must be a constant between 1 and 16384", g.Position.Line, g.ArrayLenName)
			}
			g.ArrayLen = int(v)
		}
		elemSize, structName, err := cg.elemSize(g.TypeName)
		if err != nil {
			return fmt.Errorf("line %d: global %s: %w", g.Position.Line, g.Name, err)
		}
		if structName != "" && (g.Init != nil || len(g.InitList) > 0) {
			return fmt.Errorf("line %d: struct global %s starts at zero; assign its fields", g.Position.Line, g.Name)
		}
		size := elemSize
		if g.ArrayLen > 0 {
			size = elemSize * uint16(g.ArrayLen)
			if g.Init != nil {
				return fmt.Errorf("line %d: global array %s cannot take a scalar initializer (use [v0, v1, ...] or leave it zero)", g.Position.Line, g.Name)
			}
			if len(g.InitList) > 0 && len(g.InitList) != g.ArrayLen {
				return fmt.Errorf("line %d: global array %s has %d initializers but length %d", g.Position.Line, g.Name, len(g.InitList), g.ArrayLen)
			}
		} else if len(g.InitList) > 0 {
			return fmt.Errorf("line %d: global %s is not an array but has a list initializer", g.Position.Line, g.Name)
		}

		var addr uint16
		kind := "global"
		if g.HasPin {
			kind = "global(pin)"
			addr = g.PinAddr
			lo, hi := addr, addr+size-1
			if hi >= 0x8000 {
				return fmt.Errorf("line %d: global %s pinned at 0x%04X overlaps the I/O region (>= 0x8000)", g.Position.Line, g.Name, addr)
			}
			if lo >= runtimeBlockBase && lo < globalsBase {
				return fmt.Errorf("line %d: global %s pinned at 0x%04X overlaps the reserved runtime block (0x%04X-0x%04X)", g.Position.Line, g.Name, addr, runtimeBlockBase, globalsBase-1)
			}
			for _, p := range pinned {
				if lo <= p.hi && hi >= p.lo {
					return fmt.Errorf("line %d: global %s pinned at 0x%04X overlaps pinned global %s", g.Position.Line, g.Name, addr, p.name)
				}
			}
			pinned = append(pinned, span{lo, hi, g.Name})
		} else {
			if (size == 2 || structName != "") && cursor%2 != 0 {
				cursor++
			}
			addr = cursor
			if uint32(addr)+uint32(size)-1 > uint32(globalsLimit) {
				return fmt.Errorf("global WRAM region exhausted allocating %s (cursor 0x%04X, limit 0x%04X)", g.Name, addr, globalsLimit)
			}
			cursor += size
		}

		cg.globals[g.Name] = &VariableInfo{
			Name:       g.Name,
			Location:   VarLocationStack, // absolute WRAM addressing, same paths as stack slots
			StackAddr:  addr,
			ElemWidth:  uint8(elemSize),
			ArrayLen:   g.ArrayLen,
			VarType:    g.TypeName,
			StructType: structName,
			Direct:     structName != "",
		}
		cg.memoryMap = append(cg.memoryMap, MemoryMapEntry{Name: g.Name, Address: addr, Size: size, Kind: kind})
	}

	// Auto-allocated globals must not creep into pinned spans.
	for _, p := range pinned {
		if p.lo >= globalsBase && p.lo < cursor {
			return fmt.Errorf("pinned global %s at 0x%04X overlaps the auto-allocated global region (0x%04X-0x%04X); pin it outside that range", p.name, p.lo, globalsBase, cursor-1)
		}
	}
	return nil
}

// emitGlobalInits stores each global's initializer at program entry.
func (cg *CodeGenerator) emitGlobalInits() error {
	for _, g := range cg.program.Globals {
		info := cg.globals[g.Name]
		// Array list initializer: write each element to its slot.
		if len(g.InitList) > 0 {
			for i, el := range g.InitList {
				if v, err := evalConstExpr(el, cg.consts); err == nil {
					cg.builder.AddInstruction(rom.EncodeMOV(1, 0, 0))
					cg.builder.AddImmediate(uint16(v))
				} else if err := cg.generateExpr(el, 0); err != nil {
					return fmt.Errorf("global %s[%d] initializer: %w", g.Name, i, err)
				}
				addr := info.StackAddr + uint16(i)*uint16(info.ElemWidth)
				cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
				cg.builder.AddImmediate(addr)
				if info.ElemWidth == 2 {
					cg.builder.AddInstruction(rom.EncodeMOV(3, 7, 0)) // 16-bit store
				} else {
					cg.builder.AddInstruction(rom.EncodeMOV(7, 7, 0)) // 8-bit store
				}
			}
			continue
		}
		if g.Init == nil {
			continue
		}
		if err := cg.checkInitType(g.TypeName, g.Init); err != nil {
			return withCodePos(g.Position, err)
		}
		if v, err := evalConstExpr(g.Init, cg.consts); err == nil {
			cg.builder.AddInstruction(rom.EncodeMOV(1, 0, 0)) // MOV R0, #value
			cg.builder.AddImmediate(uint16(v))
		} else {
			if err := cg.generateExpr(g.Init, 0); err != nil {
				return withCodePos(g.Position, fmt.Errorf("global %s initializer: %w", g.Name, err))
			}
		}
		cg.storeGlobalScalar(info, 0)
	}
	return nil
}

func (cg *CodeGenerator) checkInitType(typeName string, value Expr) error {
	declared := arithTypeForName(typeName)
	got := cg.typeOf(value)
	if (declared == typeFixed && got == typeInt) || (declared == typeInt && got == typeFixed) {
		return fmt.Errorf("cannot initialize %s with a %s value — convert explicitly with int(x) or fixed(x)", typeName, got)
	}
	return nil
}

func (cg *CodeGenerator) storeGlobalScalar(info *VariableInfo, src uint8) {
	cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
	cg.builder.AddImmediate(info.StackAddr)
	if info.ElemWidth == 1 {
		cg.builder.AddInstruction(rom.EncodeMOV(7, 7, src))
	} else {
		cg.builder.AddInstruction(rom.EncodeMOV(3, 7, src))
	}
}

// SetImageAssets injects parsed external image assets (with ROM bank/offset).
func (cg *CodeGenerator) SetImageAssets(a map[string]*ImageAsset) { cg.imageAssets = a }

// SetMusicAssets injects parsed external music assets (with ROM bank/offset).
// Stored for the future music.* playback builtins; not consumed yet.
func (cg *CodeGenerator) SetMusicAssets(a map[string]*MusicAsset) { cg.musicAssets = a }

// SetNormalizedAssets injects compiler-normalized assets so codegen can avoid re-parsing source asset text.
func (cg *CodeGenerator) SetNormalizedAssets(assets []AssetIR) {
	for _, a := range assets {
		cg.normalizedAssets[a.Name] = a
	}
}

// Generate generates code for the program
// Generate emits code for the whole program. It's a thin wrapper around
// generateBody that additionally catches rom.CalculateBranchOffset's panic
// (see that function) and converts it to errCodeOverflowsBank when running
// in compact mode (!wideCallMode).
//
// The explicit bank-size checks elsewhere in generateBody (before the
// CALL-patch loop, and before starting each function) catch most compact-
// mode overflows early and cheaply, without ever reaching a panicking branch
// calculation -- but neither is exhaustive: code size can cross the one-
// bank threshold in the middle of a single large function's own body, where
// an in-function if/else/for/while branch (not a CALL patch) is the one
// that ends up computing an out-of-range offset. Every such branch uses
// CalculateBranchOffset directly with no size check of its own (by design --
// checking on every single branch site would be needlessly expensive for
// the overwhelmingly common in-bank case). This recover is the backstop:
// regardless of which specific call site panics, an out-of-range branch
// offset during compact-mode generation always means the same thing pass
// 1's other overflow checks already mean ("this doesn't fit in one bank,
// fall through to multi-bank"), so it's handled identically. Wide-call mode
// (pass 3) is never expected to hit this at all -- each function's body is
// always emitted entirely within its own bank there (BankedROMBuilder
// resets PC per bank, unlike pass 1's flat ROMBuilder), so a real panic
// during that pass indicates a genuine bug and is re-raised rather than
// silently swallowed.
func (cg *CodeGenerator) Generate() (err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		if !cg.wideCallMode {
			if msg, ok := r.(string); ok && strings.HasPrefix(msg, "branch offset out of range") {
				err = errCodeOverflowsBank
				return
			}
		}
		panic(r)
	}()
	return cg.generateBody()
}

func (cg *CodeGenerator) generateBody() error {
	// Collect assets
	for i, asset := range cg.program.Assets {
		cg.assets[asset.Name] = asset
		cg.assetIDs[asset.Name] = uint16(i)
	}

	// Register symbols
	for _, fn := range cg.program.Functions {
		cg.symbols[fn.Name] = &Symbol{
			Name:   fn.Name,
			IsFunc: true,
		}
	}

	// Generate code for each function
	// Prioritize __Boot() or Start() as the first function (entry point at 0x8000)
	functions := make([]*FunctionDecl, 0, len(cg.program.Functions))
	var entryFunction *FunctionDecl

	// Find entry point function
	for _, fn := range cg.program.Functions {
		if fn.Name == "__Boot" {
			entryFunction = fn
			break
		}
	}
	if entryFunction == nil {
		for _, fn := range cg.program.Functions {
			if fn.Name == "Start" {
				entryFunction = fn
				break
			}
		}
	}

	if err := cg.allocateGlobals(); err != nil {
		return err
	}

	// Add entry function first, then others
	if entryFunction != nil {
		functions = append(functions, entryFunction)
	}
	for _, fn := range cg.program.Functions {
		if fn != entryFunction {
			functions = append(functions, fn)
		}
	}

	// Generate code for each function, recording start addresses.
	for _, fn := range functions {
		// Same rationale as the CALL-patch bailout below, but earlier: a
		// function's own in-body control-flow branches (if/else, for/while)
		// resolve their offsets via CalculateBranchOffset too, immediately
		// during that function's own generateFunction call -- not deferred
		// to the CALL-patch loop at the end. Once compact mode's code is
		// already past one bank, ANY such branch (not just CALL patches)
		// can find its target more than +-32767 words away and hard-panic,
		// long before the end-of-Generate bank-size check below ever runs.
		// Bail out here, before generating another function's body at all,
		// so pass 1 always fails soft (falling through to multi-bank) once
		// it's overflowed, regardless of which branch flavor would have
		// been the one to panic.
		if !cg.wideCallMode && cg.builder.GetCodeLength()*2 > rom.ROMBankSizeBytes {
			return errCodeOverflowsBank
		}
		cg.recordFuncAddr(fn.Name)
		if fn == entryFunction {
			// Every real frame's VBlank fires an IRQ unconditionally (see
			// ppu/scanline.go), and the default IRQ/NMI vectors alias this
			// entry point (InitSystemVectors: bank 1, 0x8000). CoreLX has no
			// interrupt language surface -- wait_vblank() works by polling a
			// status flag, not by handling this IRQ -- so left alone, the
			// very first VBlank would silently re-enter the entry function
			// from the top (interrupts then stay masked, since nothing here
			// ever clears the CPU's I flag once the handler sets it), running
			// every statement before the first wait_vblank() call a second
			// time. Redirect both vectors to a harmless RET stub before any
			// other entry code runs.
			cg.emitIRQVectorFix()
		}
		if err := cg.generateFunction(fn); err != nil {
			return err
		}
	}

	// Emit arithmetic helper routines (referenced via CALL patches).
	if cg.needFixmul {
		cg.emitFixmulHelper()
	}
	if cg.needDivs {
		cg.emitDivsHelper()
	}
	if cg.needFixdiv {
		cg.emitFixdivHelper()
	}
	if cg.needDrawInt {
		cg.emitDrawIntHelper()
	}
	if len(cg.musicAssets) > 0 {
		cg.emitMusicAdvanceHelper()
	}
	cg.patchIRQVector()

	// Compact mode's CALL patches are same-bank relative offsets, which
	// CalculateBranchOffset hard-panics on once a call site and its target
	// end up more than +-32767 words apart -- a real possibility once the
	// program is large enough to need more than one bank in the first
	// place, and exactly the scenario the multi-bank compile driver
	// (compileMultiBank in compiler.go) exists to handle instead. Bail out
	// with a normal error before ever reaching that resolution loop, rather
	// than let it panic: pass 1 doesn't need correct patches for an attempt
	// it's about to discard anyway. Wide-call mode's own resolution
	// (below) uses absolute bank+offset addressing regardless of distance,
	// so it isn't at risk here and isn't gated by this check.
	if !cg.wideCallMode && cg.builder.GetCodeLength()*2 > rom.ROMBankSizeBytes {
		return errCodeOverflowsBank
	}

	// Patch all pending CALL offsets/targets now that every function address
	// is known. Resolution happens after cg.currentBank has moved on past
	// most call sites' original banks, so each patch is applied against its
	// own recorded bank (setPatchImmediate), never cg.currentBank.
	for _, patch := range cg.callPatches {
		target, ok := cg.functionAddrs[patch.target]
		if !ok {
			return fmt.Errorf("undefined function: %s", patch.target)
		}
		if patch.wide {
			cg.setPatchImmediate(patch.bank, patch.bankPos, uint16(target.bank))
			cg.setPatchImmediate(patch.bank, patch.offsetPos, uint16(rom.ROMBankOffsetBase+target.index*2))
			continue
		}
		currentPC := uint16(patch.offsetPos * 2)
		targetPC := uint16(target.index * 2)
		offset := cg.calcBranchOffset(currentPC, targetPC)
		cg.setPatchImmediate(patch.bank, patch.offsetPos, uint16(offset))
	}

	return nil
}

// recordFuncAddr switches cg.currentBank per pass 2's bank schedule (if
// one is set) and records name's start address. Called once per
// function/helper immediately before its body is emitted. A function that
// does not fit continues into later banks from maybeSpill, up to the end
// bank reserved for it.
func (cg *CodeGenerator) recordFuncAddr(name string) {
	if cg.bankSchedule != nil {
		if b, ok := cg.bankSchedule[name]; ok {
			cg.currentBank = b
		}
	}
	cg.currentFn = name
	cg.fnBankEnd = cg.currentBank
	if cg.fnBankEnds != nil {
		if end, ok := cg.fnBankEnds[name]; ok && end > cg.fnBankEnd {
			cg.fnBankEnd = end
		}
	}
	cg.functionAddrs[name] = funcAddr{bank: cg.currentBank, index: cg.builder.GetCodeLength()}
	cg.emitOrder = append(cg.emitOrder, name)
}

// setPatchImmediate patches a single immediate word at wordIndex in the
// given bank. Used for cross-function patches (callPatches, the IRQ vector
// fix-up) that resolve after cg.currentBank has moved past the bank the
// patch site was originally emitted in -- resolving via cg.builder directly
// would silently patch the wrong bank's code. In single-bank builds (pass
// 1/2, cg.bankedBuilder == nil) bank is irrelevant and the flat builder is
// patched directly, identical to today's cg.builder.SetImmediateAt.
func (cg *CodeGenerator) setPatchImmediate(bank uint8, wordIndex int, value uint16) {
	if cg.bankedBuilder != nil {
		cg.bankedBuilder.SetImmediateAt(bank, wordIndex, value)
		return
	}
	cg.builder.SetImmediateAt(wordIndex, value)
}

func (cg *CodeGenerator) generateFunction(fn *FunctionDecl) error {
	// Reset variable tracking for each function.
	cg.variables = make(map[string]*VariableInfo)
	cg.openBranches = nil

	// Each function gets the bytes its locals actually use, packed downward
	// from the previous function. Frames do not overlap, and a function
	// cannot call itself: its locals are one fixed slot.
	if cg.globalStack < stackMinAddr+2 {
		return fmt.Errorf("stack allocation exhausted before function %s (global stack base 0x%04X)", fn.Name, cg.globalStack)
	}
	cg.stackOffset = cg.globalStack
	startStack := cg.globalStack
	cg.pushVarScope()
	defer cg.popVarScope()
	cg.exprSpill = 0
	spillBase, err := cg.allocateStack(exprSpillSlots*2, "expression temps for "+fn.Name)
	if err != nil {
		return fmt.Errorf("function %s: %w", fn.Name, err)
	}
	cg.spillBase = spillBase

	// Function prologue: save parameters from registers to local stack variables.
	for i, param := range fn.Params {
		if i >= 6 {
			return fmt.Errorf("function %s: too many parameters (max 6)", fn.Name)
		}
		stackAddr, err := cg.allocateStack(2, "function parameter "+param.Name)
		if err != nil {
			return fmt.Errorf("function %s: %w", fn.Name, err)
		}
		// Save R{i} to stack
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
		cg.builder.AddImmediate(stackAddr)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 7, uint8(i)))
		paramVarType := ""
		var elemWidth uint8
		var arrayLen int
		arrayRef := false
		structType := cg.structTypeNameFromTypeExpr(param.Type)
		if named, ok := param.Type.(*NamedType); ok {
			paramVarType = named.Name
		}
		if arr, ok := param.Type.(*ArrayType); ok {
			arrayRef = true
			structType = ""
			arrayLen = arr.Len
			if arrayLen == 0 && arr.LenName != "" {
				if v, ok := cg.consts[arr.LenName]; ok {
					arrayLen = int(v)
				}
			}
			elemName := typeExprName(arr.Elem)
			paramVarType = elemName
			sz, st, err := cg.elemSize(elemName)
			if err != nil {
				return fmt.Errorf("function %s: parameter %s: %w", fn.Name, param.Name, err)
			}
			elemWidth = uint8(sz)
			structType = st
		}
		cg.bindLocal(&VariableInfo{
			Name:       param.Name,
			Location:   VarLocationStack,
			StackAddr:  stackAddr,
			StructType: structType,
			VarType:    paramVarType,
			ElemWidth:  elemWidth,
			ArrayLen:   arrayLen,
			ArrayRef:   arrayRef,
		})
	}

	// The entry function is generated first; its preamble initializes globals.
	if len(cg.functionAddrs) == 1 {
		if err := cg.emitGlobalInits(); err != nil {
			return err
		}
	}

	// Generate function body
	if err := cg.generateStmtList(fn.Body); err != nil {
		return err
	}

	// Function epilogue. An oversized function may have filled its current
	// bank on the last statement, so the RET is allowed to land in the next
	// reserved bank.
	if err := cg.maybeSpill(); err != nil {
		return err
	}
	if cg.isEntryFunc() {
		if err := cg.emitHaltLoop(); err != nil {
			return err
		}
	} else {
		cg.builder.AddInstruction(rom.EncodeRET())
	}

	// Next function starts where this one's locals ended.
	cg.globalStack = cg.stackOffset
	if cg.globalStack == startStack && cg.globalStack > stackMinAddr {
		// Keep two functions that allocate nothing from sharing one address.
		cg.globalStack--
	}

	return nil
}

func (cg *CodeGenerator) isEntryFunc() bool {
	return cg.currentFn == "Start" || cg.currentFn == "__Boot"
}

// emitHaltLoop spins forever. Falling off Start or __Boot must not RET
// into an address the caller never pushed.
func (cg *CodeGenerator) emitHaltLoop() error {
	start := cg.here()
	site := cg.emitOpenJump()
	return cg.patchPos(site, start)
}

func (cg *CodeGenerator) resolveStructMember(varInfo *VariableInfo, member string) (structMemberInfo, bool) {
	layout, ok := cg.structLayoutFor(varInfo.StructType)
	if !ok {
		return structMemberInfo{}, false
	}
	info, ok := layout.Fields[member]
	return info, ok
}

func (cg *CodeGenerator) emitStructMemberStore(varInfo *VariableInfo, member structMemberInfo, valueReg uint8) bool {
	if varInfo.Location == VarLocationStack {
		// R6 = struct base address loaded from stack slot.
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
		cg.builder.AddImmediate(varInfo.StackAddr)
		cg.builder.AddInstruction(rom.EncodeMOV(2, 6, 7))

		if member.Width == 2 {
			// 16-bit store to [R6+offset]
			cg.builder.AddInstruction(rom.EncodeMOV(10, 6, valueReg))
			cg.builder.AddImmediate(member.Offset)
			return true
		}

		// 8-bit store path
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
		cg.builder.AddImmediate(member.Offset)
		cg.builder.AddInstruction(rom.EncodeADD(0, 6, 7))
		cg.builder.AddInstruction(rom.EncodeMOV(7, 6, valueReg))
		return true
	}

	return false
}

func (cg *CodeGenerator) emitStructMemberLoad(varInfo *VariableInfo, member structMemberInfo, destReg uint8) bool {
	if varInfo.Location == VarLocationStack {
		// R6 = struct base address loaded from stack slot.
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
		cg.builder.AddImmediate(varInfo.StackAddr)
		cg.builder.AddInstruction(rom.EncodeMOV(2, 6, 7))

		if member.Width == 2 {
			// 16-bit load from [R6+offset]
			cg.builder.AddInstruction(rom.EncodeMOV(9, destReg, 6))
			cg.builder.AddImmediate(member.Offset)
			return true
		}

		// 8-bit load path
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
		cg.builder.AddImmediate(member.Offset)
		cg.builder.AddInstruction(rom.EncodeADD(0, 6, 7))
		cg.builder.AddInstruction(rom.EncodeMOV(6, destReg, 6))
		return true
	}

	return false
}

func (cg *CodeGenerator) generateStmtList(stmts []Stmt) error {
	for _, stmt := range stmts {
		if err := cg.maybeSpill(); err != nil {
			return err
		}
		if err := cg.generateStmt(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (cg *CodeGenerator) generateStmt(stmt Stmt) error {
	var err error
	switch s := stmt.(type) {
	case *VarDeclStmt:
		err = cg.generateVarDecl(s)
	case *AssignStmt:
		err = cg.generateAssign(s)
	case *IfStmt:
		err = cg.generateIf(s)
	case *WhileStmt:
		err = cg.generateWhile(s)
	case *ForStmt:
		err = cg.generateFor(s)
	case *BreakStmt:
		err = cg.generateBreak(s)
	case *ContinueStmt:
		err = cg.generateContinue(s)
	case *ReturnStmt:
		err = cg.generateReturn(s)
	case *ExprStmt:
		err = cg.generateExpr(s.Expr, 0)
	default:
		err = fmt.Errorf("unsupported statement type: %T", stmt)
	}
	return withCodePos(stmt.Pos(), err)
}

// codeError is a codegen failure that knows the source line it came from.
type codeError struct {
	pos Position
	err error
}

func (e *codeError) Error() string { return e.err.Error() }
func (e *codeError) Unwrap() error { return e.err }

func withCodePos(pos Position, err error) error {
	if err == nil || pos.Line <= 0 {
		return err
	}
	var existing *codeError
	if errors.As(err, &existing) {
		return err
	}
	return &codeError{pos: pos, err: err}
}

// generateBreak emits an unconditional jump to the innermost enclosing
// loop's exit point, backpatched once that point is known.
func (cg *CodeGenerator) generateBreak(stmt *BreakStmt) error {
	if len(cg.loopStack) == 0 {
		return fmt.Errorf("break used outside of a loop")
	}
	loop := cg.loopStack[len(cg.loopStack)-1]
	loop.breakPatches = append(loop.breakPatches, cg.emitOpenJump())
	return nil
}

// generateContinue emits an unconditional jump to the innermost enclosing
// loop's next-iteration point, backpatched once that point is known.
func (cg *CodeGenerator) generateContinue(stmt *ContinueStmt) error {
	if len(cg.loopStack) == 0 {
		return fmt.Errorf("continue used outside of a loop")
	}
	loop := cg.loopStack[len(cg.loopStack)-1]
	loop.continuePatches = append(loop.continuePatches, cg.emitOpenJump())
	return nil
}

func (cg *CodeGenerator) generateVarDecl(stmt *VarDeclStmt) error {
	if arr, ok := stmt.Type.(*ArrayType); ok {
		if stmt.Value != nil {
			return fmt.Errorf("array %s starts at zero; assign its slots", stmt.Name)
		}
		return cg.generateLocalArray(stmt, arr)
	}
	// Check if initializer is a struct initialization
	if call, ok := stmt.Value.(*CallExpr); ok {
		if ident, ok := call.Func.(*IdentExpr); ok {
			if _, isStruct := cg.structLayoutFor(ident.Name); isStruct {
				// Struct initialization - use generateCall to allocate and get address
				// This ensures the struct is properly allocated and address is returned
				if err := cg.generateCall(call, 0); err != nil {
					return err
				}
				// R0 now contains the struct address
				// We need to track this address for the variable
				// But we don't know the address at compile time, so we need to store it
				// For now, allocate a register or stack slot to hold the address
				// Actually, we can't store it because we don't know the address until runtime
				// So we need to track that this variable holds a struct address
				// The address is computed at runtime by generateCall
				// We'll need to store R0 somewhere and track it
				// Pre-alpha simplification: keep long-lived locals on stack because
				// builtins use R0-R7 freely and there is no caller/callee-save contract yet.
				stackAddr, err := cg.allocateStack(2, "struct variable "+stmt.Name)
				if err != nil {
					return err
				}
				cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #stackAddr
				cg.builder.AddImmediate(stackAddr)
				cg.builder.AddInstruction(rom.EncodeMOV(3, 7, 0)) // MOV [R7], R0
				cg.bindLocal(&VariableInfo{
					Name:       stmt.Name,
					Location:   VarLocationStack,
					StackAddr:  stackAddr,
					StructType: ident.Name,
				})
				return nil
			}
		}
	}

	// Regular variable initialization
	// Generate code for initializer
	if err := cg.generateExpr(stmt.Value, 0); err != nil {
		return err
	}
	// Value is now in R0
	// Pre-alpha simplification: store locals on stack. This avoids register clobbering
	// by builtins until a real calling convention/register allocator is implemented.
	stackAddr, err := cg.allocateStack(2, "variable "+stmt.Name) // Allocate 2 bytes (16-bit value)
	if err != nil {
		return err
	}
	cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #stackAddr
	cg.builder.AddImmediate(stackAddr)
	cg.builder.AddInstruction(rom.EncodeMOV(3, 7, 0)) // MOV [R7], R0
	varType := cg.typeOf(stmt.Value)
	if name := typeExprName(stmt.Type); name != "" {
		if err := cg.checkInitType(name, stmt.Value); err != nil {
			return err
		}
		varType = name
	}
	cg.bindLocal(&VariableInfo{
		Name:       stmt.Name,
		Location:   VarLocationStack,
		StackAddr:  stackAddr,
		StructType: cg.structTypeNameFromTypeExpr(stmt.Type),
		VarType:    varType,
	})
	return nil
}

func (cg *CodeGenerator) generateLocalArray(stmt *VarDeclStmt, arr *ArrayType) error {
	length := arr.Len
	if length == 0 && arr.LenName != "" {
		v, ok := cg.consts[arr.LenName]
		if !ok || v <= 0 || v > 0x4000 {
			return fmt.Errorf("array length %s must be a constant between 1 and 16384", arr.LenName)
		}
		length = int(v)
	}
	if length <= 0 {
		return fmt.Errorf("array %s needs a length", stmt.Name)
	}
	elemName := typeExprName(arr.Elem)
	elemSize, structName, err := cg.elemSize(elemName)
	if err != nil {
		return err
	}
	bytes := uint16(length) * elemSize
	alloc := bytes
	if alloc%2 == 1 {
		alloc++
	}
	addr, err := cg.allocateStack(alloc, "array "+stmt.Name)
	if err != nil {
		return err
	}
	cg.zeroMemory(addr, bytes)
	cg.bindLocal(&VariableInfo{
		Name:       stmt.Name,
		Location:   VarLocationStack,
		StackAddr:  addr,
		ElemWidth:  uint8(elemSize),
		ArrayLen:   length,
		VarType:    elemName,
		StructType: structName,
		Direct:     true,
	})
	return nil
}

func (cg *CodeGenerator) zeroMemory(addr, size uint16) {
	cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
	cg.builder.AddImmediate(addr)
	cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
	cg.builder.AddImmediate(0)
	for off := uint16(0); off+1 < size; off += 2 {
		cg.builder.AddInstruction(rom.EncodeMOV(10, 7, 6))
		cg.builder.AddImmediate(off)
	}
	if size%2 == 1 {
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
		cg.builder.AddImmediate(size - 1)
		cg.builder.AddInstruction(rom.EncodeADD(0, 7, 6))
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
		cg.builder.AddImmediate(0)
		cg.builder.AddInstruction(rom.EncodeMOV(7, 7, 6))
	}
}

func (cg *CodeGenerator) pushVarScope() {
	cg.varScopes = append(cg.varScopes, varScope{
		saved: map[string]*VariableInfo{},
		had:   map[string]bool{},
	})
}

func (cg *CodeGenerator) popVarScope() {
	if len(cg.varScopes) == 0 {
		return
	}
	top := cg.varScopes[len(cg.varScopes)-1]
	cg.varScopes = cg.varScopes[:len(cg.varScopes)-1]
	for name := range top.saved {
		if top.had[name] {
			cg.variables[name] = top.saved[name]
		} else {
			delete(cg.variables, name)
		}
	}
}

func (cg *CodeGenerator) bindLocal(info *VariableInfo) {
	if len(cg.varScopes) == 0 {
		cg.variables[info.Name] = info
		return
	}
	top := &cg.varScopes[len(cg.varScopes)-1]
	if _, seen := top.saved[info.Name]; !seen {
		prev, had := cg.variables[info.Name]
		top.saved[info.Name] = prev
		top.had[info.Name] = had
	}
	cg.variables[info.Name] = info
}

func (cg *CodeGenerator) generateAssign(stmt *AssignStmt) error {
	// Charter D4: assigning across fixed/int without conversion is an error.
	tt, vt := cg.typeOf(stmt.Target), cg.typeOf(stmt.Value)
	if (tt == typeFixed && vt == typeInt) || (tt == typeInt && vt == typeFixed) {
		return fmt.Errorf("cannot assign %s value to %s variable — convert explicitly with int(x) or fixed(x)", vt, tt)
	}
	// Generate code for value
	if err := cg.generateExpr(stmt.Value, 0); err != nil {
		return err
	}
	// Value is in R0
	// Generate code to store to target
	if idx, ok := stmt.Target.(*IndexExpr); ok {
		info, err := cg.resolveArrayGlobal(idx)
		if err != nil {
			return err
		}
		if info.StructType != "" {
			return fmt.Errorf("index a field of %s", info.Name)
		}
		// Preserve R0 (value) in the runtime-block scratch slot while the
		// index expression is evaluated (it may contain calls that clobber
		// registers).
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #scratch
		cg.builder.AddImmediate(runtimeBlockBase)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 7, 0)) // MOV [R7], R0
		if err := cg.emitIndexAddr(idx, info); err != nil {
			return err
		}
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #scratch
		cg.builder.AddImmediate(runtimeBlockBase)
		cg.builder.AddInstruction(rom.EncodeMOV(2, 5, 6)) // MOV R5, [R6] (reload value)
		if info.ElemWidth == 2 {
			cg.builder.AddInstruction(rom.EncodeMOV(3, 7, 5)) // 16-bit store [R7], R5
		} else {
			cg.builder.AddInstruction(rom.EncodeMOV(7, 7, 5)) // 8-bit store [R7], R5
		}
		return nil
	}
	if member, ok := stmt.Target.(*MemberExpr); ok {
		// A local created with Sprite() or Player() holds a pointer. That
		// path is unchanged. Globals, nested fields, and struct arrays use
		// the address walker below.
		if ident, ok := member.Object.(*IdentExpr); ok {
			if varInfo, exists := cg.variables[ident.Name]; exists && !varInfo.Direct {
				if varInfo.StructType == "" {
					return fmt.Errorf("cannot assign to %s.%s: %s is not a struct variable", ident.Name, member.Member, ident.Name)
				}
				memberInfo, found := cg.resolveStructMember(varInfo, member.Member)
				if !found {
					return fmt.Errorf("struct %s has no field %q", varInfo.StructType, member.Member)
				}
				if memberInfo.Width != 0 {
					if !cg.emitStructMemberStore(varInfo, memberInfo, 0) {
						return fmt.Errorf("cannot assign to %s.%s: unsupported storage location", ident.Name, member.Member)
					}
					return nil
				}
			}
		}
		return cg.storePlaced(member)
	}

	// Regular assignment: x = value
	if ident, ok := stmt.Target.(*IdentExpr); ok {
		if _, isConst := cg.consts[ident.Name]; isConst {
			if _, shadowed := cg.variables[ident.Name]; !shadowed {
				return fmt.Errorf("cannot assign to constant %s", ident.Name)
			}
		}
		if _, exists := cg.variables[ident.Name]; !exists {
			if gInfo, isGlobal := cg.globals[ident.Name]; isGlobal {
				cg.storeGlobalScalar(gInfo, 0)
				return nil
			}
		}
		if varInfo, exists := cg.variables[ident.Name]; exists {
			// Store value to variable location
			if varInfo.Location == VarLocationStack {
				// Store to stack
				cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #stackAddr
				cg.builder.AddImmediate(varInfo.StackAddr)
				cg.builder.AddInstruction(rom.EncodeMOV(3, 7, 0)) // MOV [R7], R0
			}
			return nil
		}
		// Variable not found - this is an error (should have been declared)
		// But for compatibility, create it as a new variable
		// This handles cases like: x = 10 where x wasn't declared
		return cg.generateVarDecl(&VarDeclStmt{
			Position: stmt.Position,
			Name:     ident.Name,
			Value:    stmt.Value,
		})
	}

	return fmt.Errorf("assignment target not supported: %T", stmt.Target)
}

func (cg *CodeGenerator) generateIf(stmt *IfStmt) error {
	// Generate condition
	if err := cg.generateExpr(stmt.Condition, 0); err != nil {
		return err
	}

	// Compare with 0
	cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
	cg.builder.AddImmediate(0)
	cg.builder.AddInstruction(rom.EncodeCMP(0, 0, 7)) // CMP R0, R7

	// Branch if false. The site records its bank so a then-block that
	// continues into a later bank can still reach the else clause.
	elseSite := cg.emitOpenCond(rom.EncodeBEQ())

	cg.pushVarScope()
	err := cg.generateStmtList(stmt.Then)
	cg.popVarScope()
	if err != nil {
		return err
	}

	// Jump past every later clause. A taken branch must not fall into the
	// next else-if or the else.
	endSites := []codePos{cg.emitOpenJump()}
	if err := cg.patchPos(elseSite, cg.here()); err != nil {
		return err
	}
	for _, clause := range stmt.ElseIf {
		if err := cg.generateExpr(clause.Condition, 0); err != nil {
			return err
		}
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
		cg.builder.AddImmediate(0)
		cg.builder.AddInstruction(rom.EncodeCMP(0, 0, 7))
		skipSite := cg.emitOpenCond(rom.EncodeBEQ())
		cg.pushVarScope()
		err := cg.generateStmtList(clause.Body)
		cg.popVarScope()
		if err != nil {
			return err
		}
		endSites = append(endSites, cg.emitOpenJump())
		if err := cg.patchPos(skipSite, cg.here()); err != nil {
			return err
		}
	}

	cg.pushVarScope()
	err = cg.generateStmtList(stmt.Else)
	cg.popVarScope()
	if err != nil {
		return err
	}
	end := cg.here()
	for _, site := range endSites {
		if err := cg.patchPos(site, end); err != nil {
			return err
		}
	}
	return nil
}

func (cg *CodeGenerator) generateWhile(stmt *WhileStmt) error {
	loopStart := cg.here()

	// Generate condition
	if err := cg.generateExpr(stmt.Condition, 0); err != nil {
		return err
	}

	// Compare with 0
	cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
	cg.builder.AddImmediate(0)
	cg.builder.AddInstruction(rom.EncodeCMP(0, 0, 7))

	// Branch if false (exit loop)
	exitSite := cg.emitOpenCond(rom.EncodeBEQ())

	// Generate body
	loop := &loopContext{}
	cg.loopStack = append(cg.loopStack, loop)
	cg.pushVarScope()
	err := cg.generateStmtList(stmt.Body)
	cg.popVarScope()
	if err != nil {
		return err
	}
	cg.loopStack = cg.loopStack[:len(cg.loopStack)-1]

	// continue jumps here: back to the top of the loop, re-checking the condition.
	for _, p := range loop.continuePatches {
		if err := cg.patchPos(p, loopStart); err != nil {
			return err
		}
	}

	// Jump back to start. The body may have continued into a later bank.
	if err := cg.maybeSpill(); err != nil {
		return err
	}
	back := cg.emitOpenJump()
	if err := cg.patchPos(back, loopStart); err != nil {
		return err
	}

	exit := cg.here()
	if err := cg.patchPos(exitSite, exit); err != nil {
		return err
	}
	for _, p := range loop.breakPatches {
		if err := cg.patchPos(p, exit); err != nil {
			return err
		}
	}
	return nil
}

func (cg *CodeGenerator) generateFor(stmt *ForStmt) error {
	// BASIC counting loop: for i = start to end [step n], inclusive bounds.
	// Step must be a compile-time constant so the loop direction (and thus the
	// exit comparison) is known at compile time. Default step is +1.
	step := int64(1)
	if stmt.Step != nil {
		v, err := evalConstExpr(stmt.Step, cg.consts)
		if err != nil {
			return fmt.Errorf("for loop 'step' must be a constant: %w", err)
		}
		if v == 0 {
			return fmt.Errorf("for loop 'step' cannot be zero")
		}
		step = v
	}

	// Loop variable: a fresh local in WRAM, visible only for this loop.
	stackAddr, err := cg.allocateStack(2, "for loop variable "+stmt.VarName)
	if err != nil {
		return err
	}
	cg.pushVarScope()
	defer cg.popVarScope()
	cg.bindLocal(&VariableInfo{
		Name:      stmt.VarName,
		Location:  VarLocationStack,
		StackAddr: stackAddr,
		VarType:   typeInt,
	})

	// i = start
	if err := cg.generateExpr(stmt.Start, 0); err != nil {
		return err
	}
	cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
	cg.builder.AddImmediate(stackAddr)
	cg.builder.AddInstruction(rom.EncodeMOV(3, 7, 0)) // [i] = R0

	loopStart := cg.here()

	// Re-evaluate the limit each iteration into R1 (correct if it's a variable).
	if err := cg.generateExpr(stmt.End, 1); err != nil {
		return err
	}
	// Load i into R0.
	cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
	cg.builder.AddImmediate(stackAddr)
	cg.builder.AddInstruction(rom.EncodeMOV(2, 0, 7)) // R0 = [i]
	// Compare i (R0) to end (R1).
	cg.builder.AddInstruction(rom.EncodeCMP(0, 0, 1))

	// Exit when past the inclusive limit: step>0 exits if i>end (BGT);
	// step<0 exits if i<end (BLT).
	var exitSite codePos
	if step > 0 {
		exitSite = cg.emitOpenCond(rom.EncodeBGT())
	} else {
		exitSite = cg.emitOpenCond(rom.EncodeBLT())
	}

	// Body.
	loop := &loopContext{}
	cg.loopStack = append(cg.loopStack, loop)
	if err := cg.generateStmtList(stmt.Body); err != nil {
		return err
	}
	cg.loopStack = cg.loopStack[:len(cg.loopStack)-1]

	// continue jumps here: increment i, then recheck the condition — skipping
	// the increment would leave i unchanged and loop forever. Spill first so
	// the increment itself is not split off the label continues target.
	if err := cg.maybeSpill(); err != nil {
		return err
	}
	continueTarget := cg.here()
	for _, p := range loop.continuePatches {
		if err := cg.patchPos(p, continueTarget); err != nil {
			return err
		}
	}

	// i = i + step
	cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
	cg.builder.AddImmediate(stackAddr)
	cg.builder.AddInstruction(rom.EncodeMOV(2, 0, 7)) // R0 = [i]
	cg.builder.AddInstruction(rom.EncodeMOV(1, 1, 0)) // R1 = #step (as 16-bit two's complement)
	cg.builder.AddImmediate(uint16(int16(step)))
	cg.builder.AddInstruction(rom.EncodeADD(0, 0, 1)) // R0 += step
	cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
	cg.builder.AddImmediate(stackAddr)
	cg.builder.AddInstruction(rom.EncodeMOV(3, 7, 0)) // [i] = R0

	// Jump back to the condition.
	if err := cg.maybeSpill(); err != nil {
		return err
	}
	back := cg.emitOpenJump()
	if err := cg.patchPos(back, loopStart); err != nil {
		return err
	}

	exit := cg.here()
	if err := cg.patchPos(exitSite, exit); err != nil {
		return err
	}
	for _, p := range loop.breakPatches {
		if err := cg.patchPos(p, exit); err != nil {
			return err
		}
	}
	return nil
}

func (cg *CodeGenerator) generateReturn(stmt *ReturnStmt) error {
	if stmt.Value != nil {
		if err := cg.generateExpr(stmt.Value, 0); err != nil {
			return err
		}
		// Value is in R0
	}
	if cg.isEntryFunc() {
		return cg.emitHaltLoop()
	}
	cg.builder.AddInstruction(rom.EncodeRET())
	return nil
}

func (cg *CodeGenerator) generateExpr(expr Expr, destReg uint8) error {
	switch e := expr.(type) {
	case *NumberExpr:
		cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
		value := uint16(e.Value)
		if e.Value > 0xFFFF {
			value = uint16(e.Value & 0xFFFF)
		}
		cg.builder.AddImmediate(value)
		return nil

	case *BoolExpr:
		value := uint16(0)
		if e.Value {
			value = 1
		}
		cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
		cg.builder.AddImmediate(value)
		return nil

	case *IdentExpr:
		// Handle built-in constants and variables
		if strings.HasPrefix(e.Name, "ASSET_") {
			// Asset constant
			assetName := strings.TrimPrefix(e.Name, "ASSET_")
			if _, ok := cg.assets[assetName]; ok {
				assetID, ok := cg.assetIDs[assetName]
				if !ok {
					return fmt.Errorf("missing asset ID for %s", assetName)
				}
				// Return compiler-assigned asset ID.
				cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
				cg.builder.AddImmediate(assetID)
				return nil
			}
		}
		if assetID, ok := cg.assetIDs[e.Name]; ok {
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(assetID)
			return nil
		}
		// Variable access
		if varInfo, exists := cg.variables[e.Name]; exists {
			// Load from variable location
			if varInfo.Location == VarLocationStack {
				// Load from stack
				cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #stackAddr
				cg.builder.AddImmediate(varInfo.StackAddr)
				cg.builder.AddInstruction(rom.EncodeMOV(2, destReg, 7)) // MOV R{destReg}, [R7]
			}
			return nil
		}
		// Compile-time constant: load as immediate.
		if v, isConst := cg.consts[e.Name]; isConst {
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(uint16(v))
			return nil
		}
		// WRAM global: absolute load.
		if gInfo, isGlobal := cg.globals[e.Name]; isGlobal {
			cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #addr
			cg.builder.AddImmediate(gInfo.StackAddr)
			if gInfo.ElemWidth == 1 && gInfo.ArrayLen == 0 {
				cg.builder.AddInstruction(rom.EncodeMOV(6, destReg, 7))
			} else {
				cg.builder.AddInstruction(rom.EncodeMOV(2, destReg, 7))
			}
			return nil
		}
		return fmt.Errorf("undefined identifier %s", e.Name)

	case *BinaryExpr:
		// Charter D4: fixed/int operands may not mix without conversion.
		switch e.Op {
		case TOKEN_PLUS, TOKEN_MINUS, TOKEN_STAR, TOKEN_SLASH, TOKEN_PERCENT,
			TOKEN_EQUAL_EQUAL, TOKEN_BANG_EQUAL, TOKEN_LESS, TOKEN_LESS_EQUAL,
			TOKEN_GREATER, TOKEN_GREATER_EQUAL:
			if err := cg.checkNumericMix(tokenOpName(e.Op), e.Left, e.Right); err != nil {
				return err
			}
		}
		// Generate left operand, then the right. A non-leaf right operand
		// (another binary op, a call, a field) reuses R1, so the left value
		// is parked in the compiler's spill slots for that case. A leaf
		// right operand stays in R1, which is the one-instruction save the
		// hot path already used.
		if err := cg.generateExpr(e.Left, destReg); err != nil {
			return err
		}
		spilled := exprNeedsSpill(e.Right)
		if spilled {
			if err := cg.spillPush(destReg); err != nil {
				return err
			}
		} else {
			cg.builder.AddInstruction(rom.EncodeMOV(0, 1, destReg))
		}
		// The right operand normally lands in R2. When this expression's own
		// destination is R2 (it is the right-hand side of another operator,
		// or a call's third argument), R2 would be both the result and the
		// operand, so the right value goes to R3 instead.
		rightReg := uint8(2)
		if destReg == rightReg {
			rightReg = 3
		}
		if err := cg.generateExpr(e.Right, rightReg); err != nil {
			return err
		}
		if spilled {
			cg.spillPop(1)
		}
		// Perform operation
		switch e.Op {
		case TOKEN_PLUS:
			// Restore left result to destReg, then add
			cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 1)) // MOV R{destReg}, R1 (restore left)
			cg.builder.AddInstruction(rom.EncodeADD(0, destReg, rightReg))
		case TOKEN_MINUS:
			// Restore left result to destReg, then subtract
			cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 1)) // MOV R{destReg}, R1 (restore left)
			cg.builder.AddInstruction(rom.EncodeSUB(0, destReg, rightReg))
		case TOKEN_STAR:
			// fixed * fixed routes through the signed 8.8 software multiply.
			if cg.typeOf(e.Left) == typeFixed && cg.typeOf(e.Right) == typeFixed {
				cg.needFixmul = true
				cg.builder.AddInstruction(rom.EncodeMOV(0, 0, 1)) // MOV R0, R1 (left)
				cg.builder.AddInstruction(rom.EncodeMOV(0, 1, rightReg))
				cg.emitHelperCall("__fixmul")
				if destReg != 0 {
					cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 0))
				}
				return nil
			}
			// General constant multiplication via shift-add decomposition.
			// For runtime multipliers, fall back to shift-add loop.
			if numExpr, ok := e.Right.(*NumberExpr); ok {
				val := numExpr.Value
				if val == 0 {
					cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
					cg.builder.AddImmediate(0)
					return nil
				}
				if val == 1 {
					if destReg != 1 {
						cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 1))
					}
					return nil
				}
				// Shift-add decomposition. Scratch must not be destReg: array
				// indexing evaluates the index expression into R6, and a
				// multiplier such as 3 uses R6 for the shifted partial.
				cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 1))
				scratch := uint8(4)
				if destReg == scratch {
					scratch = 5
				}
				countReg := uint8(7)
				if destReg == countReg {
					countReg = 3
				}
				first := true
				for bit := 0; bit < 16; bit++ {
					if val&(1<<uint(bit)) != 0 {
						if bit == 0 {
							if first {
								first = false
							} else {
								cg.builder.AddInstruction(rom.EncodeADD(0, destReg, 1))
							}
						} else {
							cg.builder.AddInstruction(rom.EncodeMOV(0, scratch, 1))
							cg.builder.AddInstruction(rom.EncodeMOV(1, countReg, 0))
							cg.builder.AddImmediate(uint16(bit))
							cg.builder.AddInstruction(rom.EncodeSHL(0, scratch, countReg))
							if first {
								cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, scratch))
								first = false
							} else {
								cg.builder.AddInstruction(rom.EncodeADD(0, destReg, scratch))
							}
						}
					}
				}
				return nil
			}
			// Runtime multiply: hardware MUL (low 16 bits).
			cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 1)) // restore left
			cg.builder.AddInstruction(rom.EncodeMUL(0, destReg, rightReg))
			return nil
		case TOKEN_SLASH:
			if numExpr, ok := e.Right.(*NumberExpr); ok && numExpr.Value == 0 {
				return fmt.Errorf("division by zero")
			}
			// Signed, toward zero. fixed/fixed is 8.8 (see __fixdiv);
			// int/int is 16-bit (see __divs). Both match const folding.
			// Non-negative ints divide inline. A far call on every positive
			// quotient is enough to push the overworld's sprite writes past
			// the scanline that displays them.
			cg.builder.AddInstruction(rom.EncodeMOV(0, 0, 1))
			if cg.typeOf(e.Left) == typeFixed {
				cg.builder.AddInstruction(rom.EncodeMOV(0, 1, rightReg))
				cg.needFixdiv = true
				cg.emitHelperCall("__fixdiv")
			} else if shift, ok := constPow2Shift(e.Right); ok {
				cg.emitNonNegDivFast(true, shift)
			} else {
				cg.builder.AddInstruction(rom.EncodeMOV(0, 1, rightReg))
				cg.emitNonNegDivFast(false, 0)
			}
			if destReg != 0 {
				cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 0))
			}
			return nil
		case TOKEN_PERCENT:
			if cg.typeOf(e.Left) == typeFixed || cg.typeOf(e.Right) == typeFixed {
				return fmt.Errorf("modulo is only defined for int")
			}
			if numExpr, ok := e.Right.(*NumberExpr); ok && numExpr.Value == 0 {
				return fmt.Errorf("modulo by zero")
			}
			// rem = dividend - quotient*divisor, toward zero, sign of the
			// dividend. Non-negative operands stay inline; a negative one
			// uses __divs, which does not clobber R4/R5.
			cg.needDivs = true
			cg.builder.AddInstruction(rom.EncodeMOV(0, 0, 1))
			cg.builder.AddInstruction(rom.EncodeMOV(0, 1, rightReg))
			cg.builder.AddInstruction(rom.EncodeMOV(0, 4, 0))
			cg.builder.AddInstruction(rom.EncodeMOV(0, 5, 1))
			cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 0))
			cg.builder.AddInstruction(rom.EncodeOR(0, 6, 1))
			cg.hShrImm(6, 15)
			cg.hCmpImm(6, 0)
			negMod := cg.hBranch(rom.EncodeBNE())
			cg.builder.AddInstruction(rom.EncodeDIV(0, 0, 5))
			joinMod := cg.hBranch(rom.EncodeJMP())
			cg.hPatchToHere(negMod)
			cg.emitHelperCall("__divs")
			cg.hPatchToHere(joinMod)
			cg.builder.AddInstruction(rom.EncodeMOV(0, 2, 0))
			cg.builder.AddInstruction(rom.EncodeMUL(0, 2, 5))
			cg.builder.AddInstruction(rom.EncodeMOV(0, 0, 4))
			cg.builder.AddInstruction(rom.EncodeSUB(0, 0, 2))
			if destReg != 0 {
				cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 0))
			}
			return nil
		case TOKEN_EQUAL_EQUAL:
			// Compare and set result: 1 if equal, 0 if not.
			// Important: branch immediately after CMP (MOV updates flags).
			cg.builder.AddInstruction(rom.EncodeCMP(0, 1, rightReg))
			cg.builder.AddInstruction(rom.EncodeBNE()) // BNE false
			falsePos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0)) // true => 1
			cg.builder.AddImmediate(1)
			cg.builder.AddInstruction(rom.EncodeJMP()) // JMP end
			endPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.patchLabel(falsePos)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0)) // false => 0
			cg.builder.AddImmediate(0)
			cg.patchLabel(endPos)
			return nil
		case TOKEN_BANG_EQUAL:
			// Compare and set result: 1 if not equal, 0 if equal.
			cg.builder.AddInstruction(rom.EncodeCMP(0, 1, rightReg))
			cg.builder.AddInstruction(rom.EncodeBEQ()) // BEQ false (equal => false)
			falsePos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0)) // true => 1
			cg.builder.AddImmediate(1)
			cg.builder.AddInstruction(rom.EncodeJMP())
			endPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.patchLabel(falsePos)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0)) // false => 0
			cg.builder.AddImmediate(0)
			cg.patchLabel(endPos)
			return nil
		case TOKEN_LESS:
			cg.builder.AddInstruction(rom.EncodeCMP(0, 1, rightReg))
			cg.builder.AddInstruction(rom.EncodeBGE()) // >= => false
			falsePos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0)) // true => 1
			cg.builder.AddImmediate(1)
			cg.builder.AddInstruction(rom.EncodeJMP())
			endPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.patchLabel(falsePos)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0)) // false => 0
			cg.builder.AddImmediate(0)
			cg.patchLabel(endPos)
			return nil
		case TOKEN_GREATER:
			cg.builder.AddInstruction(rom.EncodeCMP(0, 1, rightReg))
			cg.builder.AddInstruction(rom.EncodeBLE()) // <= => false
			falsePos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(1)
			cg.builder.AddInstruction(rom.EncodeJMP())
			endPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.patchLabel(falsePos)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(0)
			cg.patchLabel(endPos)
			return nil
		case TOKEN_LESS_EQUAL:
			cg.builder.AddInstruction(rom.EncodeCMP(0, 1, rightReg))
			cg.builder.AddInstruction(rom.EncodeBGT()) // > => false
			falsePos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(1)
			cg.builder.AddInstruction(rom.EncodeJMP())
			endPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.patchLabel(falsePos)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(0)
			cg.patchLabel(endPos)
			return nil
		case TOKEN_GREATER_EQUAL:
			cg.builder.AddInstruction(rom.EncodeCMP(0, 1, rightReg))
			cg.builder.AddInstruction(rom.EncodeBLT()) // < => false
			falsePos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(1)
			cg.builder.AddInstruction(rom.EncodeJMP())
			endPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.patchLabel(falsePos)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(0)
			cg.patchLabel(endPos)
			return nil
		case TOKEN_AND:
			// Logical AND: both must be non-zero
			// R1 already has left, R2 has right
			// Set R0 to 1 if both non-zero, else 0
			cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeCMP(0, 1, 7)) // CMP R1, R7
			cg.builder.AddInstruction(rom.EncodeBEQ())        // BEQ false
			falsePos1 := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeCMP(0, rightReg, 7))
			cg.builder.AddInstruction(rom.EncodeBEQ()) // BEQ false
			falsePos2 := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			// Both non-zero, set to 1
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(1)
			cg.builder.AddInstruction(rom.EncodeJMP())
			endPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			// False case
			cg.patchLabel(falsePos1)
			cg.patchLabel(falsePos2)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(0)
			cg.patchLabel(endPos)
			return nil
		case TOKEN_OR:
			// Logical OR: at least one non-zero
			cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeCMP(0, 1, 7))
			cg.builder.AddInstruction(rom.EncodeBNE()) // BNE true
			truePos1 := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeCMP(0, rightReg, 7))
			cg.builder.AddInstruction(rom.EncodeBNE()) // BNE true
			truePos2 := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			// Both zero, set to 0
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeJMP())
			endPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			// True case
			cg.patchLabel(truePos1)
			cg.patchLabel(truePos2)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(1)
			cg.patchLabel(endPos)
			return nil
		case TOKEN_PIPE:
			// Bitwise OR: left result is in R1, right result is in R2
			// OR R1, R2 -> result in R1, then move to destReg
			cg.builder.AddInstruction(rom.EncodeOR(0, 1, rightReg))
			cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 1)) // MOV R{destReg}, R1
			return nil
		case TOKEN_AMPERSAND:
			// Bitwise AND: left result is in R1, right result is in R2
			// AND R1, R2 -> result in R1, then move to destReg
			cg.builder.AddInstruction(rom.EncodeAND(0, 1, rightReg))
			cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 1)) // MOV R{destReg}, R1
			return nil
		case TOKEN_CARET:
			// Bitwise XOR
			cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 1))
			cg.builder.AddInstruction(rom.EncodeXOR(0, destReg, rightReg))
			return nil
		case TOKEN_LSHIFT:
			cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 1))
			cg.builder.AddInstruction(rom.EncodeSHL(0, destReg, rightReg))
			return nil
		case TOKEN_RSHIFT:
			// Arithmetic right shift (SAR): sign-extend. Logical SHR stays
			// available to compiler helpers via hShrImm.
			cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 1))
			cg.builder.AddInstruction(rom.EncodeSHR(2, destReg, rightReg))
			return nil
		case TOKEN_EQUAL:
			// Assignment operator in expression context (shouldn't happen in binary expr)
			// But handle it anyway
			return fmt.Errorf("assignment operator not allowed in expression context")
		default:
			return fmt.Errorf("binary operator not yet implemented: %v (%d)", e.Op, int(e.Op))
		}
		return nil

	case *CallExpr:
		return cg.generateCall(e, destReg)

	case *MemberExpr:
		// Handle member expressions
		// First check if it's a struct member access (variable exists)
		if ident, ok := e.Object.(*IdentExpr); ok {
			// Check if variable exists first (prioritize variable over namespace)
			if varInfo, exists := cg.variables[ident.Name]; exists && !varInfo.Direct {
				if varInfo.StructType == "" {
					return fmt.Errorf("cannot read %s.%s: %s is not a struct variable", ident.Name, e.Member, ident.Name)
				}
				memberInfo, found := cg.resolveStructMember(varInfo, e.Member)
				if !found {
					return fmt.Errorf("struct %s has no field %q", varInfo.StructType, e.Member)
				}
				if memberInfo.Width != 0 {
					if !cg.emitStructMemberLoad(varInfo, memberInfo, destReg) {
						return fmt.Errorf("cannot read %s.%s: unsupported storage location", ident.Name, e.Member)
					}
					return nil
				}
			}
		}
		if err := cg.loadPlaced(e, destReg); err == nil {
			return nil
		} else if ident, ok := e.Object.(*IdentExpr); ok {
			if _, isVar := cg.variables[ident.Name]; !isVar {
				if _, isGlobal := cg.globals[ident.Name]; !isGlobal {
					// Namespace member reads are not values. Keep the old
					// placeholder so a mistaken read still compiles to 0
					// only when semantic did not already reject the name.
					if err2 := cg.generateExpr(e.Object, destReg); err2 != nil {
						return err2
					}
					cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
					cg.builder.AddImmediate(0)
					return nil
				}
			}
			return err
		} else {
			return err
		}

	case *UnaryExpr:
		return cg.generateUnary(e, destReg)

	case *IndexExpr:
		return cg.generateIndexLoad(e, destReg)

	case *StringExpr:
		return fmt.Errorf("strings can only be used directly as a text.draw argument in v1 (strings are labels, not a data type)")

	default:
		return fmt.Errorf("unsupported expression type: %T", expr)
	}
}

const (
	placeDirect = iota
	placePointer
	placeIndex
)

// placed is the address of a struct or one of its fields.
type placed struct {
	kind       int
	addr       uint16
	slot       uint16
	index      *IndexExpr
	info       *VariableInfo
	offset     uint16
	width      uint8
	structName string
}

func (cg *CodeGenerator) placeExpr(expr Expr) (placed, error) {
	switch e := expr.(type) {
	case *IdentExpr:
		info, ok := cg.variables[e.Name]
		if !ok {
			info, ok = cg.globals[e.Name]
		}
		if !ok || info.StructType == "" || info.ArrayLen > 0 {
			return placed{}, fmt.Errorf("%s is not a struct", e.Name)
		}
		if info.Direct {
			return placed{kind: placeDirect, addr: info.StackAddr, structName: info.StructType}, nil
		}
		return placed{kind: placePointer, slot: info.StackAddr, structName: info.StructType}, nil
	case *IndexExpr:
		info, err := cg.resolveArrayGlobal(e)
		if err != nil {
			return placed{}, err
		}
		if info.StructType == "" {
			return placed{}, fmt.Errorf("%s is not an array of structs", info.Name)
		}
		return placed{kind: placeIndex, index: e, info: info, structName: info.StructType}, nil
	case *MemberExpr:
		base, err := cg.placeExpr(e.Object)
		if err != nil {
			return placed{}, err
		}
		layout, ok := cg.structLayoutFor(base.structName)
		if !ok || base.structName == "" {
			return placed{}, fmt.Errorf("no struct to read %s from", e.Member)
		}
		field, ok := layout.Fields[e.Member]
		if !ok {
			return placed{}, fmt.Errorf("struct %s has no field %q", base.structName, e.Member)
		}
		base.offset += field.Offset
		base.width = field.Width
		base.structName = field.Nested
		return base, nil
	default:
		return placed{}, fmt.Errorf("unsupported struct location")
	}
}

func (cg *CodeGenerator) loadPlaced(e *MemberExpr, dest uint8) error {
	p, err := cg.placeExpr(e)
	if err != nil {
		return err
	}
	if p.width == 0 {
		return fmt.Errorf("read a field of struct %s", p.structName)
	}
	return cg.loadPlacedValue(p, dest)
}

func (cg *CodeGenerator) loadPlacedValue(p placed, dest uint8) error {
	switch p.kind {
	case placeDirect:
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
		cg.builder.AddImmediate(p.addr + p.offset)
		if p.width == 2 {
			cg.builder.AddInstruction(rom.EncodeMOV(2, dest, 7))
		} else {
			cg.builder.AddInstruction(rom.EncodeMOV(6, dest, 7))
		}
		return nil
	case placePointer:
		if !cg.emitStructMemberLoad(&VariableInfo{Location: VarLocationStack, StackAddr: p.slot}, structMemberInfo{Offset: p.offset, Width: p.width}, dest) {
			return fmt.Errorf("cannot read struct field")
		}
		return nil
	case placeIndex:
		if err := cg.emitIndexAddr(p.index, p.info); err != nil {
			return err
		}
		if p.offset != 0 {
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(p.offset)
			cg.builder.AddInstruction(rom.EncodeADD(0, 7, 6))
		}
		if p.width == 2 {
			cg.builder.AddInstruction(rom.EncodeMOV(2, dest, 7))
		} else {
			cg.builder.AddInstruction(rom.EncodeMOV(6, dest, 7))
		}
		return nil
	default:
		return fmt.Errorf("cannot read struct field")
	}
}

func (cg *CodeGenerator) storePlaced(e *MemberExpr) error {
	p, err := cg.placeExpr(e)
	if err != nil {
		return err
	}
	if p.width == 0 {
		return fmt.Errorf("assign a field of struct %s", p.structName)
	}
	switch p.kind {
	case placeDirect:
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
		cg.builder.AddImmediate(p.addr + p.offset)
		if p.width == 2 {
			cg.builder.AddInstruction(rom.EncodeMOV(3, 7, 0))
		} else {
			cg.builder.AddInstruction(rom.EncodeMOV(7, 7, 0))
		}
		return nil
	case placePointer:
		if !cg.emitStructMemberStore(&VariableInfo{Location: VarLocationStack, StackAddr: p.slot}, structMemberInfo{Offset: p.offset, Width: p.width}, 0) {
			return fmt.Errorf("cannot assign struct field")
		}
		return nil
	case placeIndex:
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
		cg.builder.AddImmediate(runtimeBlockBase)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 7, 0))
		if err := cg.emitIndexAddr(p.index, p.info); err != nil {
			return err
		}
		if p.offset != 0 {
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
			cg.builder.AddImmediate(p.offset)
			cg.builder.AddInstruction(rom.EncodeADD(0, 7, 6))
		}
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
		cg.builder.AddImmediate(runtimeBlockBase)
		cg.builder.AddInstruction(rom.EncodeMOV(2, 5, 6))
		if p.width == 2 {
			cg.builder.AddInstruction(rom.EncodeMOV(3, 7, 5))
		} else {
			cg.builder.AddInstruction(rom.EncodeMOV(7, 7, 5))
		}
		return nil
	default:
		return fmt.Errorf("cannot assign struct field")
	}
}

func (cg *CodeGenerator) emitArrayArg(arg Expr, reg uint8) error {
	ident, ok := arg.(*IdentExpr)
	if !ok {
		return fmt.Errorf("an array argument must be an array name")
	}
	info, ok := cg.variables[ident.Name]
	if !ok {
		info, ok = cg.globals[ident.Name]
	}
	if !ok || info.ArrayLen == 0 {
		return fmt.Errorf("%s is not an array", ident.Name)
	}
	if info.ArrayRef {
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
		cg.builder.AddImmediate(info.StackAddr)
		cg.builder.AddInstruction(rom.EncodeMOV(2, reg, 7))
		return nil
	}
	cg.builder.AddInstruction(rom.EncodeMOV(1, reg, 0))
	cg.builder.AddImmediate(info.StackAddr)
	return nil
}

// resolveArrayGlobal returns the VariableInfo for an index expression whose
// base names an array.
func (cg *CodeGenerator) resolveArrayGlobal(e *IndexExpr) (*VariableInfo, error) {
	ident, ok := e.Array.(*IdentExpr)
	if !ok {
		return nil, fmt.Errorf("indexing is only supported on array names")
	}
	if info, ok := cg.variables[ident.Name]; ok && info.ArrayLen > 0 {
		return info, nil
	}
	info, isGlobal := cg.globals[ident.Name]
	if !isGlobal || info.ArrayLen == 0 {
		return nil, fmt.Errorf("%s is not an array", ident.Name)
	}
	return info, nil
}

// emitIndexAddr leaves the element's absolute WRAM address in R7.
// Constant indices are bounds-checked at compile time and folded.
func (cg *CodeGenerator) emitIndexAddr(e *IndexExpr, info *VariableInfo) error {
	if v, err := evalConstExpr(e.Index, cg.consts); err == nil {
		if v < 0 || int(v) >= info.ArrayLen {
			return fmt.Errorf("index %d out of bounds for %s[%d]", v, info.Name, info.ArrayLen)
		}
		if !info.ArrayRef {
			cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
			cg.builder.AddImmediate(info.StackAddr + uint16(v)*uint16(info.ElemWidth))
			return nil
		}
	}
	// Dynamic index: R6 = index, scaled; R7 = base + R6.
	if err := cg.generateExpr(e.Index, 6); err != nil {
		return err
	}
	cg.scaleIndex(info.ElemWidth)
	if info.ArrayRef {
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
		cg.builder.AddImmediate(info.StackAddr)
		cg.builder.AddInstruction(rom.EncodeMOV(2, 5, 7)) // R5 = base pointer
		cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 5))
		cg.builder.AddInstruction(rom.EncodeADD(0, 7, 6))
		return nil
	}
	cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #base
	cg.builder.AddImmediate(info.StackAddr)
	cg.builder.AddInstruction(rom.EncodeADD(0, 7, 6)) // R7 = base + scaled idx
	return nil
}

func (cg *CodeGenerator) scaleIndex(width uint8) {
	if width == 2 {
		cg.builder.AddInstruction(rom.EncodeADD(0, 6, 6)) // R6 += R6 (idx*2)
		return
	}
	if width == 1 || width == 0 {
		return
	}
	cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	cg.builder.AddImmediate(uint16(width))
	cg.builder.AddInstruction(rom.EncodeMUL(0, 6, 5))
}

// generateIndexLoad loads arr[i] into destReg (8- or 16-bit by element width).
func (cg *CodeGenerator) generateIndexLoad(e *IndexExpr, destReg uint8) error {
	info, err := cg.resolveArrayGlobal(e)
	if err != nil {
		return err
	}
	if info.StructType != "" {
		return fmt.Errorf("index a field of %s", info.Name)
	}
	if err := cg.emitIndexAddr(e, info); err != nil {
		return err
	}
	if info.ElemWidth == 2 {
		cg.builder.AddInstruction(rom.EncodeMOV(2, destReg, 7)) // 16-bit load
	} else {
		cg.builder.AddInstruction(rom.EncodeMOV(6, destReg, 7)) // 8-bit load
	}
	return nil
}

func (cg *CodeGenerator) generateCall(call *CallExpr, destReg uint8) error {
	// Get function name
	var funcName string
	if ident, ok := call.Func.(*IdentExpr); ok {
		funcName = ident.Name
	} else if member, ok := call.Func.(*MemberExpr); ok {
		// Handle member calls like sprite.set_pos
		if obj, ok := member.Object.(*IdentExpr); ok {
			funcName = obj.Name + "." + member.Member
		}
	}

	// Charter D4 numeric conversions.
	if (funcName == "int" || funcName == "fixed") && len(call.Args) == 1 {
		argType := cg.typeOf(call.Args[0])
		if err := cg.generateExpr(call.Args[0], destReg); err != nil {
			return err
		}
		if funcName == "int" && argType != typeInt {
			// fixed -> int: arithmetic shift right 8 (truncates toward -inf).
			cg.builder.AddInstruction(rom.EncodeSHR(3, destReg, 0))
			cg.builder.AddImmediate(8)
		}
		if funcName == "fixed" && argType != typeFixed {
			cg.builder.AddInstruction(rom.EncodeSHL(1, destReg, 0))
			cg.builder.AddImmediate(8)
		}
		return nil
	}

	// text.draw(x, y, r, g, b, "string") streams a string literal to the
	// hardware text port (0x8070-0x8076). Strings are labels in v1 (charter
	// D11), so the literal is emitted inline rather than as string data.
	if funcName == "text.draw" {
		if len(call.Args) != 6 {
			return fmt.Errorf("text.draw expects (x, y, r, g, b, string), got %d args", len(call.Args))
		}
		str, ok := call.Args[5].(*StringExpr)
		if !ok {
			return fmt.Errorf("text.draw: the last argument must be a string literal")
		}
		// X is 16-bit: low byte -> 0x8070, high byte -> 0x8071.
		if err := cg.generateExpr(call.Args[0], 0); err != nil {
			return err
		}
		cg.storeIOByte(0x8070, 0)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 0)) // R6 = X
		cg.builder.AddInstruction(rom.EncodeSHR(1, 6, 0)) // R6 >>= 8
		cg.builder.AddImmediate(8)
		cg.storeIOByte(0x8071, 6)
		// Y, R, G, B are single bytes.
		for i, addr := range []uint16{0x8072, 0x8073, 0x8074, 0x8075} {
			if err := cg.generateExpr(call.Args[i+1], 0); err != nil {
				return err
			}
			cg.storeIOByte(addr, 0)
		}
		// Stream each character to 0x8076 (the port auto-advances X by 8).
		cg.hMovImm(7, 0x8076)
		for _, ch := range str.Value {
			cg.builder.AddInstruction(rom.EncodeMOV(1, 0, 0)) // MOV R0, #char
			cg.builder.AddImmediate(uint16(ch))
			cg.builder.AddInstruction(rom.EncodeMOV(7, 7, 0)) // MOV [R7], R0 (8-bit store)
		}
		return nil
	}

	// text.draw_int(x, y, r, g, b, value) draws a signed integer as decimal
	// digits at the text port (so games can show scores, counters, positions).
	if funcName == "text.draw_int" {
		if len(call.Args) != 6 {
			return fmt.Errorf("text.draw_int expects (x, y, r, g, b, value), got %d args", len(call.Args))
		}
		// Port setup: X 16-bit, Y, R, G, B (same registers as text.draw).
		if err := cg.generateExpr(call.Args[0], 0); err != nil {
			return err
		}
		cg.storeIOByte(0x8070, 0)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 0))
		cg.builder.AddInstruction(rom.EncodeSHR(1, 6, 0))
		cg.builder.AddImmediate(8)
		cg.storeIOByte(0x8071, 6)
		for i, addr := range []uint16{0x8072, 0x8073, 0x8074, 0x8075} {
			if err := cg.generateExpr(call.Args[i+1], 0); err != nil {
				return err
			}
			cg.storeIOByte(addr, 0)
		}
		// Value into R0, then the digit-rendering routine.
		if err := cg.generateExpr(call.Args[5], 0); err != nil {
			return err
		}
		cg.needDrawInt = true
		cg.emitHelperCall("__drawint")
		return nil
	}

	// music.play(asset) / music.play_loop(asset) / music.play_jingle(asset):
	// the argument is a music asset name (resolved at compile time against
	// cg.musicAssets), not a runtime expression — it can't go through the
	// generic arg-evaluation loop below the same way text.draw's
	// string-literal argument can't.
	if funcName == "music.play" || funcName == "music.play_loop" || funcName == "music.play_jingle" {
		if len(call.Args) != 1 {
			return fmt.Errorf("%s requires 1 argument (a music asset)", funcName)
		}
		assetIdent, ok := call.Args[0].(*IdentExpr)
		if !ok {
			return fmt.Errorf("%s: argument must be a music asset name", funcName)
		}
		asset, ok := cg.musicAssets[assetIdent.Name]
		if !ok {
			return fmt.Errorf("%s: %q is not a declared music asset", funcName, assetIdent.Name)
		}
		mode := uint16(1)
		switch funcName {
		case "music.play_loop":
			mode = 2
		case "music.play_jingle":
			mode = 3
		}
		store := func(addr uint16, v uint16) {
			cg.hMovImm(6, v)
			cg.hStore16(addr, 6)
		}
		if funcName == "music.play_jingle" {
			// Stash the currently-playing song's state (whatever it is,
			// including "nothing playing") so __musicadvance can restore it
			// once the jingle reaches its end.
			cg.copyWRAMSlot(musicSavedActiveSlot, musicActiveSlot)
			cg.copyWRAMSlot(musicSavedFrameSlot, musicFrameSlot)
			cg.copyWRAMSlot(musicSavedCountSlot, musicCountSlot)
			cg.copyWRAMSlot(musicSavedCBankSlot, musicCBankSlot)
			cg.copyWRAMSlot(musicSavedCOffSlot, musicCOffSlot)
			cg.copyWRAMSlot(musicSavedPBankSlot, musicPBankSlot)
			cg.copyWRAMSlot(musicSavedPOffSlot, musicPOffSlot)
		}
		store(musicActiveSlot, mode)
		store(musicFrameSlot, 0)
		store(musicCountSlot, uint16(asset.FrameCount))
		store(musicCBankSlot, uint16(asset.CountsBank))
		store(musicCOffSlot, asset.CountsOff)
		store(musicPBankSlot, uint16(asset.PtrBank))
		store(musicPOffSlot, asset.PtrOff)
		return nil
	}

	if funcName == "" {
		return fmt.Errorf("cannot determine function name in call")
	}

	// User-defined function calls are capped to R0-R5: R6/R7 are reserved
	// scratch for far-call bank/offset operands in multi-bank builds (see
	// the matching param cap in generateFunctionPrologue). Builtins are
	// inlined -- no CALL is ever emitted for them -- so they're free to use
	// the full R0-R7 register file (matrix_plane.fill_rect needs 7 args).
	userFn := cg.findFunction(funcName)
	argCap := 8
	if userFn != nil {
		argCap = 6
	}

	// Arguments land in R0..R(argCap-1). A later argument that is itself a
	// call or a binary expression clobbers those registers, so every argument
	// before the last such one is spilled and reloaded once they are all done.
	lastUnsafe := -1
	for i, arg := range call.Args {
		if i >= argCap {
			break
		}
		if exprNeedsSpill(arg) {
			lastUnsafe = i
		}
	}
	var spilledArgs []uint8
	for i, arg := range call.Args {
		if i >= argCap {
			return fmt.Errorf("too many arguments (max %d)", argCap)
		}
		if userFn != nil && i < len(userFn.Params) {
			if _, ok := userFn.Params[i].Type.(*ArrayType); ok {
				if err := cg.emitArrayArg(arg, uint8(i)); err != nil {
					return err
				}
				if i < lastUnsafe {
					if err := cg.spillPush(uint8(i)); err != nil {
						return err
					}
					spilledArgs = append(spilledArgs, uint8(i))
				}
				continue
			}
		}
		if err := cg.generateExpr(arg, uint8(i)); err != nil {
			return err
		}
		if i < lastUnsafe {
			if err := cg.spillPush(uint8(i)); err != nil {
				return err
			}
			spilledArgs = append(spilledArgs, uint8(i))
		}
	}
	for i := len(spilledArgs) - 1; i >= 0; i-- {
		cg.spillPop(spilledArgs[i])
	}

	// Try built-in functions first.
	// Propagate real builtin failures instead of masking them as "unknown function".
	if err := cg.generateBuiltinCall(funcName, call.Args, destReg); err == nil {
		return nil
	} else if !errors.Is(err, errUnknownBuiltin) {
		return err
	}

	// Check if it's a user-defined function -- emit CALL instruction.
	if fn := cg.findFunction(funcName); fn != nil {
		// Arguments are already evaluated into R0..R(n-1) by the loop above.
		// Emit CALL (or far-CALL, in wide-call mode) with placeholder
		// operands; patched once every function's final address is known.
		cg.emitCallPatch(funcName)
		// Return value is in R0; move to destReg if needed.
		if destReg != 0 {
			cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 0))
		}
		return nil
	}

	// Handle struct initialization like Sprite() or Vec2()
	if layout, isStruct := cg.structLayoutFor(funcName); isStruct {
		// Struct initialization creates a zero-initialized struct.
		structSize := layout.Size
		if structSize%2 != 0 {
			return fmt.Errorf("struct %s has odd total size (%d bytes); only whole-word struct layouts are supported", funcName, structSize)
		}
		stackAddr, err := cg.allocateStack(structSize, "struct "+funcName)
		if err != nil {
			return err
		}

		// Initialize struct to zero
		// Zero out struct memory in 16-bit chunks (current built-ins are even-sized).
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #stackAddr
		cg.builder.AddImmediate(stackAddr)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0
		cg.builder.AddImmediate(0)
		for off := uint16(0); off < structSize; off += 2 {
			cg.builder.AddInstruction(rom.EncodeMOV(10, 7, 6)) // MOV [R7+off], R6 (16-bit store)
			cg.builder.AddImmediate(off)
		}

		// Return struct address in destReg
		cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
		cg.builder.AddImmediate(stackAddr)

		// Note: The caller (VarDecl) will track this variable
		// Struct address is returned in destReg
		return nil
	}

	return fmt.Errorf("unknown function: %s", funcName)
}

func (cg *CodeGenerator) findFunction(name string) *FunctionDecl {
	for _, fn := range cg.program.Functions {
		if fn.Name == name {
			return fn
		}
	}
	return nil
}

func (cg *CodeGenerator) generateBuiltinCall(name string, args []Expr, destReg uint8) error {
	// Register contract for every builtin in this file and the
	// codegen_builtin_*.go handlers:
	//
	// User calls pass arguments in R0-R5 and return in R0. R6 and R7 are
	// scratch for far-call bank/offset and for address arithmetic. Builtins
	// are inlined, so they may use R0-R7, but a nested generateExpr clobbers
	// every register. Spill a live value before evaluating another expression.
	//
	// A handler returns errUnknownBuiltin when the name is not one of its
	// own. Any other error is the failure for that builtin.
	for _, h := range []func(string, []Expr, uint8) error{
		cg.builtinFlow,
		cg.builtinSprite,
		cg.builtinGfx,
		cg.builtinInput,
		cg.builtinApu,
		cg.builtinMem,
		cg.builtinMusic,
		cg.builtinBG,
		cg.builtinMatrixPlane,
		cg.builtinRaster,
		cg.builtinMatrix,
	} {
		err := h(name, args, destReg)
		if !errors.Is(err, errUnknownBuiltin) {
			return err
		}
	}
	return fmt.Errorf("%w: %s", errUnknownBuiltin, name)
}

// generateRuntimeTileLoadDispatch generates runtime asset-ID dispatch for gfx.load_tiles.
// Assumes R0=assetID, R1=base tile index.
func (cg *CodeGenerator) generateRuntimeTileLoadDispatch(destReg uint8) error {
	// Preserve runtime inputs across case probes.
	cg.builder.AddInstruction(rom.EncodeMOV(0, 2, 1)) // MOV R2, R1 (base)
	cg.builder.AddInstruction(rom.EncodeMOV(0, 3, 0)) // MOV R3, R0 (asset id)

	jumpToEnd := make([]int, 0, len(cg.program.Assets))

	for _, asset := range cg.program.Assets {
		assetID, ok := cg.assetIDs[asset.Name]
		if !ok {
			return fmt.Errorf("missing runtime asset ID for %s", asset.Name)
		}

		// if R3 != assetID -> skip this case
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #assetID
		cg.builder.AddImmediate(assetID)
		cg.builder.AddInstruction(rom.EncodeCMP(0, 3, 4)) // CMP R3, R4
		cg.builder.AddInstruction(rom.EncodeBNE())
		skipOffsetPos := cg.builder.GetCodeLength()
		cg.builder.AddImmediate(0) // patch after case body

		// Matched case: restore base to R1 and inline-load the chosen asset.
		cg.builder.AddInstruction(rom.EncodeMOV(0, 1, 2)) // MOV R1, R2
		if err := cg.generateInlineTileLoadFromBaseReg(asset, 1, destReg); err != nil {
			return err
		}

		// Jump to common end after handling this match.
		cg.builder.AddInstruction(rom.EncodeJMP())
		jumpPos := cg.builder.GetCodeLength()
		cg.builder.AddImmediate(0)
		jumpToEnd = append(jumpToEnd, jumpPos)

		nextCasePC := uint16(cg.builder.GetCodeLength() * 2)
		currentPC := uint16(skipOffsetPos * 2)
		offset := cg.calcBranchOffset(currentPC, nextCasePC)
		cg.builder.SetImmediateAt(skipOffsetPos, uint16(offset))
	}

	// Unknown runtime ID: leave VRAM unchanged and return base.
	cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 2)) // MOV R{destReg}, R2

	endPC := uint16(cg.builder.GetCodeLength() * 2)
	for _, jumpPos := range jumpToEnd {
		currentPC := uint16(jumpPos * 2)
		offset := cg.calcBranchOffset(currentPC, endPC)
		cg.builder.SetImmediateAt(jumpPos, uint16(offset))
	}

	return nil
}

func (cg *CodeGenerator) generateRuntimeTilemapLoadDispatch(destReg uint8) error {
	// Runtime asset-ID dispatch for bg.load_tilemap(asset, layer).
	cg.builder.AddInstruction(rom.EncodeMOV(0, 2, 1)) // R2 = layer (preserve)

	tilemapAssets := make([]*AssetDecl, 0)
	for _, asset := range cg.program.Assets {
		if asset.Type == "tilemap" {
			tilemapAssets = append(tilemapAssets, asset)
		}
	}
	if len(tilemapAssets) == 0 {
		return fmt.Errorf("bg.load_tilemap requires at least one tilemap asset")
	}

	jumpToEnd := make([]int, 0, len(tilemapAssets))
	for _, asset := range tilemapAssets {
		value, ok := cg.assetIDs[asset.Name]
		if !ok {
			continue
		}

		cg.builder.AddInstruction(rom.EncodeMOV(1, 3, 0))
		cg.builder.AddImmediate(value)
		cg.builder.AddInstruction(rom.EncodeCMP(0, 0, 3))
		cg.builder.AddInstruction(rom.EncodeBNE())
		skipOffsetPos := cg.builder.GetCodeLength()
		cg.builder.AddImmediate(0)

		cg.builder.AddInstruction(rom.EncodeMOV(0, 1, 2)) // restore layer to R1
		if err := cg.generateInlineTilemapLoadFromLayerReg(asset, 1, destReg); err != nil {
			return err
		}

		cg.builder.AddInstruction(rom.EncodeJMP())
		jumpPos := cg.builder.GetCodeLength()
		cg.builder.AddImmediate(0)
		jumpToEnd = append(jumpToEnd, jumpPos)

		nextCasePC := uint16(cg.builder.GetCodeLength() * 2)
		currentPC := uint16(skipOffsetPos * 2)
		offset := cg.calcBranchOffset(currentPC, nextCasePC)
		cg.builder.SetImmediateAt(skipOffsetPos, uint16(offset))
	}

	// Unknown runtime ID: return 0.
	cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
	cg.builder.AddImmediate(0)

	endPC := uint16(cg.builder.GetCodeLength() * 2)
	for _, jumpPos := range jumpToEnd {
		currentPC := uint16(jumpPos * 2)
		offset := cg.calcBranchOffset(currentPC, endPC)
		cg.builder.SetImmediateAt(jumpPos, uint16(offset))
	}

	return nil
}

// generateRuntimeMatrixPlaneTileLoadDispatch generates runtime asset-ID dispatch
// for matrix_plane.load_tiles. Assumes R0=assetID, R1=channel, R2=base tile index.
func (cg *CodeGenerator) generateRuntimeMatrixPlaneTileLoadDispatch(destReg uint8) error {
	cg.builder.AddInstruction(rom.EncodeMOV(0, 3, 1)) // R3 = channel
	cg.builder.AddInstruction(rom.EncodeMOV(0, 4, 2)) // R4 = base

	jumpToEnd := make([]int, 0, len(cg.program.Assets))

	for _, asset := range cg.program.Assets {
		assetID, ok := cg.assetIDs[asset.Name]
		if !ok {
			return fmt.Errorf("missing runtime asset ID for %s", asset.Name)
		}

		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
		cg.builder.AddImmediate(assetID)
		cg.builder.AddInstruction(rom.EncodeCMP(0, 0, 5))
		cg.builder.AddInstruction(rom.EncodeBNE())
		skipOffsetPos := cg.builder.GetCodeLength()
		cg.builder.AddImmediate(0)

		cg.builder.AddInstruction(rom.EncodeMOV(0, 1, 3)) // restore channel to R1
		cg.builder.AddInstruction(rom.EncodeMOV(0, 2, 4)) // restore base to R2
		if err := cg.generateInlineMatrixPlaneTileLoadFromRegs(asset, 1, 2, destReg); err != nil {
			return err
		}

		cg.builder.AddInstruction(rom.EncodeJMP())
		jumpPos := cg.builder.GetCodeLength()
		cg.builder.AddImmediate(0)
		jumpToEnd = append(jumpToEnd, jumpPos)

		nextCasePC := uint16(cg.builder.GetCodeLength() * 2)
		currentPC := uint16(skipOffsetPos * 2)
		offset := cg.calcBranchOffset(currentPC, nextCasePC)
		cg.builder.SetImmediateAt(skipOffsetPos, uint16(offset))
	}

	cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 4)) // return base unchanged

	endPC := uint16(cg.builder.GetCodeLength() * 2)
	for _, jumpPos := range jumpToEnd {
		currentPC := uint16(jumpPos * 2)
		offset := cg.calcBranchOffset(currentPC, endPC)
		cg.builder.SetImmediateAt(jumpPos, uint16(offset))
	}
	return nil
}

func (cg *CodeGenerator) generateRuntimeMatrixPlaneTilemapLoadDispatch(destReg uint8) error {
	cg.builder.AddInstruction(rom.EncodeMOV(0, 2, 1)) // R2 = channel

	tilemapAssets := make([]*AssetDecl, 0)
	for _, asset := range cg.program.Assets {
		if asset.Type == "tilemap" {
			tilemapAssets = append(tilemapAssets, asset)
		}
	}
	if len(tilemapAssets) == 0 {
		return fmt.Errorf("matrix_plane.load_tilemap requires at least one tilemap asset")
	}

	jumpToEnd := make([]int, 0, len(tilemapAssets))
	for _, asset := range tilemapAssets {
		value, ok := cg.assetIDs[asset.Name]
		if !ok {
			continue
		}

		cg.builder.AddInstruction(rom.EncodeMOV(1, 3, 0))
		cg.builder.AddImmediate(value)
		cg.builder.AddInstruction(rom.EncodeCMP(0, 0, 3))
		cg.builder.AddInstruction(rom.EncodeBNE())
		skipOffsetPos := cg.builder.GetCodeLength()
		cg.builder.AddImmediate(0)

		cg.builder.AddInstruction(rom.EncodeMOV(0, 1, 2)) // restore channel to R1
		if err := cg.generateInlineMatrixPlaneTilemapLoadFromChannelReg(asset, 1, destReg); err != nil {
			return err
		}

		cg.builder.AddInstruction(rom.EncodeJMP())
		jumpPos := cg.builder.GetCodeLength()
		cg.builder.AddImmediate(0)
		jumpToEnd = append(jumpToEnd, jumpPos)

		nextCasePC := uint16(cg.builder.GetCodeLength() * 2)
		currentPC := uint16(skipOffsetPos * 2)
		offset := cg.calcBranchOffset(currentPC, nextCasePC)
		cg.builder.SetImmediateAt(skipOffsetPos, uint16(offset))
	}

	cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 2)) // return channel unchanged

	endPC := uint16(cg.builder.GetCodeLength() * 2)
	for _, jumpPos := range jumpToEnd {
		currentPC := uint16(jumpPos * 2)
		offset := cg.calcBranchOffset(currentPC, endPC)
		cg.builder.SetImmediateAt(jumpPos, uint16(offset))
	}
	return nil
}

// generateInlineTileLoad generates code to load tile data from an asset directly to VRAM
func (cg *CodeGenerator) generateInlineTileLoad(asset *AssetDecl, baseExpr Expr, destReg uint8) error {
	if asset.Type != "tiles8" && asset.Type != "tiles16" && asset.Type != "sprite" && asset.Type != "tileset" {
		return fmt.Errorf("gfx.load_tiles requires tile asset type, got %s", asset.Type)
	}
	// Generate base tile index (second argument)
	if err := cg.generateExpr(baseExpr, 1); err != nil {
		return err
	}
	return cg.generateInlineTileLoadFromBaseReg(asset, 1, destReg)
}

func (cg *CodeGenerator) generateInlineTileLoadFromBaseReg(asset *AssetDecl, baseReg uint8, destReg uint8) error {
	// Calculate VRAM address from the base tile index.
	cg.builder.AddInstruction(rom.EncodeMOV(0, 2, baseReg)) // MOV R2, R{baseReg} (save base)
	cg.builder.AddInstruction(rom.EncodeMOV(0, 3, 2))       // MOV R3, R2 (working copy)

	dataBytes, err := cg.inlineTileAssetBytes(asset)
	if err != nil {
		return err
	}
	bytesPayload := len(dataBytes)

	use16x16Stride := asset.Type == "tiles16" || (asset.Type == "tileset" && bytesPayload == 128)
	if use16x16Stride {
		// 16x16 tile: base * 128 = base << 7 (PPU reads sprite tile at index*128)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #7
		cg.builder.AddImmediate(7)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 3, 4))
	} else {
		// 8x8 tile: base * 32 = base << 5
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #5
		cg.builder.AddImmediate(5)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 3, 4))
	}

	// Set VRAM address low/high registers.
	cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800E
	cg.builder.AddImmediate(0x800E)
	cg.builder.AddInstruction(rom.EncodeMOV(0, 5, 3))
	cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0xFF
	cg.builder.AddImmediate(0xFF)
	cg.builder.AddInstruction(rom.EncodeAND(0, 5, 6))
	cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

	cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800F
	cg.builder.AddImmediate(0x800F)
	cg.builder.AddInstruction(rom.EncodeMOV(0, 5, 3))
	cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #8
	cg.builder.AddImmediate(8)
	cg.builder.AddInstruction(rom.EncodeSHR(0, 5, 6))
	cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

	cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8010 (VRAM_DATA)
	cg.builder.AddImmediate(0x8010)

	bytesToWrite := bytesPayload
	switch asset.Type {
	case "tiles8":
		// tiles8 maps to a single 8x8 tile payload.
		if bytesToWrite > 32 {
			bytesToWrite = 32
		}
	case "tiles16":
		// tiles16 maps to a single 16x16 tile payload.
		if bytesToWrite > 128 {
			bytesToWrite = 128
		}
	default:
		// sprite/tileset payloads may represent multiple contiguous tiles.
		// Write the full normalized payload so tools can emit larger data blocks.
	}
	for i, value := range dataBytes {
		if i >= bytesToWrite {
			break
		}
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #value
		cg.builder.AddImmediate(uint16(value))
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5))
	}

	// Return base tile index.
	cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 2))
	return nil
}

func (cg *CodeGenerator) generateInlineMatrixPlaneTileLoadFromRegs(asset *AssetDecl, channelReg, baseReg uint8, destReg uint8) error {
	if asset.Type != "tiles8" && asset.Type != "tiles16" && asset.Type != "sprite" && asset.Type != "tileset" {
		return fmt.Errorf("matrix_plane.load_tiles requires tile asset type, got %s", asset.Type)
	}
	dataBytes, err := cg.inlineTileAssetBytes(asset)
	if err != nil {
		return err
	}
	bytesPayload := len(dataBytes)
	use16x16Stride := asset.Type == "tiles16" || (asset.Type == "tileset" && bytesPayload == 128)

	// Select the plane first.
	cg.emitSelectMatrixPlane(channelReg, 3, 4)

	// Calculate pattern address from tile index.
	cg.builder.AddInstruction(rom.EncodeMOV(0, 3, baseReg)) // R3 = base
	if use16x16Stride {
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
		cg.builder.AddImmediate(7)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 3, 4))
	} else {
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
		cg.builder.AddImmediate(5)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 3, 4))
	}

	// Set pattern address registers.
	cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	cg.builder.AddImmediate(0x8085)
	cg.builder.AddInstruction(rom.EncodeMOV(0, 5, 3))
	cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
	cg.builder.AddImmediate(0x00FF)
	cg.builder.AddInstruction(rom.EncodeAND(0, 5, 6))
	cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

	cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	cg.builder.AddImmediate(0x8086)
	cg.builder.AddInstruction(rom.EncodeMOV(0, 5, 3))
	cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
	cg.builder.AddImmediate(8)
	cg.builder.AddInstruction(rom.EncodeSHR(0, 5, 6))
	cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5))

	cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	cg.builder.AddImmediate(0x8087)

	bytesToWrite := bytesPayload
	switch asset.Type {
	case "tiles8":
		if bytesToWrite > 32 {
			bytesToWrite = 32
		}
	case "tiles16":
		if bytesToWrite > 128 {
			bytesToWrite = 128
		}
	}
	for i, value := range dataBytes {
		if i >= bytesToWrite {
			break
		}
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
		cg.builder.AddImmediate(uint16(value))
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5))
	}

	cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, baseReg))
	return nil
}

func (cg *CodeGenerator) generateInlineMatrixPlaneTilemapLoadFromChannelReg(asset *AssetDecl, channelReg uint8, destReg uint8) error {
	if asset.Type != "tilemap" {
		return fmt.Errorf("matrix_plane.load_tilemap requires tilemap asset type, got %s", asset.Type)
	}
	dataBytes, err := cg.inlineTilemapAssetBytes(asset)
	if err != nil {
		return err
	}

	cg.emitSelectMatrixPlane(channelReg, 3, 4)

	cg.builder.AddInstruction(rom.EncodeMOV(1, 3, 0))
	cg.builder.AddImmediate(0x8082)
	cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	cg.builder.AddImmediate(0)
	cg.builder.AddInstruction(rom.EncodeMOV(3, 3, 4))
	cg.builder.AddInstruction(rom.EncodeMOV(1, 3, 0))
	cg.builder.AddImmediate(0x8083)
	cg.builder.AddInstruction(rom.EncodeMOV(3, 3, 4))
	cg.builder.AddInstruction(rom.EncodeMOV(1, 3, 0))
	cg.builder.AddImmediate(0x8084)
	for _, value := range dataBytes {
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
		cg.builder.AddImmediate(uint16(value))
		cg.builder.AddInstruction(rom.EncodeMOV(3, 3, 4))
	}

	cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, channelReg))
	return nil
}

func (cg *CodeGenerator) generateMatrixPlaneFillRect() error {
	xAddr, err := cg.allocateStack(2, "matrix plane fill rect x start")
	if err != nil {
		return err
	}
	yAddr, err := cg.allocateStack(2, "matrix plane fill rect y current")
	if err != nil {
		return err
	}
	wAddr, err := cg.allocateStack(2, "matrix plane fill rect width")
	if err != nil {
		return err
	}
	hAddr, err := cg.allocateStack(2, "matrix plane fill rect height")
	if err != nil {
		return err
	}
	tileAddr, err := cg.allocateStack(2, "matrix plane fill rect tile")
	if err != nil {
		return err
	}
	attrAddr, err := cg.allocateStack(2, "matrix plane fill rect attr")
	if err != nil {
		return err
	}
	shiftAddr, err := cg.allocateStack(2, "matrix plane fill rect shift")
	if err != nil {
		return err
	}

	// Spill arguments needed across the nested loops.
	cg.emitStoreStackWord(1, xAddr)
	cg.emitStoreStackWord(2, yAddr)
	cg.emitStoreStackWord(3, wAddr)
	cg.emitStoreStackWord(4, hAddr)
	cg.emitStoreStackWord(5, tileAddr)
	cg.emitStoreStackWord(6, attrAddr)

	cg.emitSelectMatrixPlane(0, 6, 5)
	cg.emitSelectedMatrixPlaneShift(5, 6, 4)
	cg.emitStoreStackWord(5, shiftAddr)

	rowLoopStart := cg.builder.GetCodeLength()
	cg.emitLoadStackWord(4, hAddr) // R4 = remaining rows
	cg.builder.AddInstruction(rom.EncodeMOV(1, 0, 0))
	cg.builder.AddImmediate(0)
	cg.builder.AddInstruction(rom.EncodeCMP(0, 4, 0))
	cg.builder.AddInstruction(rom.EncodeBEQ())
	rowLoopEndPos := cg.builder.GetCodeLength()
	cg.builder.AddImmediate(0)

	cg.emitLoadStackWord(1, yAddr) // R1 = current y
	cg.emitLoadStackWord(2, xAddr) // R2 = current x
	cg.emitLoadStackWord(3, wAddr) // R3 = remaining cols

	colLoopStart := cg.builder.GetCodeLength()
	cg.builder.AddInstruction(rom.EncodeMOV(1, 0, 0))
	cg.builder.AddImmediate(0)
	cg.builder.AddInstruction(rom.EncodeCMP(0, 3, 0))
	cg.builder.AddInstruction(rom.EncodeBEQ())
	colLoopEndPos := cg.builder.GetCodeLength()
	cg.builder.AddImmediate(0)

	// offset = ((y << shift) + x) << 1
	cg.builder.AddInstruction(rom.EncodeMOV(0, 4, 1)) // R4 = y
	cg.emitLoadStackWord(5, shiftAddr)
	cg.builder.AddInstruction(rom.EncodeSHL(0, 4, 5))
	cg.builder.AddInstruction(rom.EncodeADD(0, 4, 2))
	cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	cg.builder.AddImmediate(1)
	cg.builder.AddInstruction(rom.EncodeSHL(0, 4, 5))

	// Address low/high
	cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	cg.builder.AddImmediate(0x8082)
	cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 4))
	cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
	cg.builder.AddImmediate(0x00FF)
	cg.builder.AddInstruction(rom.EncodeAND(0, 6, 7))
	cg.builder.AddInstruction(rom.EncodeMOV(3, 5, 6))
	cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	cg.builder.AddImmediate(0x8083)
	cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 4))
	cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
	cg.builder.AddImmediate(8)
	cg.builder.AddInstruction(rom.EncodeSHR(0, 6, 7))
	cg.builder.AddInstruction(rom.EncodeMOV(3, 5, 6))

	// Data writes
	cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	cg.builder.AddImmediate(0x8084)
	cg.emitLoadStackWord(6, tileAddr)
	cg.builder.AddInstruction(rom.EncodeMOV(3, 5, 6))
	cg.emitLoadStackWord(6, attrAddr)
	cg.builder.AddInstruction(rom.EncodeMOV(3, 5, 6))

	// x++, remaining cols--
	cg.builder.AddInstruction(rom.EncodeMOV(1, 0, 0))
	cg.builder.AddImmediate(1)
	cg.builder.AddInstruction(rom.EncodeADD(0, 2, 0))
	cg.builder.AddInstruction(rom.EncodeSUB(0, 3, 0))
	cg.builder.AddInstruction(rom.EncodeJMP())
	colLoopJumpPos := cg.builder.GetCodeLength()
	cg.builder.AddImmediate(uint16(cg.calcBranchOffset(uint16(colLoopJumpPos*2), uint16(colLoopStart*2))))

	colLoopEndPC := uint16(cg.builder.GetCodeLength() * 2)
	cg.builder.SetImmediateAt(colLoopEndPos, uint16(cg.calcBranchOffset(uint16(colLoopEndPos*2), colLoopEndPC)))

	// y++, remaining rows--
	cg.builder.AddInstruction(rom.EncodeMOV(1, 0, 0))
	cg.builder.AddImmediate(1)
	cg.builder.AddInstruction(rom.EncodeADD(0, 1, 0))
	cg.emitStoreStackWord(1, yAddr)
	cg.emitLoadStackWord(4, hAddr)
	cg.builder.AddInstruction(rom.EncodeSUB(0, 4, 0))
	cg.emitStoreStackWord(4, hAddr)
	cg.builder.AddInstruction(rom.EncodeJMP())
	rowLoopJumpPos := cg.builder.GetCodeLength()
	cg.builder.AddImmediate(uint16(cg.calcBranchOffset(uint16(rowLoopJumpPos*2), uint16(rowLoopStart*2))))

	rowLoopEndPC := uint16(cg.builder.GetCodeLength() * 2)
	cg.builder.SetImmediateAt(rowLoopEndPos, uint16(cg.calcBranchOffset(uint16(rowLoopEndPos*2), rowLoopEndPC)))

	return nil
}

func (cg *CodeGenerator) generateMatrixPlaneClear() error {
	// Args: R0 = channel, R1 = tile, R2 = attr
	cg.emitSelectMatrixPlane(0, 6, 5)
	cg.emitSelectedMatrixPlaneShift(5, 6, 4) // shift in R5

	// total entries based on selected plane size:
	// 32x32 -> 1024, 64x64 -> 4096, 128x128 -> 16384
	cg.builder.AddInstruction(rom.EncodeMOV(1, 3, 0))
	cg.builder.AddImmediate(1024)
	cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	cg.builder.AddImmediate(6)
	cg.builder.AddInstruction(rom.EncodeCMP(0, 5, 4))
	cg.builder.AddInstruction(rom.EncodeBNE())
	not64Pos := cg.builder.GetCodeLength()
	cg.builder.AddImmediate(0)
	cg.builder.AddInstruction(rom.EncodeMOV(1, 3, 0))
	cg.builder.AddImmediate(4096)
	cg.builder.AddInstruction(rom.EncodeJMP())
	after64Pos := cg.builder.GetCodeLength()
	cg.builder.AddImmediate(0)
	not64PC := uint16(cg.builder.GetCodeLength() * 2)
	cg.builder.SetImmediateAt(not64Pos, uint16(cg.calcBranchOffset(uint16(not64Pos*2), not64PC)))

	cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	cg.builder.AddImmediate(7)
	cg.builder.AddInstruction(rom.EncodeCMP(0, 5, 4))
	cg.builder.AddInstruction(rom.EncodeBNE())
	not128Pos := cg.builder.GetCodeLength()
	cg.builder.AddImmediate(0)
	cg.builder.AddInstruction(rom.EncodeMOV(1, 3, 0))
	cg.builder.AddImmediate(16384)
	not128PC := uint16(cg.builder.GetCodeLength() * 2)
	cg.builder.SetImmediateAt(not128Pos, uint16(cg.calcBranchOffset(uint16(not128Pos*2), not128PC)))

	after64PC := uint16(cg.builder.GetCodeLength() * 2)
	cg.builder.SetImmediateAt(after64Pos, uint16(cg.calcBranchOffset(uint16(after64Pos*2), after64PC)))

	// zero tilemap address
	cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	cg.builder.AddImmediate(0x8082)
	cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
	cg.builder.AddImmediate(0)
	cg.builder.AddInstruction(rom.EncodeMOV(3, 5, 6))
	cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	cg.builder.AddImmediate(0x8083)
	cg.builder.AddInstruction(rom.EncodeMOV(3, 5, 6))

	cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
	cg.builder.AddImmediate(0x8084)
	loopStart := cg.builder.GetCodeLength()
	cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
	cg.builder.AddImmediate(0)
	cg.builder.AddInstruction(rom.EncodeCMP(0, 3, 6))
	cg.builder.AddInstruction(rom.EncodeBEQ())
	loopEndPos := cg.builder.GetCodeLength()
	cg.builder.AddImmediate(0)
	cg.builder.AddInstruction(rom.EncodeMOV(3, 5, 1))
	cg.builder.AddInstruction(rom.EncodeMOV(3, 5, 2))
	cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
	cg.builder.AddImmediate(1)
	cg.builder.AddInstruction(rom.EncodeSUB(0, 3, 6))
	cg.builder.AddInstruction(rom.EncodeJMP())
	loopJumpPos := cg.builder.GetCodeLength()
	cg.builder.AddImmediate(uint16(cg.calcBranchOffset(uint16(loopJumpPos*2), uint16(loopStart*2))))
	loopEndPC := uint16(cg.builder.GetCodeLength() * 2)
	cg.builder.SetImmediateAt(loopEndPos, uint16(cg.calcBranchOffset(uint16(loopEndPos*2), loopEndPC)))
	return nil
}

func (cg *CodeGenerator) generateInlineTilemapLoad(asset *AssetDecl, layerExpr Expr, destReg uint8) error {
	if asset.Type != "tilemap" {
		return fmt.Errorf("bg.load_tilemap requires tilemap asset type, got %s", asset.Type)
	}
	if err := cg.generateExpr(layerExpr, 1); err != nil {
		return err
	}
	return cg.generateInlineTilemapLoadFromLayerReg(asset, 1, destReg)
}

func (cg *CodeGenerator) generateInlineTilemapLoadFromLayerReg(asset *AssetDecl, layerReg uint8, destReg uint8) error {
	if asset.Type != "tilemap" {
		return fmt.Errorf("bg.load_tilemap requires tilemap asset type, got %s", asset.Type)
	}
	dataBytes, err := cg.inlineTilemapAssetBytes(asset)
	if err != nil {
		return err
	}

	// Resolve the configured tilemap base for the selected layer into R5.
	bgTilemapAddrs := []uint16{0x8077, 0x8079, 0x807B, 0x807D}
	jumpToEnd := make([]int, 0, 4)
	for i, addr := range bgTilemapAddrs {
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
		cg.builder.AddImmediate(uint16(i))
		cg.builder.AddInstruction(rom.EncodeCMP(0, layerReg, 6))
		cg.builder.AddInstruction(rom.EncodeBNE())
		skipPos := cg.builder.GetCodeLength()
		cg.builder.AddImmediate(0)

		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
		cg.builder.AddImmediate(addr)
		cg.builder.AddInstruction(rom.EncodeMOV(2, 5, 6))
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
		cg.builder.AddImmediate(addr + 1)
		cg.builder.AddInstruction(rom.EncodeMOV(2, 6, 6))
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
		cg.builder.AddImmediate(8)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 6, 7))
		cg.builder.AddInstruction(rom.EncodeOR(0, 5, 6))
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
		cg.builder.AddImmediate(0)
		cg.builder.AddInstruction(rom.EncodeCMP(0, 5, 7))
		cg.builder.AddInstruction(rom.EncodeBNE())
		baseReadyPos := cg.builder.GetCodeLength()
		cg.builder.AddImmediate(0)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0))
		cg.builder.AddImmediate(0x4000)
		baseReadyPC := uint16(cg.builder.GetCodeLength() * 2)
		cg.builder.SetImmediateAt(baseReadyPos, uint16(cg.calcBranchOffset(uint16(baseReadyPos*2), baseReadyPC)))

		cg.builder.AddInstruction(rom.EncodeJMP())
		jumpPos := cg.builder.GetCodeLength()
		cg.builder.AddImmediate(0)
		jumpToEnd = append(jumpToEnd, jumpPos)

		nextPC := uint16(cg.builder.GetCodeLength() * 2)
		skipPC := uint16(skipPos * 2)
		cg.builder.SetImmediateAt(skipPos, uint16(cg.calcBranchOffset(skipPC, nextPC)))
	}
	endPC := uint16(cg.builder.GetCodeLength() * 2)
	for _, jp := range jumpToEnd {
		jpPC := uint16(jp * 2)
		cg.builder.SetImmediateAt(jp, uint16(cg.calcBranchOffset(jpPC, endPC)))
	}

	// Program VRAM address from resolved base.
	cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
	cg.builder.AddImmediate(0x800E)
	cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 5))
	cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	cg.builder.AddImmediate(0x00FF)
	cg.builder.AddInstruction(rom.EncodeAND(0, 7, 4))
	cg.builder.AddInstruction(rom.EncodeMOV(3, 6, 7))

	cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
	cg.builder.AddImmediate(0x800F)
	cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 5))
	cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0))
	cg.builder.AddImmediate(8)
	cg.builder.AddInstruction(rom.EncodeSHR(0, 7, 4))
	cg.builder.AddInstruction(rom.EncodeMOV(3, 6, 7))

	cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
	cg.builder.AddImmediate(0x8010)
	for _, value := range dataBytes {
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
		cg.builder.AddImmediate(uint16(value))
		cg.builder.AddInstruction(rom.EncodeMOV(3, 6, 7))
	}

	// Return the tilemap base used for the transfer.
	cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 5))
	return nil
}

func (cg *CodeGenerator) inlineTileAssetBytes(asset *AssetDecl) ([]byte, error) {
	if norm, ok := cg.normalizedAssets[asset.Name]; ok {
		return norm.Data, nil
	}
	// Fallback for direct codegen use outside the compiler pipeline.
	if asset.Encoding != "hex" {
		return nil, fmt.Errorf("inline tile load requires normalized asset data for %s encoding", asset.Encoding)
	}
	return decodeHexAssetData(asset.Data)
}

func (cg *CodeGenerator) inlineTilemapAssetBytes(asset *AssetDecl) ([]byte, error) {
	if norm, ok := cg.normalizedAssets[asset.Name]; ok {
		return norm.Data, nil
	}
	switch asset.Encoding {
	case "hex":
		return decodeHexAssetData(asset.Data)
	case "b64":
		return decodeBase64AssetData(asset.Data)
	default:
		return nil, fmt.Errorf("inline tilemap load requires normalized asset data for %s encoding", asset.Encoding)
	}
}

func (cg *CodeGenerator) emitLoadMMIO8(destReg uint8, addr uint16) {
	cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
	cg.builder.AddImmediate(addr)
	cg.builder.AddInstruction(rom.EncodeMOV(2, destReg, destReg))
}

func (cg *CodeGenerator) emitLoadStackWord(destReg uint8, addr uint16) {
	cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
	cg.builder.AddImmediate(addr)
	cg.builder.AddInstruction(rom.EncodeMOV(2, destReg, 7))
}

func (cg *CodeGenerator) emitStoreStackWord(srcReg uint8, addr uint16) {
	cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
	cg.builder.AddImmediate(addr)
	cg.builder.AddInstruction(rom.EncodeMOV(3, 7, srcReg))
}

func (cg *CodeGenerator) emitSelectMatrixPlane(channelReg, addrReg, tmpReg uint8) {
	cg.builder.AddInstruction(rom.EncodeMOV(1, addrReg, 0))
	cg.builder.AddImmediate(0x8080)
	cg.builder.AddInstruction(rom.EncodeMOV(0, tmpReg, channelReg))
	cg.builder.AddInstruction(rom.EncodeMOV(1, 0, 0))
	cg.builder.AddImmediate(0x03)
	cg.builder.AddInstruction(rom.EncodeAND(0, tmpReg, 0))
	cg.builder.AddInstruction(rom.EncodeMOV(3, addrReg, tmpReg))
}

func (cg *CodeGenerator) emitSelectedMatrixPlaneShift(shiftReg, controlReg, scratchReg uint8) {
	cg.emitLoadMMIO8(controlReg, 0x8081)
	cg.builder.AddInstruction(rom.EncodeMOV(1, scratchReg, 0))
	cg.builder.AddImmediate(1)
	cg.builder.AddInstruction(rom.EncodeSHR(0, controlReg, scratchReg))
	cg.builder.AddInstruction(rom.EncodeMOV(1, scratchReg, 0))
	cg.builder.AddImmediate(0x03)
	cg.builder.AddInstruction(rom.EncodeAND(0, controlReg, scratchReg))
	cg.builder.AddInstruction(rom.EncodeMOV(1, shiftReg, 0))
	cg.builder.AddImmediate(5)

	cg.builder.AddInstruction(rom.EncodeMOV(1, scratchReg, 0))
	cg.builder.AddImmediate(1)
	cg.builder.AddInstruction(rom.EncodeCMP(0, controlReg, scratchReg))
	cg.builder.AddInstruction(rom.EncodeBNE())
	not64Pos := cg.builder.GetCodeLength()
	cg.builder.AddImmediate(0)
	cg.builder.AddInstruction(rom.EncodeMOV(1, shiftReg, 0))
	cg.builder.AddImmediate(6)
	cg.builder.AddInstruction(rom.EncodeJMP())
	after64Pos := cg.builder.GetCodeLength()
	cg.builder.AddImmediate(0)
	not64PC := uint16(cg.builder.GetCodeLength() * 2)
	cg.builder.SetImmediateAt(not64Pos, uint16(cg.calcBranchOffset(uint16(not64Pos*2), not64PC)))

	cg.builder.AddInstruction(rom.EncodeMOV(1, scratchReg, 0))
	cg.builder.AddImmediate(2)
	cg.builder.AddInstruction(rom.EncodeCMP(0, controlReg, scratchReg))
	cg.builder.AddInstruction(rom.EncodeBNE())
	not128Pos := cg.builder.GetCodeLength()
	cg.builder.AddImmediate(0)
	cg.builder.AddInstruction(rom.EncodeMOV(1, shiftReg, 0))
	cg.builder.AddImmediate(7)
	not128PC := uint16(cg.builder.GetCodeLength() * 2)
	cg.builder.SetImmediateAt(not128Pos, uint16(cg.calcBranchOffset(uint16(not128Pos*2), not128PC)))

	after64PC := uint16(cg.builder.GetCodeLength() * 2)
	cg.builder.SetImmediateAt(after64Pos, uint16(cg.calcBranchOffset(uint16(after64Pos*2), after64PC)))
}

func (cg *CodeGenerator) emitMaybeAddShiftedScanlineIfMMIOFlag(mmioAddr, mask, shiftImm uint16, scanlineReg, destReg, scratchReg uint8) {
	cg.emitLoadMMIO8(scratchReg, mmioAddr)
	cg.builder.AddInstruction(rom.EncodeAND(1, scratchReg, 0))
	cg.builder.AddImmediate(mask)
	cg.builder.AddInstruction(rom.EncodeBEQ())
	skipPos := cg.builder.GetCodeLength()
	cg.builder.AddImmediate(0)

	cg.builder.AddInstruction(rom.EncodeMOV(0, scratchReg, scanlineReg))
	if shiftImm != 0 {
		cg.builder.AddInstruction(rom.EncodeSHL(1, scratchReg, 0))
		cg.builder.AddImmediate(shiftImm)
	}
	cg.builder.AddInstruction(rom.EncodeADD(0, destReg, scratchReg))

	skipPC := uint16(skipPos * 2)
	nextPC := uint16(cg.builder.GetCodeLength() * 2)
	cg.builder.SetImmediateAt(skipPos, uint16(cg.calcBranchOffset(skipPC, nextPC)))
}

func (cg *CodeGenerator) emitMaybeAddImmediateIfMMIOFlag(mmioAddr, mask, addImm uint16, destReg, scratchReg uint8) {
	cg.emitLoadMMIO8(scratchReg, mmioAddr)
	cg.builder.AddInstruction(rom.EncodeAND(1, scratchReg, 0))
	cg.builder.AddImmediate(mask)
	cg.builder.AddInstruction(rom.EncodeBEQ())
	skipPos := cg.builder.GetCodeLength()
	cg.builder.AddImmediate(0)

	cg.builder.AddInstruction(rom.EncodeADD(1, destReg, 0))
	cg.builder.AddImmediate(addImm)

	skipPC := uint16(skipPos * 2)
	nextPC := uint16(cg.builder.GetCodeLength() * 2)
	cg.builder.SetImmediateAt(skipPos, uint16(cg.calcBranchOffset(skipPC, nextPC)))
}

func (cg *CodeGenerator) emitComputeRasterScanlineBase(scanlineReg, destReg, scratchReg uint8) {
	cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, scanlineReg))
	cg.builder.AddInstruction(rom.EncodeSHL(1, destReg, 0))
	cg.builder.AddImmediate(6)
	cg.emitMaybeAddShiftedScanlineIfMMIOFlag(0x805D, 0x20, 2, scanlineReg, destReg, scratchReg)
	cg.emitMaybeAddShiftedScanlineIfMMIOFlag(0x805D, 0x40, 2, scanlineReg, destReg, scratchReg)
	cg.emitMaybeAddShiftedScanlineIfMMIOFlag(0x805D, 0x80, 3, scanlineReg, destReg, scratchReg)
	cg.emitMaybeAddShiftedScanlineIfMMIOFlag(0x807F, 0x01, 2, scanlineReg, destReg, scratchReg)

	cg.emitLoadMMIO8(scratchReg, 0x805E)
	cg.builder.AddInstruction(rom.EncodeADD(0, destReg, scratchReg))
	cg.emitLoadMMIO8(scratchReg, 0x805F)
	cg.builder.AddInstruction(rom.EncodeSHL(1, scratchReg, 0))
	cg.builder.AddImmediate(8)
	cg.builder.AddInstruction(rom.EncodeADD(0, destReg, scratchReg))
}

func (cg *CodeGenerator) emitWriteVRAM16AtAddrReg(addrReg, valueReg, scratchReg, ioReg uint8) {
	cg.builder.AddInstruction(rom.EncodeMOV(1, ioReg, 0))
	cg.builder.AddImmediate(0x800E)
	cg.builder.AddInstruction(rom.EncodeMOV(0, scratchReg, addrReg))
	cg.builder.AddInstruction(rom.EncodeAND(1, scratchReg, 0))
	cg.builder.AddImmediate(0x00FF)
	cg.builder.AddInstruction(rom.EncodeMOV(3, ioReg, scratchReg))

	cg.builder.AddInstruction(rom.EncodeMOV(1, ioReg, 0))
	cg.builder.AddImmediate(0x800F)
	cg.builder.AddInstruction(rom.EncodeMOV(0, scratchReg, addrReg))
	cg.builder.AddInstruction(rom.EncodeSHR(1, scratchReg, 0))
	cg.builder.AddImmediate(8)
	cg.builder.AddInstruction(rom.EncodeMOV(3, ioReg, scratchReg))

	cg.builder.AddInstruction(rom.EncodeMOV(1, ioReg, 0))
	cg.builder.AddImmediate(0x8010)
	cg.builder.AddInstruction(rom.EncodeMOV(0, scratchReg, valueReg))
	cg.builder.AddInstruction(rom.EncodeAND(1, scratchReg, 0))
	cg.builder.AddImmediate(0x00FF)
	cg.builder.AddInstruction(rom.EncodeMOV(3, ioReg, scratchReg))
	cg.builder.AddInstruction(rom.EncodeMOV(0, scratchReg, valueReg))
	cg.builder.AddInstruction(rom.EncodeSHR(1, scratchReg, 0))
	cg.builder.AddImmediate(8)
	cg.builder.AddInstruction(rom.EncodeMOV(3, ioReg, scratchReg))
}

func (cg *CodeGenerator) emitWriteVRAM8AtAddrReg(addrReg, valueReg, scratchReg, ioReg uint8) {
	cg.builder.AddInstruction(rom.EncodeMOV(1, ioReg, 0))
	cg.builder.AddImmediate(0x800E)
	cg.builder.AddInstruction(rom.EncodeMOV(0, scratchReg, addrReg))
	cg.builder.AddInstruction(rom.EncodeAND(1, scratchReg, 0))
	cg.builder.AddImmediate(0x00FF)
	cg.builder.AddInstruction(rom.EncodeMOV(3, ioReg, scratchReg))

	cg.builder.AddInstruction(rom.EncodeMOV(1, ioReg, 0))
	cg.builder.AddImmediate(0x800F)
	cg.builder.AddInstruction(rom.EncodeMOV(0, scratchReg, addrReg))
	cg.builder.AddInstruction(rom.EncodeSHR(1, scratchReg, 0))
	cg.builder.AddImmediate(8)
	cg.builder.AddInstruction(rom.EncodeMOV(3, ioReg, scratchReg))

	cg.builder.AddInstruction(rom.EncodeMOV(1, ioReg, 0))
	cg.builder.AddImmediate(0x8010)
	cg.builder.AddInstruction(rom.EncodeMOV(0, scratchReg, valueReg))
	cg.builder.AddInstruction(rom.EncodeAND(1, scratchReg, 0))
	cg.builder.AddImmediate(0x00FF)
	cg.builder.AddInstruction(rom.EncodeMOV(3, ioReg, scratchReg))
}

func (cg *CodeGenerator) generateUnary(expr *UnaryExpr, destReg uint8) error {
	if err := cg.generateExpr(expr.Operand, destReg); err != nil {
		return err
	}
	switch expr.Op {
	case TOKEN_MINUS:
		// Negate: 0 - value. Scratch must not be R1: a binary op parks its
		// left operand there while this unary (the right operand) runs.
		cg.emitUnaryNeg(destReg, 0)
	case TOKEN_NOT:
		scratch := unaryScratch(destReg)
		cg.hMovImm(scratch, 0)
		cg.builder.AddInstruction(rom.EncodeCMP(0, destReg, scratch))
		cg.hMovImm(destReg, 0)
		cg.builder.AddInstruction(rom.EncodeBNE())
		skipPos := cg.builder.GetCodeLength()
		cg.builder.AddImmediate(0)
		cg.hMovImm(destReg, 1)
		cg.patchLabel(skipPos)
		return nil
	case TOKEN_TILDE:
		cg.emitUnaryNeg(destReg, 0xFFFF)
		return nil
	default:
		return fmt.Errorf("unary operator not yet implemented: %v", expr.Op)
	}
	return nil
}

func (cg *CodeGenerator) allocateStack(bytes uint16, what string) (uint16, error) {
	if bytes == 0 {
		return cg.stackOffset, nil
	}
	if uint32(cg.stackOffset) < uint32(stackMinAddr)+uint32(bytes) {
		return 0, fmt.Errorf(
			"stack exhausted while allocating %s (%d bytes, SP=0x%04X, floor=0x%04X)",
			what, bytes, cg.stackOffset, stackMinAddr,
		)
	}
	cg.stackOffset -= bytes
	return cg.stackOffset, nil
}

func (cg *CodeGenerator) here() codePos {
	return codePos{bank: cg.currentBank, word: cg.builder.GetCodeLength()}
}

func (cg *CodeGenerator) openCount(bank uint8) int {
	n := 0
	for _, site := range cg.openBranches {
		if site.bank == bank {
			n++
		}
	}
	return n
}

func (cg *CodeGenerator) clearOpen(site codePos) {
	kept := cg.openBranches[:0]
	for _, existing := range cg.openBranches {
		if existing != site {
			kept = append(kept, existing)
		}
	}
	cg.openBranches = kept
}

func (cg *CodeGenerator) captureOpenSite() codePos {
	pos := codePos{bank: cg.currentBank, word: cg.builder.GetCodeLength()}
	cg.builder.AddImmediate(0)
	cg.openBranches = append(cg.openBranches, pos)
	return pos
}

func (cg *CodeGenerator) emitOpenJump() codePos {
	cg.builder.AddInstruction(rom.EncodeJMP())
	return cg.captureOpenSite()
}

func (cg *CodeGenerator) emitOpenCond(inst uint16) codePos {
	cg.builder.AddInstruction(inst)
	return cg.captureOpenSite()
}

// maybeSpill continues the function under emission into the next reserved
// ROM bank when the current bank cannot hold another statement plus the
// trampolines already-open branches may need. Passes 1 and 2, and any
// function that fits in its assigned bank, do not spill.
func (cg *CodeGenerator) maybeSpill() error {
	if cg.bankedBuilder == nil || cg.fnBankEnd <= cg.currentBank {
		return nil
	}
	openWords := farJumpWords * cg.openCount(cg.currentBank)
	limit := rom.ROMBankSizeWords - openWords
	if limit < farJumpWords {
		return fmt.Errorf("function %s: too many open branches to continue past ROM bank %d", cg.currentFn, cg.currentBank)
	}
	if cg.builder.GetCodeLength()+farJumpWords+bankSpillSlackWords <= limit {
		return nil
	}
	next := cg.currentBank + 1
	if next > cg.fnBankEnd || next > rom.ROMMaxProgramBank {
		return fmt.Errorf("function %s is larger than the ROM banks reserved for it (through bank %d)", cg.currentFn, cg.fnBankEnd)
	}
	if cg.builder.GetCodeLength()+farJumpWords > limit {
		return fmt.Errorf("function %s: no room in ROM bank %d to continue into bank %d", cg.currentFn, cg.currentBank, next)
	}
	destWord := cg.bankedBuilder.GetCodeLength(next)
	cg.emitFarJump(next, uint16(rom.ROMBankOffsetBase+destWord*2))
	cg.currentBank = next
	return nil
}

func (cg *CodeGenerator) emitFarJump(bank uint8, pc uint16) {
	cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
	cg.builder.AddImmediate(uint16(bank))
	cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
	cg.builder.AddImmediate(pc)
	cg.builder.AddInstruction(rom.EncodeJMPFar(6, 7))
}

// patchPos backpatches site so it transfers to target. Same-bank sites stay
// a relative branch. A target in another bank gets a far-jump trampoline
// appended to the site's own bank, and the original branch lands on that.
func (cg *CodeGenerator) patchPos(site, target codePos) error {
	cg.clearOpen(site)
	if cg.bankedBuilder == nil || site.bank == target.bank {
		currentPC := uint16(site.word * 2)
		targetPC := uint16(target.word * 2)
		offset := cg.calcBranchOffset(currentPC, targetPC)
		cg.setPatchImmediate(site.bank, site.word, uint16(offset))
		return nil
	}
	return cg.emitTrampoline(site, target)
}

func (cg *CodeGenerator) emitTrampoline(site, target codePos) error {
	start := cg.bankedBuilder.GetCodeLength(site.bank)
	if start+farJumpWords > rom.ROMBankSizeWords {
		return fmt.Errorf("function %s: ROM bank %d has no room for a cross-bank branch", cg.currentFn, site.bank)
	}
	pc := uint16(rom.ROMBankOffsetBase + target.word*2)
	cg.bankedBuilder.AddInstruction(site.bank, rom.EncodeMOV(1, 6, 0))
	cg.bankedBuilder.AddImmediate(site.bank, uint16(target.bank))
	cg.bankedBuilder.AddInstruction(site.bank, rom.EncodeMOV(1, 7, 0))
	cg.bankedBuilder.AddImmediate(site.bank, pc)
	cg.bankedBuilder.AddInstruction(site.bank, rom.EncodeJMPFar(6, 7))

	currentPC := uint16(site.word * 2)
	targetPC := uint16(start * 2)
	offset := rom.CalculateBranchOffset(currentPC, targetPC)
	cg.setPatchImmediate(site.bank, site.word, uint16(offset))
	return nil
}

func (cg *CodeGenerator) patchLabel(offsetPos int) {
	cg.patchJumpTo(offsetPos, cg.builder.GetCodeLength())
}

// patchJumpTo backpatches a jump/branch immediate at offsetPos (word index of
// the placeholder) to branch to targetPos (word index of the destination).
// Unlike patchLabel, the target need not be the current end of code — used
// for `continue`, whose target (a for loop's increment step) is already
// behind the jump site by the time it's patched.
func (cg *CodeGenerator) patchJumpTo(offsetPos, targetPos int) {
	currentPC := uint16(offsetPos * 2)
	targetPC := uint16(targetPos * 2)
	offset := cg.calcBranchOffset(currentPC, targetPC)
	cg.builder.SetImmediateAt(offsetPos, uint16(offset))
}

// calcBranchOffset wraps rom.CalculateBranchOffset, tolerating an out-of-
// range offset in passes 1/2 (cg.bankedBuilder == nil -- both use a flat,
// unbounded-address ROMBuilder with no per-bank reset). An overflow there
// is expected, not a bug, once a program needs more than one bank: pass 1
// detects it via its own size checks (see generateBody) and abandons the
// attempt, falling through to multi-bank -- and pass 2 only needs word
// counts to build its bank schedule, never the branch's actual runtime
// correctness, so a dummy offset here is harmless as long as generation
// keeps running to completion (every function needs to be measured, not
// just the ones before the first overflow). In pass 3, a real
// BankedROMBuilder resets addressing per bank, so every same-function
// branch should always be comfortably in range; a panic there means a
// genuine bug (a direct branch crossed banks without patchPos) and is left
// to propagate rather than being silently masked into corrupted code.
func (cg *CodeGenerator) calcBranchOffset(currentPC, targetPC uint16) (offset int16) {
	if cg.bankedBuilder != nil {
		return rom.CalculateBranchOffset(currentPC, targetPC)
	}
	defer func() {
		if recover() != nil {
			offset = 0
		}
	}()
	return rom.CalculateBranchOffset(currentPC, targetPC)
}

// Scratch slots inside the reserved runtime block used by the software
// arithmetic helpers (the block at runtimeBlockBase is compiler-owned).
const (
	scratchIndexStore = runtimeBlockBase + 0x00 // array-store value stash
	exprSpillSlots    = 15
	inputCurrSlot     = runtimeBlockBase + 0x20 // input.poll current frame state
	inputPrevSlot     = runtimeBlockBase + 0x22 // input.poll previous frame state

	// Music player state (music.play / music.stop). The __musicadvance helper,
	// called once per wait_vblank when the program declares a music asset,
	// reads these to stream the current frame's YM2608 writes and advance.
	musicActiveSlot    = runtimeBlockBase + 0x40 // 3 = jingle, 2 = looping, 1 = one-shot, 0 = stopped
	musicFrameSlot     = runtimeBlockBase + 0x42 // current frame index
	musicCountSlot     = runtimeBlockBase + 0x44 // total frame count (one-shot end)
	musicCBankSlot     = runtimeBlockBase + 0x46 // counts-table bank
	musicCOffSlot      = runtimeBlockBase + 0x48 // counts-table offset
	musicPBankSlot     = runtimeBlockBase + 0x4A // pointer-table bank
	musicPOffSlot      = runtimeBlockBase + 0x4C // pointer-table offset
	musicLastFrameSlot = runtimeBlockBase + 0x4E // last frame_counter() value processed

	// music.fade_to state, advanced once per real frame alongside the song
	// player (independent of whether a song is actively playing).
	musicFadeActiveSlot     = runtimeBlockBase + 0x50 // 1 = fading, 0 = idle
	musicFadeTargetSlot     = runtimeBlockBase + 0x52 // target volume (0-255)
	musicFadeFramesLeftSlot = runtimeBlockBase + 0x54 // frames remaining

	// music.play_jingle save slots: the BGM's state (mode 0/1/2, never 3)
	// stashed while a jingle (mode 3) plays, restored when the jingle ends.
	// Mirror the musicActiveSlot..musicPOffSlot layout one-for-one.
	musicSavedActiveSlot = runtimeBlockBase + 0x56
	musicSavedFrameSlot  = runtimeBlockBase + 0x58
	musicSavedCountSlot  = runtimeBlockBase + 0x5A
	musicSavedCBankSlot  = runtimeBlockBase + 0x5C
	musicSavedCOffSlot   = runtimeBlockBase + 0x5E
	musicSavedPBankSlot  = runtimeBlockBase + 0x60
	musicSavedPOffSlot   = runtimeBlockBase + 0x62
)

// emitHelperCall emits a CALL to a named helper routine, patched after all
// code is generated (same mechanism as user function calls).
func (cg *CodeGenerator) emitHelperCall(name string) {
	cg.emitCallPatch(name)
}

// emitCallPatch emits a CALL to target (a user function or internal helper
// name) and records a callPatch, resolved once every function's final
// address is known (see Generate's patch-resolution loop). In compact mode
// this is the same 2-word relative CALL form used before multi-bank
// support existed. In wide mode (pass 2 measurement, pass 3 final
// multi-bank emission) it emits the 5-word far-call form instead: two
// placeholder MOV immediates loading the target's bank/offset into R6/R7
// (reserved scratch -- see the param cap in generateFunctionPrologue), then
// a far CALL[R6:R7]. Every call site (user calls, boot.show_default,
// internal helpers) goes through this one function so their emitted size
// stays identical between the pass that measures it and the pass that
// schedules banks against that measurement.
func (cg *CodeGenerator) emitCallPatch(target string) {
	if cg.wideCallMode {
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
		bankPos := cg.builder.GetCodeLength()
		cg.builder.AddImmediate(0) // placeholder bank
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
		offsetPos := cg.builder.GetCodeLength()
		cg.builder.AddImmediate(0) // placeholder offset
		cg.builder.AddInstruction(rom.EncodeCALLFar(6, 7))
		cg.callPatches = append(cg.callPatches, callPatch{
			bankPos:   bankPos,
			offsetPos: offsetPos,
			bank:      cg.currentBank,
			target:    target,
			wide:      true,
		})
		return
	}
	cg.builder.AddInstruction(rom.EncodeCALL())
	offsetPos := cg.builder.GetCodeLength()
	cg.builder.AddImmediate(0) // placeholder
	cg.callPatches = append(cg.callPatches, callPatch{
		offsetPos: offsetPos,
		bank:      cg.currentBank,
		target:    target,
	})
}

// helper emit primitives
func (cg *CodeGenerator) hMovImm(reg uint8, v uint16) {
	cg.builder.AddInstruction(rom.EncodeMOV(1, reg, 0))
	cg.builder.AddImmediate(v)
}

// exprNeedsSpill reports whether evaluating e will reuse R1 (the register a
// binary operator parks its left operand in) or the argument registers of
// a call already in progress. Leaves — literals, names, unary operators on
// leaves, and indexes of leaves — only write their destination and R6/R7.
func exprNeedsSpill(e Expr) bool {
	switch x := e.(type) {
	case *NumberExpr, *BoolExpr, *IdentExpr:
		return false
	case *UnaryExpr:
		return exprNeedsSpill(x.Operand)
	case *IndexExpr:
		return exprNeedsSpill(x.Index)
	default:
		return true
	}
}

func (cg *CodeGenerator) spillPush(reg uint8) error {
	if cg.exprSpill >= exprSpillSlots {
		return fmt.Errorf("expression too complex")
	}
	addr := cg.spillBase + uint16(cg.exprSpill)*2
	cg.exprSpill++
	cg.hStore16(addr, reg)
	return nil
}

func (cg *CodeGenerator) spillPop(reg uint8) {
	cg.exprSpill--
	addr := cg.spillBase + uint16(cg.exprSpill)*2
	cg.hLoad16(reg, addr)
}

func (cg *CodeGenerator) hStore16(addr uint16, src uint8) {
	cg.hMovImm(7, addr)
	cg.builder.AddInstruction(rom.EncodeMOV(3, 7, src))
}
func (cg *CodeGenerator) hLoad16(dst uint8, addr uint16) {
	cg.hMovImm(7, addr)
	cg.builder.AddInstruction(rom.EncodeMOV(2, dst, 7))
}

// copyWRAMSlot copies a 16-bit WRAM value from src to dst. Clobbers R6-R7.
func (cg *CodeGenerator) copyWRAMSlot(dst, src uint16) {
	cg.hLoad16(6, src)
	cg.hStore16(dst, 6)
}
func (cg *CodeGenerator) hAndImm(reg uint8, v uint16) {
	cg.builder.AddInstruction(rom.EncodeAND(1, reg, 0))
	cg.builder.AddImmediate(v)
}
func (cg *CodeGenerator) hShrImm(reg uint8, v uint16) {
	cg.builder.AddInstruction(rom.EncodeSHR(1, reg, 0))
	cg.builder.AddImmediate(v)
}
func (cg *CodeGenerator) hShlImm(reg uint8, v uint16) {
	cg.builder.AddInstruction(rom.EncodeSHL(1, reg, 0))
	cg.builder.AddImmediate(v)
}
func (cg *CodeGenerator) hCmpImm(reg uint8, v uint16) {
	cg.builder.AddInstruction(rom.EncodeCMP(7, reg, 0))
	cg.builder.AddImmediate(v)
}

// emitSizeCodeBits extracts a sprite's 3-bit size code from a ctrl-shaped
// value in srcReg and ORs it, shifted into OAM X-high-byte bit positions
// [3:1], into accumReg. Clobbers scratchReg; srcReg is read-only.
//
// The 3 bits come from: bit 1 = code bit 0 (the legacy SPR_SIZE_16
// position -- existing CoreLX source that only ever calls
// SPR_SIZE_8()/SPR_SIZE_16() keeps working unchanged, since that's the
// only bit it ever sets); bits 8/9 = code bits 1/2, new, set only by
// SPR_SIZE_32X16() and larger. See spriteSizeTable (internal/ppu/
// scanline.go) for what each of the resulting 8 codes means, and
// oam.write_sprite_data/sprite.set_size for the two call sites.
func (cg *CodeGenerator) emitSizeCodeBits(accumReg, srcReg, scratchReg uint8) {
	for _, bit := range []struct{ shr, shl uint16 }{{1, 1}, {8, 2}, {9, 3}} {
		cg.builder.AddInstruction(rom.EncodeMOV(0, scratchReg, srcReg))
		cg.hShrImm(scratchReg, bit.shr)
		cg.hAndImm(scratchReg, 1)
		cg.hShlImm(scratchReg, bit.shl)
		cg.builder.AddInstruction(rom.EncodeOR(0, accumReg, scratchReg))
	}
}

// hBranch emits a conditional branch with a placeholder offset and returns
// the immediate position for later patching.
func (cg *CodeGenerator) hBranch(inst uint16) int {
	cg.builder.AddInstruction(inst)
	pos := cg.builder.GetCodeLength()
	cg.builder.AddImmediate(0)
	return pos
}

func (cg *CodeGenerator) hPatchToHere(immPos int) {
	herePC := uint16(cg.builder.GetCodeLength() * 2)
	cg.builder.SetImmediateAt(immPos, uint16(cg.calcBranchOffset(uint16(immPos*2), herePC)))
}

func (cg *CodeGenerator) hJumpBack(toWordIdx int) {
	cg.builder.AddInstruction(rom.EncodeJMP())
	fromPC := uint16(cg.builder.GetCodeLength() * 2)
	cg.builder.AddImmediate(uint16(cg.calcBranchOffset(fromPC, uint16(toWordIdx*2))))
}

// irqVectorPatch records one placeholder pair (bank byte + offset-high
// byte) written into the entry function by emitIRQVectorFix, plus the bank
// that placeholder pair itself lives in (== the entry function's bank,
// always 1 by the hardware boot vector requirement, but recorded rather
// than assumed so patchIRQVector never needs its own bank constant).
type irqVectorPatch struct {
	bank      uint8
	bankPos   int
	offsetPos int
}

// emitIRQVectorFix writes bank0:0xFFE0-0xFFE3 (the IRQ and NMI vectors) so
// they point at a trivial RET stub instead of the hardware default, which
// aliases the entry point. Must run before any other entry-function code.
// Both the bank byte and the offset-high byte are placeholders, patched by
// patchIRQVector once __irqstub's real (bank, address) is known -- its
// bank isn't fixed, since __irqstub is emitted wherever cg.currentBank
// happens to be once every user function/helper has been emitted. Clobbers
// R6-R7.
func (cg *CodeGenerator) emitIRQVectorFix() {
	patchVector := func(vectorAddr uint16) {
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
		bankPos := cg.builder.GetCodeLength()
		cg.builder.AddImmediate(0) // placeholder: patched to __irqstub's bank
		cg.storeIOByte(vectorAddr, 6)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))
		offsetPos := cg.builder.GetCodeLength()
		cg.builder.AddImmediate(0) // placeholder: patched to __irqstub's page
		cg.storeIOByte(vectorAddr+1, 6)
		cg.irqVectorPatchPositions = append(cg.irqVectorPatchPositions, irqVectorPatch{
			bank:      cg.currentBank,
			bankPos:   bankPos,
			offsetPos: offsetPos,
		})
	}
	patchVector(0xFFE0) // IRQ vector
	patchVector(0xFFE2) // NMI vector
}

// patchIRQVector emits __irqstub (a lone RET) padded to a 0x100-byte page
// boundary -- the vector format only stores a page number (offset low byte
// is always 0x00) -- then backfills the placeholders emitIRQVectorFix left
// in the entry function's prologue with __irqstub's actual bank and that
// page's high byte.
func (cg *CodeGenerator) patchIRQVector() {
	if len(cg.irqVectorPatchPositions) == 0 {
		return
	}
	for (cg.builder.GetCodeLength()*2)%256 != 0 {
		cg.builder.AddInstruction(rom.EncodeNOP())
	}
	stubAbsAddr := 0x8000 + uint16(cg.builder.GetCodeLength()*2)
	stubBank := cg.currentBank
	cg.builder.AddInstruction(rom.EncodeRET())
	for _, p := range cg.irqVectorPatchPositions {
		cg.setPatchImmediate(p.bank, p.bankPos, uint16(stubBank))
		cg.setPatchImmediate(p.bank, p.offsetPos, uint16(stubAbsAddr>>8))
	}
}

// emitFixmulHelper emits __fixmul: signed 8.8 fixed multiply with
// 32-bit-correct partial products on the hardware 16x16->low16 MUL.
// In: R0 = a, R1 = b. Out: R0 = (a*b) >> 8 (8.8). Clobbers R1-R3, R6, R7.
//
//	result = (ah*bh)<<8 + ah*bl + al*bh + (al*bl)>>8   (on |a|,|b|)
func (cg *CodeGenerator) emitFixmulHelper() {
	cg.recordFuncAddr("__fixmul")

	// sign = (a ^ b) >> 15  -> R3 (kept live; no calls below)
	cg.builder.AddInstruction(rom.EncodeMOV(0, 3, 0))
	cg.builder.AddInstruction(rom.EncodeXOR(0, 3, 1))
	cg.hShrImm(3, 15)

	// a = abs(a)
	cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 0))
	cg.hShrImm(6, 15)
	cg.hCmpImm(6, 0)
	aPos := cg.hBranch(rom.EncodeBEQ())
	cg.hMovImm(6, 0)
	cg.builder.AddInstruction(rom.EncodeSUB(0, 6, 0))
	cg.builder.AddInstruction(rom.EncodeMOV(0, 0, 6))
	cg.hPatchToHere(aPos)

	// b = abs(b)
	cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 1))
	cg.hShrImm(6, 15)
	cg.hCmpImm(6, 0)
	bPos := cg.hBranch(rom.EncodeBEQ())
	cg.hMovImm(6, 0)
	cg.builder.AddInstruction(rom.EncodeSUB(0, 6, 1))
	cg.builder.AddInstruction(rom.EncodeMOV(0, 1, 6))
	cg.hPatchToHere(bPos)

	// R2 = (al*bl) >> 8
	cg.builder.AddInstruction(rom.EncodeMOV(0, 2, 0))
	cg.hAndImm(2, 0x00FF)
	cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 1))
	cg.hAndImm(6, 0x00FF)
	cg.builder.AddInstruction(rom.EncodeMUL(0, 2, 6))
	cg.hShrImm(2, 8)

	// R2 += al*bh
	cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 0))
	cg.hAndImm(6, 0x00FF)
	cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 1))
	cg.hShrImm(7, 8)
	cg.builder.AddInstruction(rom.EncodeMUL(0, 6, 7))
	cg.builder.AddInstruction(rom.EncodeADD(0, 2, 6))

	// R2 += ah*bl
	cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 0))
	cg.hShrImm(6, 8)
	cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 1))
	cg.hAndImm(7, 0x00FF)
	cg.builder.AddInstruction(rom.EncodeMUL(0, 6, 7))
	cg.builder.AddInstruction(rom.EncodeADD(0, 2, 6))

	// R2 += (ah*bh) << 8
	cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 0))
	cg.hShrImm(6, 8)
	cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 1))
	cg.hShrImm(7, 8)
	cg.builder.AddInstruction(rom.EncodeMUL(0, 6, 7))
	cg.hShlImm(6, 8)
	cg.builder.AddInstruction(rom.EncodeADD(0, 2, 6))

	// apply sign
	cg.hCmpImm(3, 0)
	pos := cg.hBranch(rom.EncodeBEQ())
	cg.hMovImm(6, 0)
	cg.builder.AddInstruction(rom.EncodeSUB(0, 6, 2))
	cg.builder.AddInstruction(rom.EncodeMOV(0, 0, 6))
	cg.builder.AddInstruction(rom.EncodeRET())
	cg.hPatchToHere(pos)
	cg.builder.AddInstruction(rom.EncodeMOV(0, 0, 2))
	cg.builder.AddInstruction(rom.EncodeRET())
}

// unaryScratch is a register unary operators may clobber. It is never R1,
// which holds a binary operator's left operand while the right is evaluated,
// and it is never destReg, which holds the operand.
func unaryScratch(destReg uint8) uint8 {
	if destReg == 6 {
		return 7
	}
	return 6
}

func (cg *CodeGenerator) emitUnaryNeg(destReg uint8, minuend uint16) {
	scratch := unaryScratch(destReg)
	cg.hMovImm(scratch, minuend)
	cg.builder.AddInstruction(rom.EncodeSUB(0, scratch, destReg))
	cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, scratch))
}

// constPow2Shift reports the shift count when expr is a positive power-of-two
// constant in 1..32768. Dividing a non-negative int by that constant is a
// logical right shift.
func constPow2Shift(expr Expr) (uint16, bool) {
	num, ok := expr.(*NumberExpr)
	if !ok || num.Value <= 0 || num.Value > 32768 {
		return 0, false
	}
	u := uint16(num.Value)
	if u == 0 || u&(u-1) != 0 {
		return 0, false
	}
	shift := uint16(0)
	for u > 1 {
		u >>= 1
		shift++
	}
	return shift, true
}

// emitNonNegDivFast divides R0 by the divisor. shiftOK means the divisor is
// 2^shift and a non-negative dividend is a logical right shift; otherwise
// the divisor is already in R1 and a non-negative pair uses hardware DIV.
// A negative operand calls __divs, which overwrites R0 with the signed
// quotient. Clobbers R6.
func (cg *CodeGenerator) emitNonNegDivFast(shiftOK bool, shift uint16) {
	// Non-negative operands stay inline (a shift, or one DIV). A negative
	// operand calls __divs. The test is what keeps a positive quotient off
	// the far-call path.
	cg.needDivs = true
	cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 0))
	if !shiftOK {
		cg.builder.AddInstruction(rom.EncodeOR(0, 6, 1))
	}
	cg.hShrImm(6, 15)
	cg.hCmpImm(6, 0)
	neg := cg.hBranch(rom.EncodeBNE())
	if shiftOK {
		if shift > 0 {
			cg.hShrImm(0, shift)
		}
	} else {
		cg.builder.AddInstruction(rom.EncodeDIV(0, 0, 1))
	}
	done := cg.hBranch(rom.EncodeJMP())
	cg.hPatchToHere(neg)
	if shiftOK {
		cg.builder.AddInstruction(rom.EncodeMOV(0, 1, 2))
	}
	cg.emitHelperCall("__divs")
	cg.hPatchToHere(done)
}

// emitAbsInPlace replaces reg with its 16-bit absolute value. -32768 stays
// 0x8000, which is the unsigned magnitude the following DIV expects.
// Clobbers R6.
func (cg *CodeGenerator) emitAbsInPlace(reg uint8) {
	cg.builder.AddInstruction(rom.EncodeMOV(0, 6, reg))
	cg.hShrImm(6, 15)
	cg.hCmpImm(6, 0)
	nonNeg := cg.hBranch(rom.EncodeBEQ())
	cg.hMovImm(6, 0)
	cg.builder.AddInstruction(rom.EncodeSUB(0, 6, reg))
	cg.builder.AddInstruction(rom.EncodeMOV(0, reg, 6))
	cg.hPatchToHere(nonNeg)
}

// emitResign negates R0 when R3 is nonzero. R3 is the saved sign flag
// (1 = result negative). Clobbers R6.
func (cg *CodeGenerator) emitResign() {
	cg.hCmpImm(3, 0)
	pos := cg.hBranch(rom.EncodeBEQ())
	cg.hMovImm(6, 0)
	cg.builder.AddInstruction(rom.EncodeSUB(0, 6, 0))
	cg.builder.AddInstruction(rom.EncodeMOV(0, 0, 6))
	cg.builder.AddInstruction(rom.EncodeRET())
	cg.hPatchToHere(pos)
	cg.builder.AddInstruction(rom.EncodeRET())
}

// emitDivsHelper emits __divs: signed 16-bit division, toward zero.
// In: R0 = dividend, R1 = divisor. Out: R0 = quotient.
// Divisor 0 yields 0xFFFF, matching hardware DIV. Clobbers R1, R3, R6.
func (cg *CodeGenerator) emitDivsHelper() {
	cg.recordFuncAddr("__divs")
	cg.builder.AddInstruction(rom.EncodeMOV(0, 3, 0))
	cg.builder.AddInstruction(rom.EncodeXOR(0, 3, 1))
	cg.hShrImm(3, 15)
	cg.emitAbsInPlace(0)
	cg.emitAbsInPlace(1)
	cg.hCmpImm(1, 0)
	ok := cg.hBranch(rom.EncodeBNE())
	cg.hMovImm(0, 0xFFFF)
	cg.builder.AddInstruction(rom.EncodeRET())
	cg.hPatchToHere(ok)
	cg.builder.AddInstruction(rom.EncodeDIV(0, 0, 1))
	cg.emitResign()
}

// emitFixdivHelper emits __fixdiv: signed 8.8 division, toward zero.
// The magnitude is ((|a|/|b|)<<8) + ((|a|%|b|)<<8)/|b|, wrapped to 16 bits.
// The fractional term is an 8-step restoring division so a remainder of
// 256 or more does not lose bits in a 16-bit shift. In: R0 = a, R1 = b.
// Out: R0 = quotient. Divisor 0 yields 0xFFFF. Clobbers R1-R7. R3 holds
// the sign and is left alone by the fractional steps.
func (cg *CodeGenerator) emitFixdivHelper() {
	cg.recordFuncAddr("__fixdiv")
	cg.builder.AddInstruction(rom.EncodeMOV(0, 3, 0))
	cg.builder.AddInstruction(rom.EncodeXOR(0, 3, 1))
	cg.hShrImm(3, 15)
	cg.emitAbsInPlace(0)
	cg.emitAbsInPlace(1)
	cg.hCmpImm(1, 0)
	ok := cg.hBranch(rom.EncodeBNE())
	cg.hMovImm(0, 0xFFFF)
	cg.builder.AddInstruction(rom.EncodeRET())
	cg.hPatchToHere(ok)
	cg.builder.AddInstruction(rom.EncodeMOV(0, 4, 0)) // |a|
	cg.builder.AddInstruction(rom.EncodeMOV(0, 5, 1)) // |b|
	cg.builder.AddInstruction(rom.EncodeDIV(0, 0, 5)) // qHi
	cg.builder.AddInstruction(rom.EncodeMOV(0, 1, 0)) // qHi lives in R1
	cg.builder.AddInstruction(rom.EncodeMOV(0, 2, 0))
	cg.builder.AddInstruction(rom.EncodeMUL(0, 2, 5)) // qHi*|b|
	cg.builder.AddInstruction(rom.EncodeMOV(0, 0, 4))
	cg.builder.AddInstruction(rom.EncodeSUB(0, 0, 2)) // rem
	cg.builder.AddInstruction(rom.EncodeMOV(0, 4, 0))
	cg.hMovImm(2, 0) // qLo
	for i := 0; i < 8; i++ {
		cg.emitFixdivFracStep()
	}
	cg.hShlImm(1, 8)
	cg.builder.AddInstruction(rom.EncodeADD(0, 1, 2))
	cg.builder.AddInstruction(rom.EncodeMOV(0, 0, 1))
	cg.emitResign()
}

// emitFixdivFracStep is one restoring-division step of (rem<<8)/divisor.
// R4 is the remainder (less than the divisor on entry), R5 the divisor,
// R2 the quotient bits gathered so far. The bit shifted out of R4 is a
// 17th remainder bit: if it is set, the true remainder is past any 16-bit
// divisor and one wrapping subtract brings it back under R5.
func (cg *CodeGenerator) emitFixdivFracStep() {
	cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 4))
	cg.hShrImm(6, 15)
	cg.hShlImm(4, 1)
	cg.hCmpImm(6, 0)
	msbSet := cg.hBranch(rom.EncodeBNE())
	// Unsigned R4 >= R5, via a sign-bit flip so BLT's signed compare applies.
	cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 4))
	cg.builder.AddInstruction(rom.EncodeXOR(1, 6, 0))
	cg.builder.AddImmediate(0x8000)
	cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 5))
	cg.builder.AddInstruction(rom.EncodeXOR(1, 7, 0))
	cg.builder.AddImmediate(0x8000)
	cg.builder.AddInstruction(rom.EncodeCMP(0, 6, 7))
	tooSmall := cg.hBranch(rom.EncodeBLT())
	cg.hPatchToHere(msbSet)
	cg.builder.AddInstruction(rom.EncodeSUB(0, 4, 5))
	cg.hShlImm(2, 1)
	cg.hMovImm(6, 1)
	cg.builder.AddInstruction(rom.EncodeADD(0, 2, 6))
	skip := cg.hBranch(rom.EncodeJMP())
	cg.hPatchToHere(tooSmall)
	cg.hShlImm(2, 1)
	cg.hPatchToHere(skip)
}

// emitYM2608Silence emits the YM2608 silence sequence: FM key-off on all six
// melodic channels, mute the SSG mixer/channels, and disable rhythm/ADPCM
// playback. Mirrors the hand-verified sequence in
// test/roms/build_pong_ym2608.go's emitSilenceYM2608. Clobbers R6.
func (cg *CodeGenerator) emitYM2608Silence() {
	writeYM := func(port0Addr, port0Data uint16) {
		cg.hMovImm(6, port0Addr)
		cg.storeIOByte(0x9100, 6)
		cg.hMovImm(6, port0Data)
		cg.storeIOByte(0x9101, 6)
	}
	for _, ch := range []uint16{0x00, 0x01, 0x02, 0x04, 0x05, 0x06} {
		writeYM(0x28, ch) // key-off (no operator bits set)
	}
	writeYM(0x07, 0x3F) // SSG mixer: all channels off
	writeYM(0x08, 0x00) // SSG channel A level
	writeYM(0x09, 0x00) // SSG channel B level
	writeYM(0x0A, 0x00) // SSG channel C level
	writeYM(0x10, 0x00) // rhythm: all off

	// ADPCM-B reset/stop, via port 1.
	cg.hMovImm(6, 0x00)
	cg.storeIOByte(0x9104, 6)
	cg.hMovImm(6, 0x01)
	cg.storeIOByte(0x9105, 6)
}

// emitMusicFadeStep emits the music.fade_to per-frame interpolation step,
// inlined into __musicadvance so it runs once per real frame regardless of
// whether a song is actively playing via music.play/play_loop. Recomputes
// the step fresh each frame (current +/- abs(target-current)/framesLeft)
// rather than storing a fixed per-frame delta, so it lands exactly on the
// target volume when framesLeft reaches 1 regardless of rounding along the
// way. Clobbers R3-R7; assumes DBR is already 0 (true at every call site —
// this only ever runs before __musicadvance's own DBR-touching sections).
func (cg *CodeGenerator) emitMusicFadeStep() {
	cg.hLoad16(6, musicFadeActiveSlot)
	cg.hCmpImm(6, 0)
	noFadePos := cg.hBranch(rom.EncodeBEQ())

	cg.hLoad16(3, musicFadeFramesLeftSlot)
	cg.hCmpImm(3, 1)
	notLastStepPos := cg.hBranch(rom.EncodeBGT())

	// Last step: snap exactly to the target and stop fading (avoids drifting
	// off-target from repeated integer-division rounding).
	cg.hLoad16(6, musicFadeTargetSlot)
	cg.storeIOByte(0x9106, 6)
	cg.hMovImm(6, 0)
	cg.hStore16(musicFadeActiveSlot, 6)
	fadeDonePos := cg.hBranch(rom.EncodeJMP())

	cg.hPatchToHere(notLastStepPos)
	// Partial step: current +/- abs(target-current)/framesLeft (framesLeft > 1).
	cg.hLoad16(4, 0x9106) // R4 = current volume (I/O-aware 8-bit load)
	cg.hLoad16(5, musicFadeTargetSlot)
	cg.builder.AddInstruction(rom.EncodeSUB(0, 5, 4)) // R5 = target - current (signed delta)

	// Extract the sign bit (same idiom as emitFixmulHelper's sign handling):
	// a logical shift right by 15 isolates bit 15, giving 0 (delta >= 0) or
	// 1 (delta < 0) regardless of shift signedness, since only that one bit
	// survives.
	cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 5))
	cg.hShrImm(6, 15)
	cg.hCmpImm(6, 0)
	deltaPositivePos := cg.hBranch(rom.EncodeBEQ())
	cg.hMovImm(7, 0)
	cg.builder.AddInstruction(rom.EncodeSUB(0, 7, 5)) // R7 = 0 - delta
	cg.builder.AddInstruction(rom.EncodeMOV(0, 5, 7)) // R5 = abs(delta)
	cg.hPatchToHere(deltaPositivePos)

	cg.builder.AddInstruction(rom.EncodeDIV(0, 5, 3)) // R5 = abs(delta) / framesLeft (hardware DIV is unsigned)

	// Re-apply the sign and add to the current volume.
	cg.hCmpImm(6, 0)
	stepPositivePos := cg.hBranch(rom.EncodeBEQ())
	cg.builder.AddInstruction(rom.EncodeSUB(0, 4, 5)) // fading down: current -= step
	appliedPos := cg.hBranch(rom.EncodeJMP())
	cg.hPatchToHere(stepPositivePos)
	cg.builder.AddInstruction(rom.EncodeADD(0, 4, 5)) // fading up: current += step
	cg.hPatchToHere(appliedPos)

	cg.storeIOByte(0x9106, 4)

	cg.hLoad16(3, musicFadeFramesLeftSlot)
	cg.builder.AddInstruction(rom.EncodeSUB(1, 3, 0))
	cg.builder.AddImmediate(1)
	cg.hStore16(musicFadeFramesLeftSlot, 3)

	cg.hPatchToHere(fadeDonePos)
	cg.hPatchToHere(noFadePos)
}

// emitMusicAdvanceHelper emits __musicadvance: called once per wait_vblank()
// whenever the program declares any music asset. If a song is playing
// (musicActiveSlot != 0), streams the current frame's YM2608 writes through
// the bus-side burst streamer (0x9110-0x9115, see internal/memory/bus.go's
// executeYMBurst) and advances the frame index. At the end of the song, a
// one-shot (musicActiveSlot == 1) silences the chip and stops; a loop
// (musicActiveSlot == 2) wraps back to frame 0. If nothing is playing,
// returns immediately.
//
// Mirrors the hand-verified sequence in
// test/roms/build_pong_ym2608.go's emitSongPlayerFrame, with one correctness
// fix: that reference writes a stale register (always 0, left over from an
// unrelated earlier assignment) as the burst count's high byte instead of
// deriving the real value, silently truncating any per-frame write count
// >= 256. This version computes it fresh from the actual combined count.
//
// Reloads the frame index from WRAM at each use rather than keeping it live
// in a register across calls to the hLoad16/hStore16/storeIOByte helpers —
// they all use R7 as internal scratch, so nothing survives a call to them
// except registers those helpers don't touch.
//
// wait_vblank()'s VBlank flag persists for the whole VBlank window by design
// (internal/ppu/ppu.go: "allows ROM to read the flag multiple times during
// VBlank"), so a `while true: wait_vblank()` loop can complete several times
// within a single real video frame if the loop body is short. Without a
// guard, that would advance the song several steps per real frame instead of
// one. musicLastFrameSlot tracks the last frame_counter() value actually
// processed, so repeat calls within the same real frame are no-ops.
func (cg *CodeGenerator) emitMusicAdvanceHelper() {
	cg.recordFuncAddr("__musicadvance")

	// Read frame_counter() (same sequence as the frame_counter() builtin:
	// FRAME_COUNTER_LOW 0x803F, FRAME_COUNTER_HIGH 0x8040) and skip this call
	// if it's already been processed for the current real frame.
	cg.hMovImm(4, 0x803F)
	cg.builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // R5 = low byte
	cg.hMovImm(4, 0x8040)
	cg.builder.AddInstruction(rom.EncodeMOV(2, 6, 4)) // R6 = high byte
	cg.hShlImm(6, 8)
	cg.builder.AddInstruction(rom.EncodeOR(0, 5, 6)) // R5 = current frame_counter()
	cg.hLoad16(6, musicLastFrameSlot)
	cg.builder.AddInstruction(rom.EncodeCMP(0, 5, 6))
	alreadyProcessedPos := cg.hBranch(rom.EncodeBEQ())
	cg.hStore16(musicLastFrameSlot, 5)

	// Advance any in-progress music.fade_to, independent of whether a song
	// is actively playing via music.play/play_loop.
	cg.emitMusicFadeStep()

	cg.hLoad16(7, musicActiveSlot)
	cg.hCmpImm(7, 0)
	notPlayingPos := cg.hBranch(rom.EncodeBEQ())

	// Counts-table lookup: address = countsOff + frameIndex*2, DBR = countsBank.
	cg.hLoad16(3, musicFrameSlot)
	cg.hShlImm(3, 1)
	cg.hLoad16(6, musicCOffSlot)
	cg.builder.AddInstruction(rom.EncodeADD(0, 3, 6))
	cg.hLoad16(6, musicCBankSlot)
	cg.builder.AddInstruction(rom.EncodeMOV(8, 6, 0)) // DBR = countsBank
	cg.builder.AddInstruction(rom.EncodeMOV(6, 4, 3)) // R4 = count lo, [R3]
	cg.builder.AddInstruction(rom.EncodeADD(1, 3, 0))
	cg.builder.AddImmediate(1)
	cg.builder.AddInstruction(rom.EncodeMOV(6, 5, 3)) // R5 = count hi, [R3]
	cg.hShlImm(5, 8)
	cg.builder.AddInstruction(rom.EncodeADD(0, 5, 4)) // R5 = combined write count

	// Reset DBR to 0 before the pointer-table lookup's own WRAM reads below
	// (musicFrameSlot/musicPOffSlot/musicPBankSlot) — they're 16-bit loads,
	// which are DBR-relative for non-I/O addresses just like the 8-bit ROM
	// reads above, so leaving DBR at countsBank here would silently read
	// those WRAM slots from the wrong bank.
	cg.hMovImm(1, 0)
	cg.builder.AddInstruction(rom.EncodeMOV(8, 1, 0))

	// Pointer-table lookup: address = ptrOff + frameIndex*4, DBR = ptrBank.
	cg.hLoad16(3, musicFrameSlot)
	cg.hShlImm(3, 2)
	cg.hLoad16(6, musicPOffSlot)
	cg.builder.AddInstruction(rom.EncodeADD(0, 3, 6))
	cg.hLoad16(6, musicPBankSlot)
	cg.builder.AddInstruction(rom.EncodeMOV(8, 6, 0)) // DBR = ptrBank
	cg.builder.AddInstruction(rom.EncodeMOV(6, 6, 3)) // R6 = data bank, [R3]
	cg.builder.AddInstruction(rom.EncodeADD(1, 3, 0))
	cg.builder.AddImmediate(1)
	cg.builder.AddInstruction(rom.EncodeMOV(6, 0, 3)) // R0 = data offset lo, [R3]
	cg.builder.AddInstruction(rom.EncodeADD(1, 3, 0))
	cg.builder.AddImmediate(1)
	cg.builder.AddInstruction(rom.EncodeMOV(6, 1, 3)) // R1 = data offset hi, [R3]
	cg.hShlImm(1, 8)
	cg.builder.AddInstruction(rom.EncodeADD(0, 0, 1)) // R0 = combined data offset

	// Reset DBR to 0 before touching bank-0 I/O/WRAM again — every other
	// generated instruction (globals, stack locals, I/O) assumes DBR == 0.
	cg.hMovImm(1, 0)
	cg.builder.AddInstruction(rom.EncodeMOV(8, 1, 0))

	// If this frame has no writes, skip the burst.
	cg.hCmpImm(5, 0)
	writeDonePos := cg.hBranch(rom.EncodeBEQ())

	// Program the bus-side YM burst streamer:
	// 0x9110/11 count, 0x9112 bank, 0x9113/14 offset, 0x9115 trigger.
	cg.storeIOByte(0x9110, 4) // count lo
	cg.builder.AddInstruction(rom.EncodeMOV(0, 2, 5))
	cg.hShrImm(2, 8)
	cg.storeIOByte(0x9111, 2) // count hi
	cg.storeIOByte(0x9112, 6) // source bank
	cg.storeIOByte(0x9113, 0) // source offset lo (I/O store truncates to the low byte)
	cg.builder.AddInstruction(rom.EncodeMOV(0, 1, 0))
	cg.hShrImm(1, 8)
	cg.storeIOByte(0x9114, 1) // source offset hi
	cg.hMovImm(1, 1)
	cg.storeIOByte(0x9115, 1) // trigger burst

	cg.hPatchToHere(writeDonePos)

	// Advance the frame index.
	cg.hLoad16(3, musicFrameSlot)
	cg.builder.AddInstruction(rom.EncodeADD(1, 3, 0))
	cg.builder.AddImmediate(1)
	cg.hLoad16(6, musicCountSlot)
	cg.builder.AddInstruction(rom.EncodeCMP(0, 3, 6))
	notAtEndPos := cg.hBranch(rom.EncodeBLT()) // frame+1 < count: not at the end yet

	// Reached the end of the song.
	cg.hLoad16(6, musicActiveSlot)
	cg.hCmpImm(6, 2)
	loopPos := cg.hBranch(rom.EncodeBEQ()) // mode == 2 (loop): wrap to 0
	cg.hCmpImm(6, 3)
	jingleEndPos := cg.hBranch(rom.EncodeBEQ()) // mode == 3 (jingle): restore the saved BGM

	// mode == 1 (one-shot): silence the chip, mark stopped, done.
	cg.emitYM2608Silence()
	cg.hMovImm(6, 0)
	cg.hStore16(musicActiveSlot, 6)
	cg.builder.AddInstruction(rom.EncodeRET())

	cg.hPatchToHere(jingleEndPos)
	// Restore whatever was playing (or not) before the jingle started.
	cg.copyWRAMSlot(musicActiveSlot, musicSavedActiveSlot)
	cg.copyWRAMSlot(musicFrameSlot, musicSavedFrameSlot)
	cg.copyWRAMSlot(musicCountSlot, musicSavedCountSlot)
	cg.copyWRAMSlot(musicCBankSlot, musicSavedCBankSlot)
	cg.copyWRAMSlot(musicCOffSlot, musicSavedCOffSlot)
	cg.copyWRAMSlot(musicPBankSlot, musicSavedPBankSlot)
	cg.copyWRAMSlot(musicPOffSlot, musicSavedPOffSlot)
	// If nothing was playing before the jingle, silence the chip too —
	// otherwise the restored song's own next-frame writes take over cleanly.
	cg.hLoad16(6, musicActiveSlot)
	cg.hCmpImm(6, 0)
	restoredSomethingPos := cg.hBranch(rom.EncodeBNE())
	cg.emitYM2608Silence()
	cg.hPatchToHere(restoredSomethingPos)
	cg.builder.AddInstruction(rom.EncodeRET())

	cg.hPatchToHere(loopPos)
	cg.hMovImm(3, 0) // loop: wrap frame index to 0

	cg.hPatchToHere(notAtEndPos)
	// R3 holds the new frame index (frame+1, or 0 if the loop path just ran).
	cg.hStore16(musicFrameSlot, 3)
	cg.builder.AddInstruction(rom.EncodeRET())

	cg.hPatchToHere(notPlayingPos)
	cg.builder.AddInstruction(rom.EncodeRET())

	cg.hPatchToHere(alreadyProcessedPos)
	cg.builder.AddInstruction(rom.EncodeRET())
}

// storeIOByte writes the low byte of valueReg to an I/O address (8-bit MMIO).
func (cg *CodeGenerator) storeIOByte(addr uint16, valueReg uint8) {
	cg.hMovImm(7, addr)
	cg.builder.AddInstruction(rom.EncodeMOV(7, 7, valueReg)) // MOV [R7], Rvalue (8-bit store)
}

// buttonMasks maps CoreLX button names to their controller bitmasks
// (bit position per internal/input: UP=0..Z=11).
var buttonMasks = map[string]int64{
	"UP": 0x0001, "DOWN": 0x0002, "LEFT": 0x0004, "RIGHT": 0x0008,
	"A": 0x0010, "B": 0x0020, "X": 0x0040, "Y": 0x0080,
	"L": 0x0100, "R": 0x0200, "START": 0x0400, "Z": 0x0800,
}

// selectMatrixPlane writes the channel to the plane-select register (0x8080)
// so subsequent matrix-plane register writes target that plane.
func (cg *CodeGenerator) selectMatrixPlane(channel Expr) error {
	if err := cg.generateExpr(channel, 0); err != nil {
		return err
	}
	cg.storeIOByte(0x8080, 0)
	return nil
}

// writePlaneByte evaluates expr and writes its low byte to an 8-bit register.
func (cg *CodeGenerator) writePlaneByte(expr Expr, addr uint16) error {
	if err := cg.generateExpr(expr, 0); err != nil {
		return err
	}
	cg.storeIOByte(addr, 0)
	return nil
}

// writePlaneReg16 evaluates expr and writes it as a low/high byte pair to two
// consecutive 8-bit registers (addrLow, addrLow+1). Matrix-plane 16-bit
// registers are exposed as adjacent 8-bit I/O ports, so a single 16-bit store
// would only land the low byte; this writes both halves explicitly.
func (cg *CodeGenerator) writePlaneReg16(expr Expr, addrLow uint16) error {
	if err := cg.generateExpr(expr, 0); err != nil {
		return err
	}
	cg.storeIOByte(addrLow, 0)                        // low byte
	cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 0)) // R6 = value
	cg.hShrImm(6, 8)                                  // R6 = value >> 8
	cg.storeIOByte(addrLow+1, 6)                      // high byte
	return nil
}

// emitDrawIntHelper emits __drawint: render the signed 16-bit value in R0 as
// decimal digits to the text port (0x8076), leading zeros suppressed, with a
// leading '-' for negatives. The port's X/Y/color must already be set. Uses
// unsigned DIV on the magnitude (works for -32768 since 0x8000 unsigned is its
// own magnitude). Clobbers R0-R3, R6, R7.
func (cg *CodeGenerator) emitDrawIntHelper() {
	cg.recordFuncAddr("__drawint")

	// sign = value >> 15 (logical). If set, emit '-' and negate.
	cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 0)) // R6 = value
	cg.hShrImm(6, 15)
	cg.hCmpImm(6, 0)
	posBranch := cg.hBranch(rom.EncodeBEQ())
	// emit '-'
	cg.hMovImm(7, 0x8076)
	cg.hMovImm(6, 0x2D)
	cg.builder.AddInstruction(rom.EncodeMOV(7, 7, 6)) // [0x8076] = '-'
	// value = -value
	cg.hMovImm(6, 0)
	cg.builder.AddInstruction(rom.EncodeSUB(0, 6, 0)) // R6 = 0 - value
	cg.builder.AddInstruction(rom.EncodeMOV(0, 0, 6)) // R0 = magnitude
	cg.hPatchToHere(posBranch)

	// R3 = started flag = 0
	cg.hMovImm(3, 0)

	// Extract each high place: 10000, 1000, 100, 10.
	for _, place := range []uint16{10000, 1000, 100, 10} {
		cg.hMovImm(1, place)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 2, 0)) // R2 = R0
		cg.builder.AddInstruction(rom.EncodeDIV(0, 2, 1)) // R2 = R0 / place (digit)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 2)) // R6 = digit
		cg.builder.AddInstruction(rom.EncodeMUL(0, 6, 1)) // R6 = digit * place
		cg.builder.AddInstruction(rom.EncodeSUB(0, 0, 6)) // R0 = remainder
		// emit if digit != 0 OR started != 0.
		cg.hCmpImm(2, 0)
		emit := cg.hBranch(rom.EncodeBNE()) // digit != 0 -> emit
		cg.hCmpImm(3, 0)
		skip := cg.hBranch(rom.EncodeBEQ()) // digit == 0 and not started -> skip
		cg.hPatchToHere(emit)
		cg.hMovImm(3, 1) // started = 1
		cg.hMovImm(7, 0x8076)
		cg.hMovImm(6, 0x30)
		cg.builder.AddInstruction(rom.EncodeADD(0, 6, 2)) // R6 = '0' + digit
		cg.builder.AddInstruction(rom.EncodeMOV(7, 7, 6)) // emit
		cg.hPatchToHere(skip)
	}

	// Units digit: always emit (R0 is 0..9).
	cg.hMovImm(7, 0x8076)
	cg.hMovImm(6, 0x30)
	cg.builder.AddInstruction(rom.EncodeADD(0, 6, 0)) // R6 = '0' + units
	cg.builder.AddInstruction(rom.EncodeMOV(7, 7, 6)) // emit
	cg.builder.AddInstruction(rom.EncodeRET())
}

// emitWriteIOByte writes a constant byte to an I/O address.
func (cg *CodeGenerator) emitWriteIOByte(addr uint16, val uint8) {
	cg.hMovImm(0, uint16(val))
	cg.storeIOByte(addr, 0)
}

// emitLoadBitmap emits the full bitmap-plane upload: palette -> CGRAM, plane
// control for bitmap source mode, and a chunked DMA of the bitmap from ROM.
func (cg *CodeGenerator) emitLoadBitmap(img *ImageAsset, channel uint8) {
	// Palette -> CGRAM (paletteBank*16 + i).
	for i, color := range img.Palette {
		if i >= 16 {
			break
		}
		idx := img.PaletteBank*16 + uint8(i)
		cg.emitWriteIOByte(0x8012, idx)
		cg.emitWriteIOByte(0x8013, uint8(color&0xFF))
		cg.emitWriteIOByte(0x8013, uint8(color>>8))
	}

	// Plane control: enable | size<<1 | bitmap(0x08) | paletteBank<<4.
	var sizeCode uint8
	switch img.PlaneSize {
	case 64:
		sizeCode = 1
	case 128:
		sizeCode = 2
	}
	control := uint8(0x01) | (sizeCode << 1) | 0x08 | (img.PaletteBank << 4)

	cg.emitWriteIOByte(0x8080, channel) // select plane
	cg.emitWriteIOByte(0x8081, control) // plane control (bitmap mode)
	cg.emitWriteIOByte(0x808C, 0x00)    // flags: opaque
	cg.emitWriteIOByte(0x8088, 0x00)    // reset bitmap dest offset
	cg.emitWriteIOByte(0x8089, 0x00)
	cg.emitWriteIOByte(0x808A, 0x00)

	// Chunked DMA from ROM (split at bank boundaries).
	remaining := len(img.Bitmap)
	bank := img.Bank
	srcOff := img.Offset
	var destOff uint32
	for remaining > 0 {
		avail := 0x10000 - int(srcOff)
		count := remaining
		if count > avail {
			count = avail
		}
		cg.emitWriteIOByte(0x8080, channel)
		cg.emitWriteIOByte(0x8088, uint8(destOff&0xFF))
		cg.emitWriteIOByte(0x8089, uint8((destOff>>8)&0xFF))
		cg.emitWriteIOByte(0x808A, uint8((destOff>>16)&0x07))
		cg.emitWriteIOByte(0x8061, bank)
		cg.emitWriteIOByte(0x8062, uint8(srcOff&0xFF))
		cg.emitWriteIOByte(0x8063, uint8(srcOff>>8))
		cg.emitWriteIOByte(0x8064, 0x00)
		cg.emitWriteIOByte(0x8065, 0x00)
		cg.emitWriteIOByte(0x8066, uint8(count&0xFF))
		cg.emitWriteIOByte(0x8067, uint8(count>>8))
		cg.emitWriteIOByte(0x8060, 0x15) // trigger DMA
		// Wait for DMA idle: loop while [0x8060] != 0.
		loopStart := cg.builder.GetCodeLength()
		cg.hMovImm(7, 0x8060)
		cg.builder.AddInstruction(rom.EncodeMOV(2, 2, 7)) // R2 = [0x8060]
		cg.hCmpImm(2, 0)
		cg.builder.AddInstruction(rom.EncodeBNE())
		fromPC := uint16(cg.builder.GetCodeLength() * 2)
		cg.builder.AddImmediate(uint16(cg.calcBranchOffset(fromPC, uint16(loopStart*2))))

		destOff += uint32(count)
		remaining -= count
		bank++
		srcOff = rom.ROMBankOffsetBase
	}
}
