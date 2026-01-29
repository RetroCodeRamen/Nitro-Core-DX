package corelx

import (
	"fmt"
	"strings"

	"nitro-core-dx/internal/rom"
)

// CodeGenerator generates Nitro Core DX machine code from AST
type CodeGenerator struct {
	program      *Program
	builder      *rom.ROMBuilder
	symbols      map[string]*Symbol
	regAlloc     *RegisterAllocator
	labelCounter int
	assets       map[string]*AssetDecl
	assetOffsets map[string]uint16
	
	// Variable storage tracking
	variables    map[string]*VariableInfo
	varCounter   int
	stackOffset  uint16 // Current stack offset for spilled variables
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
	registers [8]bool // R0-R7 usage
	spill     []string // Spilled variables
}

// NewCodeGenerator creates a new code generator
func NewCodeGenerator(program *Program, builder *rom.ROMBuilder) *CodeGenerator {
	return &CodeGenerator{
		program:      program,
		builder:      builder,
		symbols:      make(map[string]*Symbol),
		regAlloc:     &RegisterAllocator{},
		labelCounter: 0,
		assets:       make(map[string]*AssetDecl),
		assetOffsets: make(map[string]uint16),
		variables:    make(map[string]*VariableInfo),
		varCounter:   0,
		stackOffset:  0x1FFF, // Start at top of stack (grows downward)
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
	for _, fn := range cg.program.Functions {
		if err := cg.generateFunction(fn); err != nil {
			return err
		}
	}

	return nil
}

func (cg *CodeGenerator) generateFunction(fn *FunctionDecl) error {
	// Reset variable tracking for each function
	cg.variables = make(map[string]*VariableInfo)
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
				// For simplicity, allocate a register to hold the address
				var reg uint8 = 2
				for reg < 8 && cg.regAlloc.registers[reg] {
					reg++
				}
				if reg < 8 {
					// Store address in register
					cg.builder.AddInstruction(rom.EncodeMOV(0, reg, 0)) // MOV R{reg}, R0
					cg.regAlloc.registers[reg] = true
					cg.variables[stmt.Name] = &VariableInfo{
						Name:     stmt.Name,
						Location: VarLocationRegister,
						RegIndex: reg,
					}
				} else {
					// Spill to stack - store address on stack
					cg.stackOffset -= 2
					stackAddr := cg.stackOffset
					cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #stackAddr
					cg.builder.AddImmediate(stackAddr)
					cg.builder.AddInstruction(rom.EncodeMOV(3, 7, 0)) // MOV [R7], R0
					cg.variables[stmt.Name] = &VariableInfo{
						Name:     stmt.Name,
						Location: VarLocationStack,
						StackAddr: stackAddr,
					}
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
	// Allocate storage for variable
	// Try to use a register first (R2-R7, R0-R1 are used for temporaries)
	var reg uint8 = 2
	for reg < 8 && cg.regAlloc.registers[reg] {
		reg++
	}
	
	if reg < 8 {
		// Store in register
		cg.builder.AddInstruction(rom.EncodeMOV(0, reg, 0)) // MOV R{reg}, R0
		cg.regAlloc.registers[reg] = true
		cg.variables[stmt.Name] = &VariableInfo{
			Name:     stmt.Name,
			Location: VarLocationRegister,
			RegIndex: reg,
		}
	} else {
		// Spill to stack
		cg.stackOffset -= 2 // Allocate 2 bytes (16-bit value)
		stackAddr := cg.stackOffset
		// Store to stack: MOV [SP+offset], R0
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #stackAddr
		cg.builder.AddImmediate(stackAddr)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 7, 0)) // MOV [R7], R0
		cg.variables[stmt.Name] = &VariableInfo{
			Name:     stmt.Name,
			Location: VarLocationStack,
			StackAddr: stackAddr,
		}
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
						cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #offset
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
			cg.builder.AddInstruction(rom.EncodeADD(0, destReg, 2)) // ADD R{destReg}, R2
		case TOKEN_MINUS:
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
			// Compare and set result: 1 if equal, 0 if not
			cg.builder.AddInstruction(rom.EncodeCMP(0, 1, 2)) // CMP R1, R2
			// Set R0 to 1 if equal, 0 if not
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0)) // MOV R{destReg}, #0
			cg.builder.AddImmediate(0)
			// Branch past setting to 1 if not equal
			skipLabel := cg.newLabel()
			cg.builder.AddInstruction(rom.EncodeBNE()) // BNE skip
			skipPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0) // Placeholder
			// Set to 1 (equal)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(1)
			// Skip label
			cg.patchLabel(skipLabel, skipPos)
			return nil
		case TOKEN_BANG_EQUAL:
			// Compare and set result: 1 if not equal, 0 if equal
			cg.builder.AddInstruction(rom.EncodeCMP(0, 1, 2))
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(0)
			skipLabel := cg.newLabel()
			cg.builder.AddInstruction(rom.EncodeBEQ()) // BEQ skip
			skipPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(1)
			cg.patchLabel(skipLabel, skipPos)
			return nil
		case TOKEN_LESS:
			cg.builder.AddInstruction(rom.EncodeCMP(0, 1, 2))
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(0)
			skipLabel := cg.newLabel()
			cg.builder.AddInstruction(rom.EncodeBGE()) // BGE skip (if >=, not <)
			skipPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(1)
			cg.patchLabel(skipLabel, skipPos)
			return nil
		case TOKEN_GREATER:
			cg.builder.AddInstruction(rom.EncodeCMP(0, 1, 2))
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(0)
			skipLabel := cg.newLabel()
			cg.builder.AddInstruction(rom.EncodeBLE()) // BLE skip
			skipPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(1)
			cg.patchLabel(skipLabel, skipPos)
			return nil
		case TOKEN_LESS_EQUAL:
			cg.builder.AddInstruction(rom.EncodeCMP(0, 1, 2))
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(0)
			skipLabel := cg.newLabel()
			cg.builder.AddInstruction(rom.EncodeBGT()) // BGT skip
			skipPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(1)
			cg.patchLabel(skipLabel, skipPos)
			return nil
		case TOKEN_GREATER_EQUAL:
			cg.builder.AddInstruction(rom.EncodeCMP(0, 1, 2))
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(0)
			skipLabel := cg.newLabel()
			cg.builder.AddInstruction(rom.EncodeBLT()) // BLT skip
			skipPos := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
			cg.builder.AddImmediate(1)
			cg.patchLabel(skipLabel, skipPos)
			return nil
		case TOKEN_AND:
			// Logical AND: both must be non-zero
			// R1 already has left, R2 has right
			// Set R0 to 1 if both non-zero, else 0
			cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0)) // MOV R7, #0
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeCMP(0, 1, 7)) // CMP R1, R7
			cg.builder.AddInstruction(rom.EncodeBEQ()) // BEQ false
			falseLabel1 := cg.newLabel()
			falsePos1 := cg.builder.GetCodeLength()
			cg.builder.AddImmediate(0)
			cg.builder.AddInstruction(rom.EncodeCMP(0, 2, 7)) // CMP R2, R7
			cg.builder.AddInstruction(rom.EncodeBEQ()) // BEQ false
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
			// Bitwise OR
			cg.builder.AddInstruction(rom.EncodeOR(0, destReg, 2))
			return nil
		case TOKEN_AMPERSAND:
			// Bitwise AND
			cg.builder.AddInstruction(rom.EncodeAND(0, destReg, 2))
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
						cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #offset
						cg.builder.AddImmediate(offset)
						cg.builder.AddInstruction(rom.EncodeADD(0, 7, 6)) // ADD R7, R6 (member address)
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
						cg.builder.AddInstruction(rom.EncodeADD(0, 6, 7)) // ADD R6, R7 (member address)
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
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))  // MOV R7, #0x01
		cg.builder.AddImmediate(0x01)
		cg.builder.AddInstruction(rom.EncodeAND(0, 5, 7)) // AND R5, R7 (mask to bit 0)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 7, 0))  // MOV R7, #0
		cg.builder.AddImmediate(0)
		cg.builder.AddInstruction(rom.EncodeCMP(0, 5, 7))  // CMP R5, R7 (compare with 0)
		cg.builder.AddInstruction(rom.EncodeBEQ())         // BEQ waitPos (if equal to 0, keep waiting)
		currentPC := uint16(cg.builder.GetCodeLength() * 2)
		offset := rom.CalculateBranchOffset(currentPC, uint16(waitPos*2))
		cg.builder.AddImmediate(uint16(offset))
		return nil

	case "frame_counter":
		// Read frame counter (would need a register for this)
		cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
		cg.builder.AddImmediate(0) // Placeholder
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
		// R3 = sprite pointer
		
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
		// Returns priority value shifted to correct bit position
		// Arg is in R0
		// Priority is in upper bits of attr, for now just return arg
		cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 0)) // MOV R{destReg}, R0
		return nil

	case "SPR_ENABLE":
		// SPR_ENABLE() -> u8
		// Returns 0x01 (enable bit)
		cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
		cg.builder.AddImmediate(0x01)
		return nil

	case "SPR_SIZE_16":
		// SPR_SIZE_16() -> u8
		// Returns 0x02 (16x16 size bit)
		cg.builder.AddInstruction(rom.EncodeMOV(1, destReg, 0))
		cg.builder.AddImmediate(0x02)
		return nil

	case "oam.flush":
		// oam.flush() - no-op for now
		return nil

	case "ppu.enable_display":
		// Enable display (PPU_CONTROL = 0x8000, bit 0 = enable)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0x8000
		cg.builder.AddImmediate(0x8000)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 5, 0)) // MOV R5, #0x01
		cg.builder.AddImmediate(0x01)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 5)) // MOV [R4], R5
		return nil

	case "gfx.load_tiles":
		// gfx.load_tiles(asset: u16, base: u16) -> u16
		// For now, just return the base (first arg is in R0, second in R1)
		// In a full implementation, this would load tiles from ROM
		cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 1)) // MOV R{destReg}, R1 (base)
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
		cg.builder.AddInstruction(rom.EncodeOR(0, 5, 6)) // OR R5, R6 -> R5 = low | (high << 8)
		// Release latch: write 0 to 0xA001
		cg.builder.AddInstruction(rom.EncodeMOV(1, 4, 0)) // MOV R4, #0xA001
		cg.builder.AddImmediate(0xA001)
		cg.builder.AddInstruction(rom.EncodeMOV(1, 6, 0)) // MOV R6, #0
		cg.builder.AddImmediate(0)
		cg.builder.AddInstruction(rom.EncodeMOV(3, 4, 6)) // MOV [R4], R6 (release latch)
		// Return value in destReg
		cg.builder.AddInstruction(rom.EncodeMOV(0, destReg, 5)) // MOV R{destReg}, R5
		return nil

	default:
		return fmt.Errorf("unknown builtin: %s", name)
	}
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
	// In a full implementation, we'd track labels and patch them
	// For now, this is a placeholder
}
