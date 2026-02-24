package corelx

import (
	"fmt"
	"strings"

	"nitro-core-dx/internal/rom"
)

// CodeGenerator generates Nitro Core DX machine code from AST
type CodeGenerator struct {
	program          *Program
	builder          *rom.ROMBuilder
	symbols          map[string]*Symbol
	regAlloc         *RegisterAllocator
	labelCounter     int
	assets           map[string]*AssetDecl
	normalizedAssets map[string]AssetIR
	assetOffsets     map[string]uint16

	// Variable storage tracking
	variables   map[string]*VariableInfo
	varCounter  int
	stackOffset uint16 // Current stack offset for spilled variables
}

// VariableInfo tracks where a variable is stored
type VariableInfo struct {
	Name      string
	Location  VariableLocation
	RegIndex  uint8  // If in register
	StackAddr uint16 // If on stack
}

// VariableLocation indicates where variable is stored
type VariableLocation int

const (
	VarLocationRegister VariableLocation = iota
	VarLocationStack
	VarLocationMemory
)

// RegisterAllocator manages register allocation
type RegisterAllocator struct {
	registers [8]bool  // R0-R7 usage
	spill     []string // Spilled variables
}

// NewCodeGenerator creates a new code generator
func NewCodeGenerator(program *Program, builder *rom.ROMBuilder) *CodeGenerator {
	return &CodeGenerator{
		program:          program,
		builder:          builder,
		symbols:          make(map[string]*Symbol),
		regAlloc:         &RegisterAllocator{},
		labelCounter:     0,
		assets:           make(map[string]*AssetDecl),
		normalizedAssets: make(map[string]AssetIR),
		assetOffsets:     make(map[string]uint16),
		variables:        make(map[string]*VariableInfo),
		varCounter:       0,
		stackOffset:      0x1FFF, // Start at top of stack (grows downward)
	}
}

// SetNormalizedAssets injects compiler-normalized assets so codegen can avoid re-parsing source asset text.
func (cg *CodeGenerator) SetNormalizedAssets(assets []AssetIR) {
	for _, a := range assets {
		cg.normalizedAssets[a.Name] = a
	}
}

// Generate generates code for the program
func (cg *CodeGenerator) Generate() error {
	// Collect assets
	for _, asset := range cg.program.Assets {
		cg.assets[asset.Name] = asset
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

	// Add entry function first, then others
	if entryFunction != nil {
		functions = append(functions, entryFunction)
	}
	for _, fn := range cg.program.Functions {
		if fn != entryFunction {
			functions = append(functions, fn)
		}
	}

	// Generate code for each function
	for _, fn := range functions {
		if err := cg.generateFunction(fn); err != nil {
			return err
		}
	}

	return nil
}

func (cg *CodeGenerator) generateFunction(fn *FunctionDecl) error {
	// Reset variable tracking for each function
	cg.variables = make(map[string]*VariableInfo)
	cg.regAlloc = &RegisterAllocator{}
	cg.stackOffset = 0x1FFF // Reset stack for each function

	// Function prologue
	// For now, we'll use a simple calling convention:
	// - Parameters passed in R0-R7
	// - Return value in R0
	// - Caller saves registers

	// Generate function body
	for _, stmt := range fn.Body {
		if err := cg.generateStmt(stmt); err != nil {
			return err
		}
	}

	// Function epilogue
	if fn.ReturnType == nil {
		// Void return - just return
		cg.builder.AddInstruction(rom.EncodeRET())
	} else {
		// Return value should already be in R0
		cg.builder.AddInstruction(rom.EncodeRET())
	}

	return nil
}

func (cg *CodeGenerator) generateStmt(stmt Stmt) error {
	switch s := stmt.(type) {
	case *VarDeclStmt:
		return cg.generateVarDecl(s)

	case *AssignStmt:
		return cg.generateAssign(s)

	case *IfStmt:
		return cg.generateIf(s)

	case *WhileStmt:
		return cg.generateWhile(s)

	case *ForStmt:
		return cg.generateFor(s)

	case *ReturnStmt:
		return cg.generateReturn(s)

	case *ExprStmt:
		return cg.generateExpr(s.Expr, 0) // Discard result

	default:
		return fmt.Errorf("unsupported statement type: %T", stmt)
	}
}

func (cg *CodeGenerator) generateVarDecl(stmt *VarDeclStmt) error {
	// Check if initializer is a struct initialization
	if call, ok := stmt.Value.(*CallExpr); ok {
		if ident, ok := call.Func.(*IdentExpr); ok {
			knownStructs := map[string]bool{"Sprite": true, "Vec2": true}
			if knownStructs[ident.Name] {
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
				cg.stackOffset -= 2
				stackAddr := cg.stackOffset
				cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #stackAddr
				cg.builder.AddImmediate(stackAddr)
				cg.builder.AddInstruction(rom.EncodeMOV(3, 7, 0)) // MOV [R7], R0
				cg.variables[stmt.Name] = &VariableInfo{
					Name:      stmt.Name,
					Location:  VarLocationStack,
					StackAddr: stackAddr,
				}
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
	cg.stackOffset -= 2 // Allocate 2 bytes (16-bit value)
	stackAddr := cg.stackOffset
	cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #stackAddr
	cg.builder.AddImmediate(stackAddr)
	cg.builder.AddInstruction(rom.EncodeMOV(3, 7, 0)) // MOV [R7], R0
	cg.variables[stmt.Name] = &VariableInfo{
		Name:      stmt.Name,
		Location:  VarLocationStack,
		StackAddr: stackAddr,
	}
	return nil
}

func (cg *CodeGenerator) generateAssign(stmt *AssignStmt) error {
	// Generate code for value
	if err := cg.generateExpr(stmt.Value, 0); err != nil {
		return err
	}
	// Value is in R0
	// Generate code to store to target
	if member, ok := stmt.Target.(*MemberExpr); ok {
		// Struct member assignment like hero.tile = base
		if ident, ok := member.Object.(*IdentExpr); ok {
			if varInfo, exists := cg.variables[ident.Name]; exists {
				// Calculate field offset for Sprite struct
				spriteOffsets := map[string]uint16{
					"x_lo": 0, "x_hi": 1, "y": 2, "tile": 3, "attr": 4, "ctrl": 5,
				}
				vec2Offsets := map[string]uint16{
					"x": 0, "y": 2,
				}
				var offset uint16
				var found bool
				if off, ok := spriteOffsets[member.Member]; ok {
					offset = off
					found = true
				} else if off, ok := vec2Offsets[member.Member]; ok {
					offset = off
					found = true
				}
				if found {
					// Store value to struct member
					// Variable holds the struct address (either in register or on stack)
					// R0 has the value to store
					if varInfo.Location == VarLocationRegister {
						// Variable is in register, holding the struct address
						// Load struct address from register, add offset, then store member
						cg.builder.AddInstruction(rom.EncodeMOV(0, 7, varInfo.RegIndex)) // MOV R7, R{reg} (struct address)
						cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))                // MOV R6, #offset
						cg.builder.AddImmediate(offset)
						cg.builder.AddInstruction(rom.EncodeADD(0, 7, 6)) // ADD R7, R6 (member address)
						cg.builder.AddInstruction(rom.EncodeMOV(7, 7, 0)) // MOV [R7], R0 (8-bit store)
						return nil
					} else if varInfo.Location == VarLocationStack {
						// Variable is on stack, holding the struct address (16-bit)
						// Load struct address from stack, add offset, then store member
						cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #stackAddr
						cg.builder.AddImmediate(varInfo.StackAddr)
						cg.builder.AddInstruction(rom.EncodeMOV(2, 6, 7)) // MOV R6, [R7] (load struct address, 16-bit)
						cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #offset
						cg.builder.AddImmediate(offset)
						cg.builder.AddInstruction(rom.EncodeADD(0, 6, 7)) // ADD R6, R7 (member address)
						cg.builder.AddInstruction(rom.EncodeMOV(7, 6, 0)) // MOV [R6], R0 (8-bit store)
						return nil
					}
				}
			}
		}
		// Fallback: discard (would need proper struct tracking)
		return nil
	}

	// Regular assignment: x = value
	if ident, ok := stmt.Target.(*IdentExpr); ok {
		if varInfo, exists := cg.variables[ident.Name]; exists {
			// Store value to variable location
			if varInfo.Location == VarLocationRegister {
				cg.builder.AddInstruction(rom.EncodeMOV(0, varInfo.RegIndex, 0)) // MOV R{reg}, R0
			} else if varInfo.Location == VarLocationStack {
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

	// Branch if false
	elseLabel := cg.newLabel()
	cg.builder.AddInstruction(rom.EncodeBEQ()) // BEQ else_label
	elseOffsetPos := cg.builder.GetCodeLength()
	cg.builder.AddImmediate(0) // Placeholder

	// Generate then block
	for _, s := range stmt.Then {
		if err := cg.generateStmt(s); err != nil {
			return err
		}
	}

	// Jump past else
	endLabel := cg.newLabel()
	cg.builder.AddInstruction(rom.EncodeJMP())
	endOffsetPos := cg.builder.GetCodeLength()
	cg.builder.AddImmediate(0) // Placeholder

	// Generate else block
	cg.patchLabel(elseLabel, elseOffsetPos)
	for _, clause := range stmt.ElseIf {
		// Generate elseif condition
		if err := cg.generateExpr(clause.Condition, 0); err != nil {
			return err
		}
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
		cg.builder.AddImmediate(0)
		cg.builder.AddInstruction(rom.EncodeCMP(0, 0, 7))
		elseIfEnd := cg.newLabel()
		cg.builder.AddInstruction(rom.EncodeBEQ())
		elseIfOffsetPos := cg.builder.GetCodeLength()
		cg.builder.AddImmediate(0)

		for _, s := range clause.Body {
			if err := cg.generateStmt(s); err != nil {
				return err
			}
		}

		cg.builder.AddInstruction(rom.EncodeJMP())
		elseIfEndOffsetPos := cg.builder.GetCodeLength()
		cg.builder.AddImmediate(0)
		cg.patchLabel(elseIfEnd, elseIfEndOffsetPos)
		cg.patchLabel(elseIfEnd, elseIfOffsetPos)
	}

	for _, s := range stmt.Else {
		if err := cg.generateStmt(s); err != nil {
			return err
		}
	}

	cg.patchLabel(endLabel, endOffsetPos)
	return nil
}

func (cg *CodeGenerator) generateWhile(stmt *WhileStmt) error {
	loopStartPos := cg.builder.GetCodeLength()

	// Generate condition
	if err := cg.generateExpr(stmt.Condition, 0); err != nil {
		return err
	}

	// Compare with 0
	cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
	cg.builder.AddImmediate(0)
	cg.builder.AddInstruction(rom.EncodeCMP(0, 0, 7))

	// Branch if false (exit loop)
	loopEnd := cg.newLabel()
	cg.builder.AddInstruction(rom.EncodeBEQ())
	loopEndOffsetPos := cg.builder.GetCodeLength()
	cg.builder.AddImmediate(0) // Placeholder

	// Generate body
	for _, s := range stmt.Body {
		if err := cg.generateStmt(s); err != nil {
			return err
		}
	}

	// Jump back to start
	cg.builder.AddInstruction(rom.EncodeJMP())
	currentPC := uint16(cg.builder.GetCodeLength() * 2)
	offset := rom.CalculateBranchOffset(currentPC, uint16(loopStartPos*2))
	cg.builder.AddImmediate(uint16(offset))

	// Patch loop end
	cg.patchLabel(loopEnd, loopEndOffsetPos)
	return nil
}

func (cg *CodeGenerator) generateFor(stmt *ForStmt) error {
	// Generate init
	if stmt.Init != nil {
		if err := cg.generateStmt(stmt.Init); err != nil {
			return err
		}
	}

	loopStartPos := cg.builder.GetCodeLength()

	// Generate condition
	if err := cg.generateExpr(stmt.Condition, 0); err != nil {
		return err
	}

	// Compare with 0
	cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
	cg.builder.AddImmediate(0)
	cg.builder.AddInstruction(rom.EncodeCMP(0, 0, 7))

	// Branch if false (exit loop)
	loopEnd := cg.newLabel()
	cg.builder.AddInstruction(rom.EncodeBEQ())
	loopEndOffsetPos := cg.builder.GetCodeLength()
	cg.builder.AddImmediate(0) // Placeholder

	// Generate body
	for _, s := range stmt.Body {
		if err := cg.generateStmt(s); err != nil {
			return err
		}
	}

	// Generate post
	if stmt.Post != nil {
		if err := cg.generateStmt(stmt.Post); err != nil {
			return err
		}
	}

	// Jump back to start
	cg.builder.AddInstruction(rom.EncodeJMP())
	currentPC := uint16(cg.builder.GetCodeLength() * 2)
	offset := rom.CalculateBranchOffset(currentPC, uint16(loopStartPos*2))
	cg.builder.AddImmediate(uint16(offset))

	// Patch loop end
	cg.patchLabel(loopEnd, loopEndOffsetPos)
	return nil
}

func (cg *CodeGenerator) generateReturn(stmt *ReturnStmt) error {
	if stmt.Value != nil {
		if err := cg.generateExpr(stmt.Value, 0); err != nil {
			return err
		}
		// Value is in R0
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
				// Return asset offset (for now, just 0)
				cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
				cg.builder.AddImmediate(0)
				return nil
			}
		}
		// Variable access
		if varInfo, exists := cg.variables[e.Name]; exists {
			// Load from variable location
			if varInfo.Location == VarLocationRegister {
				cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, varInfo.RegIndex)) // MOV R{destReg}, R{reg}
			} else if varInfo.Location == VarLocationStack {
				// Load from stack
				cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #stackAddr
				cg.builder.AddImmediate(varInfo.StackAddr)
				cg.builder.AddInstruction(rom.EncodeMOV(2, destReg, 7)) // MOV R{destReg}, [R7]
			}
			return nil
		}
		// Variable not found - might be a built-in or error
		// For now, return 0 (would be caught in semantic analysis)
		cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
		cg.builder.AddImmediate(0)
		return nil

	case *BinaryExpr:
		// Generate left operand
		if err := cg.generateExpr(e.Left, destReg); err != nil {
			return err
		}
		// Save left result
		cg.builder.AddInstruction(rom.EncodeMOV(0, 1, destReg)) // MOV R1, R{destReg}
		// Generate right operand
		if err := cg.generateExpr(e.Right, 2); err != nil {
			return err
		}
		// Perform operation
		switch e.Op {
		case TOKEN_PLUS:
			// Restore left result to destReg, then add
			cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 1)) // MOV R{destReg}, R1 (restore left)
			cg.builder.AddInstruction(rom.EncodeADD(0, destReg, 2)) // ADD R{destReg}, R2
		case TOKEN_MINUS:
			// Restore left result to destReg, then subtract
			cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 1)) // MOV R{destReg}, R1 (restore left)
			cg.builder.AddInstruction(rom.EncodeSUB(0, destReg, 2)) // SUB R{destReg}, R2
		case TOKEN_STAR:
			// Multiplication - use repeated addition for small values
			// For now, only support multiplication by powers of 2 (shifts)
			// General multiplication would need a helper function
			// Check if right is a power of 2
			if numExpr, ok := e.Right.(*NumberExpr); ok {
				val := numExpr.Value
				if val == 1 {
					// x * 1 = x, already in destReg
					return nil
				}
				if val == 2 {
					// x * 2 = x << 1
					cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
					cg.builder.AddImmediate(1)
					cg.builder.AddInstruction(rom.EncodeSHL(0, destReg, 7))
					return nil
				}
				if val == 4 {
					cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
					cg.builder.AddImmediate(2)
					cg.builder.AddInstruction(rom.EncodeSHL(0, destReg, 7))
					return nil
				}
				if val == 8 {
					cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
					cg.builder.AddImmediate(3)
					cg.builder.AddInstruction(rom.EncodeSHL(0, destReg, 7))
					return nil
				}
			}
			return fmt.Errorf("multiplication by non-power-of-2 not yet implemented")
		case TOKEN_SLASH:
			// Division - use repeated subtraction for small values
			// For now, only support division by powers of 2 (shifts)
			if numExpr, ok := e.Right.(*NumberExpr); ok {
				val := numExpr.Value
				if val == 1 {
					// x / 1 = x
					return nil
				}
				if val == 2 {
					cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
					cg.builder.AddImmediate(1)
					cg.builder.AddInstruction(rom.EncodeSHR(0, destReg, 7))
					return nil
				}
				if val == 4 {
					cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
					cg.builder.AddImmediate(2)
					cg.builder.AddInstruction(rom.EncodeSHR(0, destReg, 7))
					return nil
				}
				if val == 8 {
					cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
					cg.builder.AddImmediate(3)
					cg.builder.AddInstruction(rom.EncodeSHR(0, destReg, 7))
					return nil
				}
			}
			return fmt.Errorf("division by non-power-of-2 not yet implemented")
		case TOKEN_PERCENT:
			// Modulo: a % b = a - (a / b) * b
			// For now, simplified - only support modulo by powers of 2
			if numExpr, ok := e.Right.(*NumberExpr); ok {
				val := numExpr.Value
				if val == 2 {
					// x % 2 = x & 1
					cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
					cg.builder.AddImmediate(1)
					cg.builder.AddInstruction(rom.EncodeAND(0, destReg, 7))
					return nil
				}
				if val == 4 {
					// x % 4 = x & 3
					cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
					cg.builder.AddImmediate(3)
					cg.builder.AddInstruction(rom.EncodeAND(0, destReg, 7))
					return nil
				}
				if val == 8 {
					// x % 8 = x & 7
					cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
					cg.builder.AddImmediate(7)
					cg.builder.AddInstruction(rom.EncodeAND(0, destReg, 7))
					return nil
				}
				if val == 16 {
					// x % 16 = x & 15
					cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
					cg.builder.AddImmediate(15)
					cg.builder.AddInstruction(rom.EncodeAND(0, destReg, 7))
					return nil
				}
				if val == 60 {
					// x % 60 - use repeated subtraction
					// Subtract 60 until less than 60
					modEnd := cg.newLabel()
					modStartPos := cg.builder.GetCodeLength()
					// Compare with 60
					cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
					cg.builder.AddImmediate(60)
					cg.builder.AddInstruction(rom.EncodeCMP(0, destReg, 7))
					cg.builder.AddInstruction(rom.EncodeBLT()) // BLT modEnd
					modEndPos := cg.builder.GetCodeLength()
					cg.builder.AddImmediate(0)
					// Subtract 60
					cg.builder.AddInstruction(rom.EncodeSUB(0, destReg, 7))
					// Jump back
					cg.builder.AddInstruction(rom.EncodeJMP())
					currentPC := uint16(cg.builder.GetCodeLength() * 2)
					offset := rom.CalculateBranchOffset(currentPC, uint16(modStartPos*2))
					cg.builder.AddImmediate(uint16(offset))
					cg.patchLabel(modEnd, modEndPos)
					return nil
				}
			}
			return fmt.Errorf("modulo by non-power-of-2 or 60 not yet implemented")
		case TOKEN_EQUAL_EQUAL:
			// Compare and set result: 1 if equal, 0 if not.
			// Important: branch immediately after CMP (MOV updates flags).
			cg.builder.AddInstruction(rom.EncodeCMP(0, 1, 2)) // CMP R1, R2
			falseLabel := cg.newLabel()
			endLabel := cg.newLabel()
			cg.builder.AddInstruction(rom.EncodeBNE()) // BNE false
			falsePos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0)) // true => 1
			cg.builder.AddImmediate(1)
			cg.builder.AddInstruction(rom.EncodeJMP()) // JMP end
			endPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.patchLabel(falseLabel, falsePos)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0)) // false => 0
			cg.builder.AddImmediate(0)
			cg.patchLabel(endLabel, endPos)
			return nil
		case TOKEN_BANG_EQUAL:
			// Compare and set result: 1 if not equal, 0 if equal.
			cg.builder.AddInstruction(rom.EncodeCMP(0, 1, 2))
			falseLabel := cg.newLabel()
			endLabel := cg.newLabel()
			cg.builder.AddInstruction(rom.EncodeBEQ()) // BEQ false (equal => false)
			falsePos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0)) // true => 1
			cg.builder.AddImmediate(1)
			cg.builder.AddInstruction(rom.EncodeJMP())
			endPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.patchLabel(falseLabel, falsePos)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0)) // false => 0
			cg.builder.AddImmediate(0)
			cg.patchLabel(endLabel, endPos)
			return nil
		case TOKEN_LESS:
			cg.builder.AddInstruction(rom.EncodeCMP(0, 1, 2))
			falseLabel := cg.newLabel()
			endLabel := cg.newLabel()
			cg.builder.AddInstruction(rom.EncodeBGE()) // >= => false
			falsePos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0)) // true => 1
			cg.builder.AddImmediate(1)
			cg.builder.AddInstruction(rom.EncodeJMP())
			endPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.patchLabel(falseLabel, falsePos)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0)) // false => 0
			cg.builder.AddImmediate(0)
			cg.patchLabel(endLabel, endPos)
			return nil
		case TOKEN_GREATER:
			cg.builder.AddInstruction(rom.EncodeCMP(0, 1, 2))
			falseLabel := cg.newLabel()
			endLabel := cg.newLabel()
			cg.builder.AddInstruction(rom.EncodeBLE()) // <= => false
			falsePos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(1)
			cg.builder.AddInstruction(rom.EncodeJMP())
			endPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.patchLabel(falseLabel, falsePos)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(0)
			cg.patchLabel(endLabel, endPos)
			return nil
		case TOKEN_LESS_EQUAL:
			cg.builder.AddInstruction(rom.EncodeCMP(0, 1, 2))
			falseLabel := cg.newLabel()
			endLabel := cg.newLabel()
			cg.builder.AddInstruction(rom.EncodeBGT()) // > => false
			falsePos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(1)
			cg.builder.AddInstruction(rom.EncodeJMP())
			endPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.patchLabel(falseLabel, falsePos)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(0)
			cg.patchLabel(endLabel, endPos)
			return nil
		case TOKEN_GREATER_EQUAL:
			cg.builder.AddInstruction(rom.EncodeCMP(0, 1, 2))
			falseLabel := cg.newLabel()
			endLabel := cg.newLabel()
			cg.builder.AddInstruction(rom.EncodeBLT()) // < => false
			falsePos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(1)
			cg.builder.AddInstruction(rom.EncodeJMP())
			endPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.patchLabel(falseLabel, falsePos)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(0)
			cg.patchLabel(endLabel, endPos)
			return nil
		case TOKEN_AND:
			// Logical AND: both must be non-zero
			// R1 already has left, R2 has right
			// Set R0 to 1 if both non-zero, else 0
			cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeCMP(0, 1, 7)) // CMP R1, R7
			cg.builder.AddInstruction(rom.EncodeBEQ())        // BEQ false
			falseLabel1 := cg.newLabel()
			falsePos1 := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeCMP(0, 2, 7)) // CMP R2, R7
			cg.builder.AddInstruction(rom.EncodeBEQ())        // BEQ false
			falseLabel2 := cg.newLabel()
			falsePos2 := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			// Both non-zero, set to 1
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(1)
			endLabel := cg.newLabel()
			cg.builder.AddInstruction(rom.EncodeJMP())
			endPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			// False case
			cg.patchLabel(falseLabel1, falsePos1)
			cg.patchLabel(falseLabel2, falsePos2)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(0)
			cg.patchLabel(endLabel, endPos)
			return nil
		case TOKEN_OR:
			// Logical OR: at least one non-zero
			cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeCMP(0, 1, 7))
			cg.builder.AddInstruction(rom.EncodeBNE()) // BNE true
			trueLabel1 := cg.newLabel()
			truePos1 := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeCMP(0, 2, 7))
			cg.builder.AddInstruction(rom.EncodeBNE()) // BNE true
			trueLabel2 := cg.newLabel()
			truePos2 := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			// Both zero, set to 0
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(0)
			endLabel := cg.newLabel()
			cg.builder.AddInstruction(rom.EncodeJMP())
			endPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			// True case
			cg.patchLabel(trueLabel1, truePos1)
			cg.patchLabel(trueLabel2, truePos2)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(1)
			cg.patchLabel(endLabel, endPos)
			return nil
		case TOKEN_PIPE:
			// Bitwise OR: left result is in R1, right result is in R2
			// OR R1, R2 -> result in R1, then move to destReg
			cg.builder.AddInstruction(rom.EncodeOR(0, 1, 2))        // OR R1, R2 -> R1 = R1 | R2
			cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 1)) // MOV R{destReg}, R1
			return nil
		case TOKEN_AMPERSAND:
			// Bitwise AND: left result is in R1, right result is in R2
			// AND R1, R2 -> result in R1, then move to destReg
			cg.builder.AddInstruction(rom.EncodeAND(0, 1, 2))       // AND R1, R2 -> R1 = R1 & R2
			cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 1)) // MOV R{destReg}, R1
			return nil
		case TOKEN_CARET:
			// Bitwise XOR
			cg.builder.AddInstruction(rom.EncodeXOR(0, destReg, 2))
			return nil
		case TOKEN_LSHIFT:
			// Left shift
			cg.builder.AddInstruction(rom.EncodeSHL(0, destReg, 2))
			return nil
		case TOKEN_RSHIFT:
			// Right shift
			cg.builder.AddInstruction(rom.EncodeSHR(0, destReg, 2))
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
			if varInfo, exists := cg.variables[ident.Name]; exists {
				// It's a struct member access like hero.tile or sprite.tile
				// Calculate field offset
				spriteOffsets := map[string]uint16{
					"x_lo": 0, "x_hi": 1, "y": 2, "tile": 3, "attr": 4, "ctrl": 5,
				}
				vec2Offsets := map[string]uint16{
					"x": 0, "y": 2,
				}
				var offset uint16
				var found bool
				if off, ok := spriteOffsets[e.Member]; ok {
					offset = off
					found = true
				} else if off, ok := vec2Offsets[e.Member]; ok {
					offset = off
					found = true
				}
				if found {
					// Load member from struct
					// Variable holds the struct address (either in register or on stack)
					if varInfo.Location == VarLocationRegister {
						// Variable is in register, holding the struct address
						// Load struct address from register, add offset, then load member
						cg.builder.AddInstruction(rom.EncodeMOV(0, 7, varInfo.RegIndex)) // MOV R7, R{reg} (struct address)
						cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0))                // MOV R6, #offset
						cg.builder.AddImmediate(offset)
						cg.builder.AddInstruction(rom.EncodeADD(0, 7, 6))       // ADD R7, R6 (member address)
						cg.builder.AddInstruction(rom.EncodeMOV(6, destReg, 7)) // MOV R{destReg}, [R7] (8-bit load)
						return nil
					} else if varInfo.Location == VarLocationStack {
						// Variable is on stack, holding the struct address (16-bit)
						// Load struct address from stack, add offset, then load member
						cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #stackAddr
						cg.builder.AddImmediate(varInfo.StackAddr)
						cg.builder.AddInstruction(rom.EncodeMOV(2, 6, 7)) // MOV R6, [R7] (load struct address, 16-bit)
						cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #offset
						cg.builder.AddImmediate(offset)
						cg.builder.AddInstruction(rom.EncodeADD(0, 6, 7))       // ADD R6, R7 (member address)
						cg.builder.AddInstruction(rom.EncodeMOV(6, destReg, 6)) // MOV R{destReg}, [R6] (8-bit load)
						return nil
					}
				}
				// Variable exists but member not found - return 0
				cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
				cg.builder.AddImmediate(0)
				return nil
			}
			// Variable doesn't exist - this is an error for member access
			// (Namespace calls like ppu.enable_display() are handled in generateCall)
		}
		// Fallback: generate object and return placeholder
		if err := cg.generateExpr(e.Object, destReg); err != nil {
			return err
		}
		cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
		cg.builder.AddImmediate(0)
		return nil

	case *UnaryExpr:
		return cg.generateUnary(e, destReg)

	default:
		return fmt.Errorf("unsupported expression type: %T", expr)
	}
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

	if funcName == "" {
		return fmt.Errorf("cannot determine function name in call")
	}

	// Generate arguments (simplified - pass in R0-R7)
	for i, arg := range call.Args {
		if i >= 8 {
			return fmt.Errorf("too many arguments (max 8)")
		}
		if err := cg.generateExpr(arg, uint8(i)); err != nil {
			return err
		}
	}

	// Try built-in functions first
	if err := cg.generateBuiltinCall(funcName, call.Args, destReg); err == nil {
		return nil
	}

	// Check if it's a user-defined function
	if fn := cg.findFunction(funcName); fn != nil {
		// For now, user functions are not fully implemented
		// In a full implementation, we'd:
		// 1. Push return address to stack
		// 2. Set up parameters
		// 3. CALL function
		// 4. Get return value from R0
		return fmt.Errorf("user-defined function calls not fully implemented: %s", funcName)
	}

	// Handle struct initialization like Sprite() or Vec2()
	// Check if it's a known struct type
	knownStructs := map[string]bool{
		"Sprite": true, "Vec2": true,
	}
	if knownStructs[funcName] {
		// Struct initialization creates a zero-initialized struct
		// Allocate stack space for struct (Sprite = 6 bytes)
		structSize := uint16(6) // Sprite struct is 6 bytes
		if funcName == "Vec2" {
			structSize = 4 // Vec2 is 2 i16s = 4 bytes
		}
		cg.stackOffset -= structSize
		stackAddr := cg.stackOffset

		// Initialize struct to zero
		// Zero out struct bytes
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #stackAddr
		cg.builder.AddImmediate(stackAddr)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0
		cg.builder.AddImmediate(0)
		// Zero out struct bytes (simplified - just zero first byte, rest will be zero-initialized)
		cg.builder.AddInstruction(rom.EncodeMOV(7, 7, 6)) // MOV [R7], R6 (8-bit store, mode 7)

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
	switch name {
	case "wait_vblank":
		// Wait for VBlank flag (0x803E, bit 0 = 1 means VBlank)
		// Pattern from manual: Read flag, AND with 0x01, CMP with 0, BEQ if 0 (keep waiting)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x803E
		cg.builder.AddImmediate(0x803E)
		waitPos := cg.builder.GetCodeLength()
		cg.builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (read flag)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0x01
		cg.builder.AddImmediate(0x01)
		cg.builder.AddInstruction(rom.EncodeAND(0, 5, 7)) // AND R5, R7 (mask to bit 0)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
		cg.builder.AddImmediate(0)
		cg.builder.AddInstruction(rom.EncodeCMP(0, 5, 7)) // CMP R5, R7 (compare with 0)
		cg.builder.AddInstruction(rom.EncodeBEQ())        // BEQ waitPos (if equal to 0, keep waiting)
		currentPC := uint16(cg.builder.GetCodeLength() * 2)
		offset := rom.CalculateBranchOffset(currentPC, uint16(waitPos*2))
		cg.builder.AddImmediate(uint16(offset))
		return nil

	case "frame_counter":
		// frame_counter() -> u32 (returns 16-bit frame counter)
		// Read FRAME_COUNTER_LOW (0x803F) and FRAME_COUNTER_HIGH (0x8040)
		// Combine into 16-bit value: (high << 8) | low

		// Read low byte from 0x803F
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x803F
		cg.builder.AddImmediate(0x803F)
		cg.builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (read low byte)

		// Read high byte from 0x8040
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8040
		cg.builder.AddImmediate(0x8040)
		cg.builder.AddInstruction(rom.EncodeMOV(2, 6, 4)) // MOV R6, [R4] (read high byte)

		// Combine: (high << 8) | low
		cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 6)) // MOV R7, R6 (copy high byte)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #8
		cg.builder.AddImmediate(8)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 7, 4)) // SHL R7, R4 -> R7 = high << 8
		cg.builder.AddInstruction(rom.EncodeOR(0, 5, 7))  // OR R5, R7 -> R5 = (high << 8) | low

		// Return value in destReg
		cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 5)) // MOV R{destReg}, R5
		return nil

	case "sprite.set_pos":
		// sprite.set_pos(s: *Sprite, x: i16, y: u8)
		// Args: R0 = sprite pointer, R1 = x (i16), R2 = y (u8)
		// Store x and y to sprite struct
		// R0 has sprite address (from &hero), R1 has x, R2 has y
		// Write x_lo (low byte of x)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 3, 0)) // MOV R3, R0 (save sprite addr)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 4, 1)) // MOV R4, R1 (save x)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 5, 2)) // MOV R5, R2 (save y)

		// Write x_lo (offset 0) - low byte of x
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0xFF
		cg.builder.AddImmediate(0xFF)
		cg.builder.AddInstruction(rom.EncodeAND(0, 4, 6)) // AND R4, R6 (mask to low byte)
		cg.builder.AddInstruction(rom.EncodeMOV(7, 3, 4)) // MOV [R3], R4 (8-bit store x_lo)

		// Write x_hi (offset 1) - high byte (sign bit)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #1
		cg.builder.AddImmediate(1)
		cg.builder.AddInstruction(rom.EncodeADD(0, 3, 6)) // ADD R3, R6 (increment to offset 1)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 4)) // MOV R6, R4 (copy x)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #8
		cg.builder.AddImmediate(8)
		cg.builder.AddInstruction(rom.EncodeSHR(0, 6, 7)) // SHR R6, R7 -> R6 = x >> 8 (high byte)
		cg.builder.AddInstruction(rom.EncodeMOV(7, 3, 6)) // MOV [R3], R6 (write x_hi)
		// Write y (offset 2)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #1
		cg.builder.AddImmediate(1)
		cg.builder.AddInstruction(rom.EncodeADD(0, 3, 6)) // ADD R3, R6 (increment to offset 2)
		cg.builder.AddInstruction(rom.EncodeMOV(7, 3, 5)) // MOV [R3], R5 (write y)
		return nil

	case "oam.write":
		// oam.write(id: u8, s: *Sprite)
		// Args: R0 = sprite id, R1 = sprite pointer
		// Set OAM_ADDR to id * 6, then write sprite data from struct to OAM_DATA

		// Save sprite pointer (R1) to R3
		cg.builder.AddInstruction(rom.EncodeMOV(0, 3, 1)) // MOV R3, R1 (sprite pointer)

		// Calculate OAM address: id * 6
		cg.builder.AddInstruction(rom.EncodeMOV(0, 2, 0)) // MOV R2, R0 (save id)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 2)) // MOV R6, R2 (copy id)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #1
		cg.builder.AddImmediate(1)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 6, 7)) // SHL R6, R7 -> R6 = id*2
		cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 2)) // MOV R7, R2 (id)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #2
		cg.builder.AddImmediate(2)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 7, 5)) // SHL R7, R5 -> R7 = id*4
		cg.builder.AddInstruction(rom.EncodeADD(0, 6, 7)) // ADD R6, R7 -> R6 = id*2 + id*4 = id*6

		// Set OAM_ADDR (0x8014)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8014
		cg.builder.AddImmediate(0x8014)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 5, 6)) // MOV R5, R6 (OAM offset)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write OAM_ADDR)

		// Set OAM_DATA address (0x8015)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8015
		cg.builder.AddImmediate(0x8015)

		// Read sprite struct and write to OAM_DATA
		// Sprite format: x_lo (offset 0), x_hi (offset 1), y (offset 2), tile (offset 3), attr (offset 4), ctrl (offset 5)
		// R3 = sprite pointer (save original in R7 for later use if needed)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 3)) // MOV R7, R3 (save original sprite pointer)

		// Write x_lo (offset 0)
		cg.builder.AddInstruction(rom.EncodeMOV(6, 5, 3)) // MOV R5, [R3] (8-bit load, mode 6)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write to OAM_DATA)

		// Write x_hi (offset 1)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #1
		cg.builder.AddImmediate(1)
		cg.builder.AddInstruction(rom.EncodeADD(0, 3, 6)) // ADD R3, R6 (increment to offset 1)
		cg.builder.AddInstruction(rom.EncodeMOV(6, 5, 3)) // MOV R5, [R3]
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

		// Write y (offset 2)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #1
		cg.builder.AddImmediate(1)
		cg.builder.AddInstruction(rom.EncodeADD(0, 3, 6)) // ADD R3, R6 (increment to offset 2)
		cg.builder.AddInstruction(rom.EncodeMOV(6, 5, 3)) // MOV R5, [R3]
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

		// Write tile (offset 3)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #1
		cg.builder.AddImmediate(1)
		cg.builder.AddInstruction(rom.EncodeADD(0, 3, 6)) // ADD R3, R6 (increment to offset 3)
		cg.builder.AddInstruction(rom.EncodeMOV(6, 5, 3)) // MOV R5, [R3]
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

		// Write attr (offset 4)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #1
		cg.builder.AddImmediate(1)
		cg.builder.AddInstruction(rom.EncodeADD(0, 3, 6)) // ADD R3, R6 (increment to offset 4)
		cg.builder.AddInstruction(rom.EncodeMOV(6, 5, 3)) // MOV R5, [R3]
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

		// Write ctrl (offset 5)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #1
		cg.builder.AddImmediate(1)
		cg.builder.AddInstruction(rom.EncodeADD(0, 3, 6)) // ADD R3, R6 (increment to offset 5)
		cg.builder.AddInstruction(rom.EncodeMOV(6, 5, 3)) // MOV R5, [R3]
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

		return nil

	case "oam.write_sprite_data":
		// oam.write_sprite_data(id: u8, x: i16, y: u8, tile: u8, attr: u8, ctrl: u8)
		// Args: R0=id, R1=x, R2=y, R3=tile, R4=attr, R5=ctrl
		// Preserve y in R6 and keep attr/ctrl in R4/R5 by using R0/R2/R7 as temporaries.
		if len(args) != 6 {
			return fmt.Errorf("oam.write_sprite_data requires 6 arguments")
		}

		idReg := uint8(0)
		xReg := uint8(1)
		yReg := uint8(2)
		tileReg := uint8(3)
		attrReg := uint8(4)
		ctrlReg := uint8(5)
		// Save y (R2) because R2 will be reused for x temp values
		cg.builder.AddInstruction(rom.EncodeMOV(0, 6, yReg)) // MOV R6, R2 (save y)

		// Set OAM_ADDR to sprite ID (0-127), NOT id * 6
		// The PPU internally multiplies by 6 to get the byte offset
		// Write sprite ID directly to OAM_ADDR (0x8014)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 2, 0)) // MOV R2, #0x8014
		cg.builder.AddImmediate(0x8014)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 2, idReg)) // MOV [R2], R0 (sprite ID)

		// Write sprite data to OAM_DATA (0x8015) using R0 as pointer (id no longer needed)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 0, 0)) // MOV R0, #0x8015
		cg.builder.AddImmediate(0x8015)

		// X low byte (R1)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 2, xReg)) // MOV R2, R1 (x temp)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))    // MOV R7, #0xFF
		cg.builder.AddImmediate(0xFF)
		cg.builder.AddInstruction(rom.EncodeAND(0, 2, 7)) // AND R2, R7 (low byte)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 0, 2)) // MOV [R0], R2

		// X high byte: extract sign bit from x (bit 8)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 2, xReg)) // MOV R2, R1 (x)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))    // MOV R7, #8
		cg.builder.AddImmediate(8)
		cg.builder.AddInstruction(rom.EncodeSHR(0, 2, 7)) // SHR R2, R7 -> x high
		cg.builder.AddInstruction(rom.EncodeMOV(3, 0, 2)) // MOV [R0], R2

		// Y (from saved R6)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 0, 6)) // MOV [R0], R6

		// Tile (R3)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 0, tileReg)) // MOV [R0], R3

		// Attr (R4)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 0, attrReg)) // MOV [R0], R4

		// Ctrl (R5)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 0, ctrlReg)) // MOV [R0], R5
		return nil

	case "oam.clear_sprite":
		// oam.clear_sprite(id: u8)
		// Args: R0 = sprite id
		// Disables sprite by setting control byte to 0
		if len(args) != 1 {
			return fmt.Errorf("oam.clear_sprite requires 1 argument")
		}

		idReg := uint8(0)

		// Set OAM_ADDR to sprite ID, then write 0 to control byte (byte 5)
		// Write sprite ID to OAM_ADDR (0x8014)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8014
		cg.builder.AddImmediate(0x8014)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 6, idReg)) // MOV R6, R0 (sprite ID)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 6))     // MOV [R4], R6 (write sprite ID to OAM_ADDR)

		// Set OAM_ADDR to sprite ID again, but this time we need to write to byte 5 (control)
		// The PPU uses OAM_ADDR * 6 + byte_index, so we need to write 5 dummy bytes first
		// Actually, simpler: just write 0 to all 6 bytes to completely disable the sprite
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8015
		cg.builder.AddImmediate(0x8015)
		// Write 0 to all 6 bytes (X_low, X_high, Y, Tile, Attr, Ctrl)
		for i := 0; i < 6; i++ {
			cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 6)) // MOV [R4], R6
		}
		return nil

	case "SPR_PAL":
		// SPR_PAL(p: u8) -> u8
		// Returns palette value (p & 0x0F)
		if len(args) != 1 {
			return fmt.Errorf("SPR_PAL requires 1 argument")
		}
		// Arg is in R0, mask to 4 bits
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0x0F
		cg.builder.AddImmediate(0x0F)
		cg.builder.AddInstruction(rom.EncodeAND(0, destReg, 7)) // AND R{destReg}, R7
		return nil

	case "SPR_PRI":
		// SPR_PRI(p: u8) -> u8
		// Returns priority value shifted to bits [7:6] of attr byte
		// Priority is in bits [7:6] of byte 4 (Attributes)
		// Shift priority value left by 6 bits: p << 6
		if len(args) != 1 {
			return fmt.Errorf("SPR_PRI requires 1 argument")
		}
		// Arg is in R0, shift left by 6 bits
		cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 0)) // MOV R{destReg}, R0
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))       // MOV R7, #6
		cg.builder.AddImmediate(6)
		cg.builder.AddInstruction(rom.EncodeSHL(0, destReg, 7)) // SHL R{destReg}, R7 -> priority << 6
		return nil

	case "SPR_HFLIP":
		// SPR_HFLIP() -> u8
		// Returns 0x10 (horizontal flip bit, bit 4 of attr byte)
		cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
		cg.builder.AddImmediate(0x10)
		return nil

	case "SPR_VFLIP":
		// SPR_VFLIP() -> u8
		// Returns 0x20 (vertical flip bit, bit 5 of attr byte)
		cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
		cg.builder.AddImmediate(0x20)
		return nil

	case "SPR_SIZE_8":
		// SPR_SIZE_8() -> u8
		// Returns 0x00 (8×8 size, bit 1 of ctrl byte = 0)
		cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
		cg.builder.AddImmediate(0x00)
		return nil

	case "SPR_ENABLE":
		// SPR_ENABLE() -> u8
		// Returns 0x01 (enable bit, bit 0 of ctrl byte)
		cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
		cg.builder.AddImmediate(0x01)
		return nil

	case "SPR_SIZE_16":
		// SPR_SIZE_16() -> u8
		// Returns 0x02 (16×16 size bit, bit 1 of ctrl byte = 1)
		cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
		cg.builder.AddImmediate(0x02)
		return nil

	case "SPR_BLEND":
		// SPR_BLEND(mode: u8) -> u8
		// Returns blend mode shifted to bits [3:2] of ctrl byte
		// Blend mode is in bits [3:2] of byte 5 (Control)
		// Shift mode left by 2 bits: mode << 2
		if len(args) != 1 {
			return fmt.Errorf("SPR_BLEND requires 1 argument")
		}
		// Arg is in R0, shift left by 2 bits
		cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 0)) // MOV R{destReg}, R0
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))       // MOV R7, #2
		cg.builder.AddImmediate(2)
		cg.builder.AddInstruction(rom.EncodeSHL(0, destReg, 7)) // SHL R{destReg}, R7 -> mode << 2
		return nil

	case "SPR_ALPHA":
		// SPR_ALPHA(a: u8) -> u8
		// Returns alpha value shifted to bits [7:4] of ctrl byte
		// Alpha is in bits [7:4] of byte 5 (Control)
		// Shift alpha left by 4 bits: a << 4
		if len(args) != 1 {
			return fmt.Errorf("SPR_ALPHA requires 1 argument")
		}
		// Arg is in R0, mask to 4 bits first (alpha is 0-15), then shift left by 4
		cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 0)) // MOV R{destReg}, R0
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))       // MOV R7, #0x0F
		cg.builder.AddImmediate(0x0F)
		cg.builder.AddInstruction(rom.EncodeAND(0, destReg, 7)) // AND R{destReg}, R7 -> mask to 4 bits
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))       // MOV R7, #4
		cg.builder.AddImmediate(4)
		cg.builder.AddInstruction(rom.EncodeSHL(0, destReg, 7)) // SHL R{destReg}, R7 -> alpha << 4
		return nil

	case "oam.flush":
		// oam.flush() - no-op for now
		return nil

	case "gfx.set_palette":
		// gfx.set_palette(palette: u8, color_index: u8, color: u16)
		// Args: R0 = palette (0-15), R1 = color_index (0-15), R2 = color (RGB555, 16-bit)
		// Sets a color in CGRAM
		// CGRAM address = (palette * 16 + color_index) * 2
		// CGRAM is RGB555 format, stored as 2 bytes (low, high)

		// Calculate CGRAM color index address: (palette * 16 + color_index)
		// Note: PPU CGRAM_ADDR register is in color-index units (the PPU multiplies by 2 internally
		// when writing low/high bytes into CGRAM storage), so we must NOT multiply by 2 here.
		// palette * 16 = palette << 4
		cg.builder.AddInstruction(rom.EncodeMOV(0, 3, 0)) // MOV R3, R0 (save palette)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #4
		cg.builder.AddImmediate(4)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 3, 4)) // SHL R3, R4 -> R3 = palette << 4 = palette * 16
		cg.builder.AddInstruction(rom.EncodeADD(0, 3, 1)) // ADD R3, R1 -> R3 = palette * 16 + color_index

		// Set CGRAM_ADDR (0x8012)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0x8012
		cg.builder.AddImmediate(0x8012)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 3)) // MOV R7, R3 (CGRAM color index address)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0xFF
		cg.builder.AddImmediate(0xFF)
		cg.builder.AddInstruction(rom.EncodeAND(0, 7, 5)) // AND R7, R5 (mask to 8 bits for CGRAM_ADDR)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 6, 7)) // MOV [R6], R7 (write CGRAM_ADDR)

		// Write color to CGRAM_DATA (0x8013)
		// CGRAM_DATA requires two writes: low byte, then high byte (both to same address)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0x8013
		cg.builder.AddImmediate(0x8013)

		// Write low byte first
		cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 2)) // MOV R7, R2 (color)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0xFF
		cg.builder.AddImmediate(0xFF)
		cg.builder.AddInstruction(rom.EncodeAND(0, 7, 5)) // AND R7, R5 (mask to low byte)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 6, 7)) // MOV [R6], R7 (write low byte)

		// Write high byte (triggers CGRAM write)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 7, 2)) // MOV R7, R2 (color)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #8
		cg.builder.AddImmediate(8)
		cg.builder.AddInstruction(rom.EncodeSHR(0, 7, 5)) // SHR R7, R5 -> R7 = color >> 8 (high byte)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 6, 7)) // MOV [R6], R7 (write high byte, triggers write)
		return nil

	case "gfx.init_default_palettes":
		// gfx.init_default_palettes()
		// Initializes default palettes with basic colors
		// Palette 0: Grayscale (black to white)
		// Palette 1: Blue tones
		// Palette 2: Green tones
		// Palette 3: Red tones

		// Initialize palette 0 (grayscale)
		for i := 0; i < 16; i++ {
			// Color value: RGB555, grayscale = (i*31/15, i*31/15, i*31/15)
			// Simplified: use i*2 for each component (0-30 range)
			comp := uint16(i * 2)
			if comp > 31 {
				comp = 31
			}
			// RGB555: RRRRR GGGGG BBBBB (bits 15-11=R, 10-6=G, 5-1=B, bit 0 unused)
			color := (comp << 11) | (comp << 6) | (comp << 1)

			// Set palette 0, color i
			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012 (CGRAM_ADDR)
			cg.builder.AddImmediate(0x8012)
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #(i*2)
			cg.builder.AddImmediate(uint16(i * 2))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

			// Write color
			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013 (CGRAM_DATA)
			cg.builder.AddImmediate(0x8013)
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #color_low
			cg.builder.AddImmediate(color & 0xFF)
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (low byte)
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #color_high
			cg.builder.AddImmediate((color >> 8) & 0xFF)
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (high byte)
		}

		// Initialize palette 1 (blue tones) - simplified, just set a few colors
		// Color 0 = black, Color 15 = bright blue
		for i := 0; i < 16; i++ {
			comp := uint16(i * 2)
			if comp > 31 {
				comp = 31
			}
			// Blue: (0, 0, comp)
			color := (comp << 1) // Blue in bits 5-1

			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012
			cg.builder.AddImmediate(0x8012)
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #(16*2 + i*2)
			cg.builder.AddImmediate(uint16(16*2 + i*2))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013
			cg.builder.AddImmediate(0x8013)
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #color_low
			cg.builder.AddImmediate(color & 0xFF)
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #color_high
			cg.builder.AddImmediate((color >> 8) & 0xFF)
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
		}

		// Initialize palette 2 (green tones)
		for i := 0; i < 16; i++ {
			comp := uint16(i * 2)
			if comp > 31 {
				comp = 31
			}
			// Green: (0, comp, 0)
			color := (comp << 6) // Green in bits 10-6

			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012
			cg.builder.AddImmediate(0x8012)
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #(32*2 + i*2)
			cg.builder.AddImmediate(uint16(32*2 + i*2))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013
			cg.builder.AddImmediate(0x8013)
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #color_low
			cg.builder.AddImmediate(color & 0xFF)
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #color_high
			cg.builder.AddImmediate((color >> 8) & 0xFF)
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
		}

		// Initialize palette 3 (red tones)
		for i := 0; i < 16; i++ {
			comp := uint16(i * 2)
			if comp > 31 {
				comp = 31
			}
			// Red: (comp, 0, 0)
			color := (comp << 11) // Red in bits 15-11

			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8012
			cg.builder.AddImmediate(0x8012)
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #(48*2 + i*2)
			cg.builder.AddImmediate(uint16(48*2 + i*2))
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5

			cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8013
			cg.builder.AddImmediate(0x8013)
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #color_low
			cg.builder.AddImmediate(color & 0xFF)
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
			cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #color_high
			cg.builder.AddImmediate((color >> 8) & 0xFF)
			cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
		}

		return nil

	case "ppu.enable_display":
		// Enable display (BG0_CONTROL = 0x8008, bit 0 = enable)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8008
		cg.builder.AddImmediate(0x8008)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01
		cg.builder.AddImmediate(0x01)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
		return nil

	case "gfx.load_tiles":
		// gfx.load_tiles(asset: u16, base: u16) -> u16
		// Args: R0 = asset ID (ASSET_* constant), R1 = base tile index
		// Loads tile data from asset to VRAM starting at base * 32 bytes
		// Returns base tile index (for chaining)

		// Check if first arg is an ASSET_ constant (compile-time known)
		if len(args) > 0 {
			if ident, ok := args[0].(*IdentExpr); ok && strings.HasPrefix(ident.Name, "ASSET_") {
				assetName := strings.TrimPrefix(ident.Name, "ASSET_")
				if asset, exists := cg.assets[assetName]; exists {
					// We know the asset at compile time - inline the data writes
					return cg.generateInlineTileLoad(asset, args[1], destReg)
				}
			}
		}

		// Runtime asset loading (asset ID is a variable)
		// Calculate VRAM address: base * 32 (each 8x8 tile is 32 bytes at 4bpp)
		// Save base (R1) to R2
		cg.builder.AddInstruction(rom.EncodeMOV(0, 2, 1)) // MOV R2, R1 (save base)

		// Calculate VRAM address: base * 32 = base << 5
		cg.builder.AddInstruction(rom.EncodeMOV(0, 3, 2)) // MOV R3, R2 (copy base)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #5
		cg.builder.AddImmediate(5)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 3, 4)) // SHL R3, R4 -> R3 = base << 5 = base * 32

		// Set VRAM_ADDR_L (0x800E)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800E
		cg.builder.AddImmediate(0x800E)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 5, 3)) // MOV R5, R3 (VRAM address low)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0xFF
		cg.builder.AddImmediate(0xFF)
		cg.builder.AddInstruction(rom.EncodeAND(0, 5, 6)) // AND R5, R6 (mask to low byte)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write VRAM_ADDR_L)

		// Set VRAM_ADDR_H (0x800F)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800F
		cg.builder.AddImmediate(0x800F)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 5, 3)) // MOV R5, R3 (VRAM address)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #8
		cg.builder.AddImmediate(8)
		cg.builder.AddInstruction(rom.EncodeSHR(0, 5, 6)) // SHR R5, R6 -> R5 = VRAM address >> 8 (high byte)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write VRAM_ADDR_H)

		// TODO: Runtime asset loading would need asset data in ROM
		// For now, return base (tile index) so code can continue
		cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 2)) // MOV R{destReg}, R2 (return base)
		return nil

	case "input.read":
		// input.read() -> u16
		// Read controller 1 buttons (16-bit)
		// Latch buttons first, then read
		// Latch: write 1 to 0xA001
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0xA001
		cg.builder.AddImmediate(0xA001)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #1
		cg.builder.AddImmediate(1)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (latch)
		// Read low byte from 0xA000
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0xA000
		cg.builder.AddImmediate(0xA000)
		cg.builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (read low byte)
		// Read high byte from 0xA001
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0xA001
		cg.builder.AddImmediate(0xA001)
		cg.builder.AddInstruction(rom.EncodeMOV(2, 6, 4)) // MOV R6, [R4] (read high byte)
		// Combine: R5 (low) | (R6 << 8) (high)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #8
		cg.builder.AddImmediate(8)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 6, 7)) // SHL R6, R7 -> R6 = high << 8
		cg.builder.AddInstruction(rom.EncodeOR(0, 5, 6))  // OR R5, R6 -> R5 = low | (high << 8)
		// Release latch: write 0 to 0xA001
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0xA001
		cg.builder.AddImmediate(0xA001)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0
		cg.builder.AddImmediate(0)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 6)) // MOV [R4], R6 (release latch)
		// Return value in destReg
		cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 5)) // MOV R{destReg}, R5
		return nil

	// APU Functions
	case "apu.enable":
		// apu.enable() - Enable APU master volume
		// Write 0xFF to MASTER_VOLUME (0x9020)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x9020
		cg.builder.AddImmediate(0x9020)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0xFF
		cg.builder.AddImmediate(0xFF)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write master volume)
		return nil

	case "apu.set_channel_wave":
		// apu.set_channel_wave(ch: u8, wave: u8)
		// Args: R0 = channel (0-3), R1 = waveform (0-3)
		// Write to CONTROL register (offset +3) with bits [1:2] = waveform
		// Channel base: CH0=0x9000, CH1=0x9008, CH2=0x9010, CH3=0x9018
		// CONTROL = channel_base + 3

		// Calculate channel base address: 0x9000 + (ch * 8)
		// ch * 8 = ch << 3
		cg.builder.AddInstruction(rom.EncodeMOV(0, 4, 0)) // MOV R4, R0 (save channel)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #3
		cg.builder.AddImmediate(3)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 4, 5)) // SHL R4, R5 -> R4 = ch << 3 = ch * 8
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x9000
		cg.builder.AddImmediate(0x9000)
		cg.builder.AddInstruction(rom.EncodeADD(0, 4, 5)) // ADD R4, R5 -> R4 = 0x9000 + (ch * 8)

		// Add offset 3 for CONTROL register
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #3
		cg.builder.AddImmediate(3)
		cg.builder.AddInstruction(rom.EncodeADD(0, 4, 5)) // ADD R4, R5 -> R4 = channel_base + 3

		// Prepare waveform value: shift to bits [1:2]
		// Waveform is in R1, need to shift left by 1 bit
		cg.builder.AddInstruction(rom.EncodeMOV(0, 5, 1)) // MOV R5, R1 (waveform)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #1
		cg.builder.AddImmediate(1)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 5, 6)) // SHL R5, R6 -> R5 = wave << 1

		// Read current CONTROL value, OR with waveform bits, write back
		// For simplicity, just write waveform bits (assumes enable bit will be set separately)
		// In practice, we'd read, mask, OR, write - but for now just write waveform
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write CONTROL)
		return nil

	case "apu.set_channel_freq":
		// apu.set_channel_freq(ch: u8, freq: u16)
		// Args: R0 = channel (0-3), R1 = frequency (16-bit)
		// Write low byte to FREQ_LOW (offset +0), then high byte to FREQ_HIGH (offset +1)
		// Writing high byte triggers phase reset

		// Calculate channel base address: 0x9000 + (ch * 8)
		// ch * 8 = ch << 3
		cg.builder.AddInstruction(rom.EncodeMOV(0, 4, 0)) // MOV R4, R0 (save channel)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #3
		cg.builder.AddImmediate(3)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 4, 5)) // SHL R4, R5 -> R4 = ch << 3 = ch * 8
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x9000
		cg.builder.AddImmediate(0x9000)
		cg.builder.AddInstruction(rom.EncodeADD(0, 4, 5)) // ADD R4, R5 -> R4 = 0x9000 + (ch * 8)

		// Save frequency value
		cg.builder.AddInstruction(rom.EncodeMOV(0, 5, 1)) // MOV R5, R1 (frequency)

		// Write low byte to FREQ_LOW (offset +0)
		cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 5)) // MOV R6, R5 (copy freq)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0xFF
		cg.builder.AddImmediate(0xFF)
		cg.builder.AddInstruction(rom.EncodeAND(0, 6, 7)) // AND R6, R7 -> R6 = freq & 0xFF (low byte)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 6)) // MOV [R4], R6 (write FREQ_LOW)

		// Write high byte to FREQ_HIGH (offset +1)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #1
		cg.builder.AddImmediate(1)
		cg.builder.AddInstruction(rom.EncodeADD(0, 4, 6)) // ADD R4, R6 -> R4 = channel_base + 1
		cg.builder.AddInstruction(rom.EncodeMOV(0, 6, 5)) // MOV R6, R5 (copy freq)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #8
		cg.builder.AddImmediate(8)
		cg.builder.AddInstruction(rom.EncodeSHR(0, 6, 7)) // SHR R6, R7 -> R6 = freq >> 8 (high byte)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 6)) // MOV [R4], R6 (write FREQ_HIGH, triggers phase reset)
		return nil

	case "apu.set_channel_volume":
		// apu.set_channel_volume(ch: u8, vol: u8)
		// Args: R0 = channel (0-3), R1 = volume (0-255)
		// Write to VOLUME register (offset +2)

		// Calculate channel base address: 0x9000 + (ch * 8)
		// ch * 8 = ch << 3
		cg.builder.AddInstruction(rom.EncodeMOV(0, 4, 0)) // MOV R4, R0 (save channel)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #3
		cg.builder.AddImmediate(3)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 4, 5)) // SHL R4, R5 -> R4 = ch << 3 = ch * 8
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x9000
		cg.builder.AddImmediate(0x9000)
		cg.builder.AddInstruction(rom.EncodeADD(0, 4, 5)) // ADD R4, R5 -> R4 = 0x9000 + (ch * 8)

		// Add offset 2 for VOLUME register
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #2
		cg.builder.AddImmediate(2)
		cg.builder.AddInstruction(rom.EncodeADD(0, 4, 5)) // ADD R4, R5 -> R4 = channel_base + 2

		// Write volume (R1) to VOLUME register
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 1)) // MOV [R4], R1 (write VOLUME)
		return nil

	case "apu.note_on":
		// apu.note_on(ch: u8)
		// Args: R0 = channel (0-3)
		// Set CONTROL register (offset +3) bit 0 to 1 (enable)

		// Calculate channel base address: 0x9000 + (ch * 8)
		// ch * 8 = ch << 3
		cg.builder.AddInstruction(rom.EncodeMOV(0, 4, 0)) // MOV R4, R0 (save channel)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #3
		cg.builder.AddImmediate(3)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 4, 5)) // SHL R4, R5 -> R4 = ch << 3 = ch * 8
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x9000
		cg.builder.AddImmediate(0x9000)
		cg.builder.AddInstruction(rom.EncodeADD(0, 4, 5)) // ADD R4, R5 -> R4 = 0x9000 + (ch * 8)

		// Add offset 3 for CONTROL register
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #3
		cg.builder.AddImmediate(3)
		cg.builder.AddInstruction(rom.EncodeADD(0, 4, 5)) // ADD R4, R5 -> R4 = channel_base + 3

		// Read current CONTROL value, OR with 0x01 (enable bit)
		cg.builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (read current CONTROL)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0x01
		cg.builder.AddImmediate(0x01)
		cg.builder.AddInstruction(rom.EncodeOR(0, 5, 6))  // OR R5, R6 -> R5 = CONTROL | 0x01
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write CONTROL with enable bit)
		return nil

	case "apu.note_off":
		// apu.note_off(ch: u8)
		// Args: R0 = channel (0-3)
		// Clear CONTROL register (offset +3) bit 0 (disable)

		// Calculate channel base address: 0x9000 + (ch * 8)
		// ch * 8 = ch << 3
		cg.builder.AddInstruction(rom.EncodeMOV(0, 4, 0)) // MOV R4, R0 (save channel)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #3
		cg.builder.AddImmediate(3)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 4, 5)) // SHL R4, R5 -> R4 = ch << 3 = ch * 8
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x9000
		cg.builder.AddImmediate(0x9000)
		cg.builder.AddInstruction(rom.EncodeADD(0, 4, 5)) // ADD R4, R5 -> R4 = 0x9000 + (ch * 8)

		// Add offset 3 for CONTROL register
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #3
		cg.builder.AddImmediate(3)
		cg.builder.AddInstruction(rom.EncodeADD(0, 4, 5)) // ADD R4, R5 -> R4 = channel_base + 3

		// Read current CONTROL value, AND with 0xFE (clear enable bit)
		cg.builder.AddInstruction(rom.EncodeMOV(2, 5, 4)) // MOV R5, [R4] (read current CONTROL)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0xFE
		cg.builder.AddImmediate(0xFE)
		cg.builder.AddInstruction(rom.EncodeAND(0, 5, 6)) // AND R5, R6 -> R5 = CONTROL & 0xFE (clear bit 0)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write CONTROL without enable bit)
		return nil

	default:
		return fmt.Errorf("unknown builtin: %s", name)
	}
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
	// R1 now has base tile index

	// Calculate VRAM address based on tile size
	// For tiles8: base * 32 = base << 5 (32 bytes per tile)
	// For tiles16: base * 128 = base << 7 (128 bytes per tile)
	cg.builder.AddInstruction(rom.EncodeMOV(0, 2, 1)) // MOV R2, R1 (save base)
	cg.builder.AddInstruction(rom.EncodeMOV(0, 3, 2)) // MOV R3, R2 (copy base)

	if asset.Type == "tiles16" {
		// 16x16 tile: base * 128 = base << 7
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #7
		cg.builder.AddImmediate(7)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 3, 4)) // SHL R3, R4 -> R3 = base << 7 = base * 128
	} else {
		// 8x8 tile: base * 32 = base << 5
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #5
		cg.builder.AddImmediate(5)
		cg.builder.AddInstruction(rom.EncodeSHL(0, 3, 4)) // SHL R3, R4 -> R3 = base << 5 = base * 32
	}

	// Set VRAM_ADDR_L (0x800E)
	cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800E
	cg.builder.AddImmediate(0x800E)
	cg.builder.AddInstruction(rom.EncodeMOV(0, 5, 3)) // MOV R5, R3 (VRAM address low)
	cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0xFF
	cg.builder.AddImmediate(0xFF)
	cg.builder.AddInstruction(rom.EncodeAND(0, 5, 6)) // AND R5, R6 (mask to low byte)
	cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write VRAM_ADDR_L)

	// Set VRAM_ADDR_H (0x800F)
	cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x800F
	cg.builder.AddImmediate(0x800F)
	cg.builder.AddInstruction(rom.EncodeMOV(0, 5, 3)) // MOV R5, R3 (VRAM address)
	cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #8
	cg.builder.AddImmediate(8)
	cg.builder.AddInstruction(rom.EncodeSHR(0, 5, 6)) // SHR R5, R6 -> R5 = VRAM address >> 8 (high byte)
	cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5 (write VRAM_ADDR_H)

	dataBytes, err := cg.inlineTileAssetBytes(asset)
	if err != nil {
		return err
	}
	vramDataAddr := uint16(0x8010)
	cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8010
	cg.builder.AddImmediate(vramDataAddr)

	tileSize := 32 // 8x8 tile at 4bpp = 32 bytes
	if asset.Type == "tiles16" {
		tileSize = 128 // 16x16 tile at 4bpp = 128 bytes
	}
	for i, value := range dataBytes {
		if i >= tileSize {
			break
		}
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #value
		cg.builder.AddImmediate(uint16(value))
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
	}

	// Return base tile index
	cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 2)) // MOV R{destReg}, R2 (return base)
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

func (cg *CodeGenerator) generateMember(expr *MemberExpr, destReg uint8) error {
	// Generate object
	if err := cg.generateExpr(expr.Object, 0); err != nil {
		return err
	}
	// Member access would need struct layout knowledge
	return fmt.Errorf("member access not fully implemented: %s", expr.Member)
}

func (cg *CodeGenerator) generateUnary(expr *UnaryExpr, destReg uint8) error {
	if err := cg.generateExpr(expr.Operand, destReg); err != nil {
		return err
	}
	switch expr.Op {
	case TOKEN_MINUS:
		// Negate: 0 - value
		cg.builder.AddInstruction(rom.EncodeMOV(1, 1, 0)) // MOV R1, #0
		cg.builder.AddImmediate(0)
		cg.builder.AddInstruction(rom.EncodeSUB(0, 1, destReg)) // SUB R1, R{destReg}
		cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 1)) // MOV R{destReg}, R1
	case TOKEN_NOT:
		// Logical NOT: compare with 0, set to 1 if zero, 0 otherwise
		// Compare operand with 0
		cg.builder.AddInstruction(rom.EncodeMOV(1, 1, 0)) // MOV R1, #0
		cg.builder.AddImmediate(0)
		cg.builder.AddInstruction(rom.EncodeCMP(0, destReg, 1)) // CMP R{destReg}, R1
		// Set to 1 if equal (zero), 0 if not equal
		cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0)) // MOV R{destReg}, #0
		cg.builder.AddImmediate(0)
		skipLabel := cg.newLabel()
		cg.builder.AddInstruction(rom.EncodeBNE()) // BNE skip
		skipPos := cg.builder.GetCodeLength()
		cg.builder.AddImmediate(0)
		cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0)) // MOV R{destReg}, #1
		cg.builder.AddImmediate(1)
		cg.patchLabel(skipLabel, skipPos)
		return nil
	case TOKEN_TILDE:
		// Bitwise NOT: 0xFFFF - value
		cg.builder.AddInstruction(rom.EncodeMOV(1, 1, 0)) // MOV R1, #0xFFFF
		cg.builder.AddImmediate(0xFFFF)
		cg.builder.AddInstruction(rom.EncodeSUB(0, 1, destReg)) // SUB R1, R{destReg}
		cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 1)) // MOV R{destReg}, R1
		return nil
	case TOKEN_AMPERSAND:
		// Address-of operator &x
		// For now, simplified - just return 0 as placeholder
		// In full implementation, would return actual address
		// The operand is already evaluated, so we just use it
		return nil
	default:
		return fmt.Errorf("unary operator not yet implemented: %v", expr.Op)
	}
	return nil
}

func (cg *CodeGenerator) generateStore(target Expr, srcReg uint8) error {
	// Store value in srcReg to target
	// This is simplified
	return fmt.Errorf("store not fully implemented")
}

func (cg *CodeGenerator) newLabel() int {
	label := cg.labelCounter
	cg.labelCounter++
	return label
}

func (cg *CodeGenerator) patchLabel(label, offsetPos int) {
	_ = label // Labels are currently patched immediately at their definition point.
	// offsetPos is the word index where the branch/jump immediate placeholder was emitted.
	// The CPU PC-relative branch offset is calculated from the address *after* the immediate.
	currentPC := uint16(offsetPos * 2)
	targetPC := uint16(cg.builder.GetCodeLength() * 2)
	offset := rom.CalculateBranchOffset(currentPC, targetPC)
	cg.builder.SetImmediateAt(offsetPos, uint16(offset))
}
