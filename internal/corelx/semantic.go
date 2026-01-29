package corelx

import (
	"fmt"
)

// SemanticAnalyzer performs semantic analysis
type SemanticAnalyzer struct {
	program     *Program
	symbols     map[string]*Symbol
	errors      []error
	currentFunc *FunctionDecl
}

// Symbol represents a symbol in the symbol table
type Symbol struct {
	Name     string
	Type     TypeExpr
	IsFunc   bool
	IsBuiltin bool
	Position Position
}

// Analyze performs semantic analysis
func Analyze(program *Program) error {
	analyzer := &SemanticAnalyzer{
		program: program,
		symbols: make(map[string]*Symbol),
		errors:  make([]error, 0),
	}

	// Register built-in types
	analyzer.registerBuiltinTypes()

	// Register built-in functions
	analyzer.registerBuiltinFunctions()

	// Analyze types
	for _, typeDecl := range program.Types {
		analyzer.analyzeType(typeDecl)
	}

	// Analyze assets
	for _, asset := range program.Assets {
		analyzer.analyzeAsset(asset)
	}

	// Analyze functions
	for _, fn := range program.Functions {
		analyzer.analyzeFunction(fn)
	}

	// Check for Start() function
	if _, ok := analyzer.symbols["Start"]; !ok {
		analyzer.errors = append(analyzer.errors, fmt.Errorf("missing required function: Start()"))
	}

	if len(analyzer.errors) > 0 {
		return fmt.Errorf("semantic errors: %v", analyzer.errors)
	}

	return nil
}

func (a *SemanticAnalyzer) registerBuiltinTypes() {
	builtins := []string{
		"i8", "u8", "i16", "u16", "i32", "u32",
		"bool", "fx8_8", "fx16_16",
		"Sprite",
	}
	for _, name := range builtins {
		a.symbols[name] = &Symbol{
			Name:     name,
			Type:     &NamedType{Name: name},
			IsBuiltin: true,
		}
	}
}

func (a *SemanticAnalyzer) registerBuiltinFunctions() {
	// Built-in functions will be handled by code generator
	// This is just for semantic checking
	builtins := []string{
		"Start", "__Boot", // Entry points
		"wait_vblank", "frame_counter",
		"sprite.set_pos", "oam.write", "oam.flush",
		"apu.enable", "apu.set_channel_wave", "apu.set_channel_freq",
		"apu.set_channel_volume", "apu.note_on", "apu.note_off",
		"ppu.enable_display", "gfx.load_tiles",
		"SPR_PAL", "SPR_HFLIP", "SPR_VFLIP", "SPR_PRI",
		"SPR_ENABLE", "SPR_SIZE_8", "SPR_SIZE_16",
		"SPR_BLEND", "SPR_ALPHA",
	}
	for _, name := range builtins {
		a.symbols[name] = &Symbol{
			Name:     name,
			IsFunc:   true,
			IsBuiltin: true,
		}
	}
}

func (a *SemanticAnalyzer) analyzeType(typeDecl *TypeDecl) {
	if _, exists := a.symbols[typeDecl.Name]; exists && !a.symbols[typeDecl.Name].IsBuiltin {
		a.errors = append(a.errors, fmt.Errorf("type %s already defined", typeDecl.Name))
		return
	}

	// Convert TypeSpec to TypeExpr for storage
	// For now, store as NamedType - in a full implementation we'd track struct types
	typeExpr := &NamedType{Name: typeDecl.Name}

	a.symbols[typeDecl.Name] = &Symbol{
		Name:     typeDecl.Name,
		Type:     typeExpr,
		Position: typeDecl.Position,
	}
}

func (a *SemanticAnalyzer) analyzeAsset(asset *AssetDecl) {
	// Assets are registered as constants
	constName := "ASSET_" + asset.Name
	a.symbols[constName] = &Symbol{
		Name:     constName,
		Type:     &NamedType{Name: "u16"},
		IsBuiltin: false,
		Position: asset.Position,
	}
}

func (a *SemanticAnalyzer) analyzeFunction(fn *FunctionDecl) {
	if fn.Name == "Start" || fn.Name == "__Boot" {
		// Entry points are special
		if len(fn.Params) > 0 {
			a.errors = append(a.errors, fmt.Errorf("function %s() must have no parameters", fn.Name))
		}
	}

	oldFunc := a.currentFunc
	a.currentFunc = fn
	defer func() { a.currentFunc = oldFunc }()

	// Analyze function body
	for _, stmt := range fn.Body {
		a.analyzeStmt(stmt)
	}
}

func (a *SemanticAnalyzer) analyzeStmt(stmt Stmt) {
	switch s := stmt.(type) {
	case *VarDeclStmt:
		// Variable declaration
		if _, exists := a.symbols[s.Name]; exists {
			a.errors = append(a.errors, fmt.Errorf("variable %s already defined", s.Name))
		} else {
			var varType TypeExpr
			if s.Type != nil {
				varType = s.Type
			} else {
				// Infer type from value
				varType = a.inferType(s.Value)
			}
			a.symbols[s.Name] = &Symbol{
				Name:     s.Name,
				Type:     varType,
				Position: s.Position,
			}
		}
		a.analyzeExpr(s.Value)

	case *AssignStmt:
		a.analyzeExpr(s.Target)
		a.analyzeExpr(s.Value)

	case *IfStmt:
		a.analyzeExpr(s.Condition)
		for _, stmt := range s.Then {
			a.analyzeStmt(stmt)
		}
		for _, clause := range s.ElseIf {
			a.analyzeExpr(clause.Condition)
			for _, stmt := range clause.Body {
				a.analyzeStmt(stmt)
			}
		}
		for _, stmt := range s.Else {
			a.analyzeStmt(stmt)
		}

	case *WhileStmt:
		a.analyzeExpr(s.Condition)
		for _, stmt := range s.Body {
			a.analyzeStmt(stmt)
		}

	case *ForStmt:
		if s.Init != nil {
			a.analyzeStmt(s.Init)
		}
		a.analyzeExpr(s.Condition)
		if s.Post != nil {
			a.analyzeStmt(s.Post)
		}
		for _, stmt := range s.Body {
			a.analyzeStmt(stmt)
		}

	case *ReturnStmt:
		if s.Value != nil {
			a.analyzeExpr(s.Value)
		}

	case *ExprStmt:
		a.analyzeExpr(s.Expr)
	}
}

func (a *SemanticAnalyzer) analyzeExpr(expr Expr) {
	switch e := expr.(type) {
	case *BinaryExpr:
		a.analyzeExpr(e.Left)
		a.analyzeExpr(e.Right)

	case *UnaryExpr:
		a.analyzeExpr(e.Operand)

	case *CallExpr:
		a.analyzeExpr(e.Func)
		for _, arg := range e.Args {
			a.analyzeExpr(arg)
		}

	case *MemberExpr:
		a.analyzeExpr(e.Object)
		// Member expressions like ppu.enable_display() are valid
		// The object (ppu, sprite, oam, etc.) doesn't need to be a defined variable
		// It's a namespace for built-in functions

	case *IndexExpr:
		a.analyzeExpr(e.Array)
		a.analyzeExpr(e.Index)

	case *IdentExpr:
		// Check if it's a built-in namespace (ppu, sprite, oam, apu, gfx)
		builtinNamespaces := map[string]bool{
			"ppu": true, "sprite": true, "oam": true, "apu": true, "gfx": true, "input": true,
		}
		if builtinNamespaces[e.Name] {
			// Built-in namespace, valid
			return
		}
		if _, exists := a.symbols[e.Name]; !exists {
			a.errors = append(a.errors, fmt.Errorf("undefined identifier: %s", e.Name))
		}

	case *NumberExpr, *StringExpr, *BoolExpr:
		// Literals are fine
	}
}

func (a *SemanticAnalyzer) inferType(expr Expr) TypeExpr {
	switch e := expr.(type) {
	case *NumberExpr:
		// Default to i16 for integers
		return &NamedType{Name: "i16"}
	case *BoolExpr:
		return &NamedType{Name: "bool"}
	case *StringExpr:
		// Strings are not directly supported, but we can use them for asset names
		return &NamedType{Name: "u16"}
	case *CallExpr:
		// Try to infer from function return type
		if ident, ok := e.Func.(*IdentExpr); ok {
			if sym, exists := a.symbols[ident.Name]; exists && sym.IsFunc {
				// For now, default to u16 for function calls
				return &NamedType{Name: "u16"}
			}
		}
		return &NamedType{Name: "u16"}
	default:
		return &NamedType{Name: "u16"}
	}
}
