package corelx

import (
	"fmt"
)

// SemanticAnalyzer performs semantic analysis
type SemanticAnalyzer struct {
	program     *Program
	symbols     map[string]*Symbol
	diagnostics []Diagnostic
	currentFunc *FunctionDecl
}

// Symbol represents a symbol in the symbol table
type Symbol struct {
	Name      string
	Type      TypeExpr
	IsFunc    bool
	IsBuiltin bool
	Position  Position
}

// Analyze performs semantic analysis
func Analyze(program *Program) error {
	diags := AnalyzeWithDiagnostics(program)
	if !HasErrors(diags) {
		return nil
	}
	return &DiagnosticsError{Diagnostics: diags}
}

// AnalyzeWithDiagnostics performs semantic analysis and returns structured diagnostics.
func AnalyzeWithDiagnostics(program *Program) []Diagnostic {
	analyzer := &SemanticAnalyzer{
		program:     program,
		symbols:     make(map[string]*Symbol),
		diagnostics: make([]Diagnostic, 0),
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

	// Register/analyze function declarations (name collisions first)
	analyzer.registerFunctionDecls()

	// Analyze functions
	for _, fn := range program.Functions {
		analyzer.analyzeFunction(fn)
	}

	// Check for Start() function
	hasStart := false
	for _, fn := range program.Functions {
		if fn.Name == "Start" {
			hasStart = true
			break
		}
	}
	if !hasStart {
		analyzer.addDiagnostic(Position{}, CategoryValidationError, "E_MISSING_ENTRYPOINT", "missing required function: Start()", "")
	}

	return analyzer.diagnostics
}

func (a *SemanticAnalyzer) registerBuiltinTypes() {
	builtins := []string{
		"i8", "u8", "i16", "u16", "i32", "u32",
		"bool", "fx8_8", "fx16_16",
		"Sprite",
	}
	for _, name := range builtins {
		a.symbols[name] = &Symbol{
			Name:      name,
			Type:      &NamedType{Name: name},
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
		"sprite.set_pos", "oam.write", "oam.write_sprite_data", "oam.clear_sprite", "oam.flush",
		"apu.enable", "apu.set_channel_wave", "apu.set_channel_freq",
		"apu.set_channel_volume", "apu.note_on", "apu.note_off",
		"ppu.enable_display", "gfx.load_tiles", "gfx.set_palette", "gfx.set_palette_color", "gfx.init_default_palettes",
		"input.read",
		"SPR_PAL", "SPR_HFLIP", "SPR_VFLIP", "SPR_PRI",
		"SPR_ENABLE", "SPR_SIZE_8", "SPR_SIZE_16",
		"SPR_BLEND", "SPR_ALPHA",
	}
	for _, name := range builtins {
		a.symbols[name] = &Symbol{
			Name:      name,
			IsFunc:    true,
			IsBuiltin: true,
		}
	}
}

func (a *SemanticAnalyzer) analyzeType(typeDecl *TypeDecl) {
	if _, exists := a.symbols[typeDecl.Name]; exists && !a.symbols[typeDecl.Name].IsBuiltin {
		existing := a.symbols[typeDecl.Name]
		a.addDuplicateDiagnostic(typeDecl.Position, CategorySymbolError, "E_TYPE_DUPLICATE", fmt.Sprintf("type %s already defined", typeDecl.Name), "", existing.Position, "previous type declaration")
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
	if _, exists := a.symbols[constName]; exists {
		existing := a.symbols[constName]
		a.addDuplicateDiagnostic(asset.Position, CategorySymbolError, "E_ASSET_DUPLICATE", fmt.Sprintf("asset %s already defined", asset.Name), "", existing.Position, "previous asset declaration")
		return
	}
	a.symbols[constName] = &Symbol{
		Name:      constName,
		Type:      &NamedType{Name: "u16"},
		IsBuiltin: false,
		Position:  asset.Position,
	}
}

func (a *SemanticAnalyzer) registerFunctionDecls() {
	seen := make(map[string]Position)
	for _, fn := range a.program.Functions {
		if prev, exists := seen[fn.Name]; exists {
			a.addDuplicateDiagnostic(fn.Position, CategorySymbolError, "E_FUNC_DUPLICATE", fmt.Sprintf("function %s already defined", fn.Name), "", prev, "previous function declaration")
			continue
		}
		seen[fn.Name] = fn.Position

		if existing, exists := a.symbols[fn.Name]; exists && !existing.IsBuiltin {
			a.addDuplicateDiagnostic(fn.Position, CategorySymbolError, "E_FUNC_DUPLICATE", fmt.Sprintf("function %s already defined", fn.Name), "", existing.Position, "previous function declaration")
			continue
		}
		// Do not overwrite builtin entries (e.g. Start in current builtin list), but still allow body analysis.
		if existing, exists := a.symbols[fn.Name]; exists && existing.IsBuiltin {
			continue
		}
		a.symbols[fn.Name] = &Symbol{
			Name:     fn.Name,
			IsFunc:   true,
			Position: fn.Position,
		}
	}
}

func (a *SemanticAnalyzer) analyzeFunction(fn *FunctionDecl) {
	if fn.Name == "Start" || fn.Name == "__Boot" {
		// Entry points are special
		if len(fn.Params) > 0 {
			a.addDiagnostic(fn.Position, CategoryValidationError, "E_ENTRY_PARAMS", fmt.Sprintf("function %s() must have no parameters", fn.Name), "")
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
			existing := a.symbols[s.Name]
			a.addDuplicateDiagnostic(s.Position, CategorySymbolError, "E_VAR_DUPLICATE", fmt.Sprintf("variable %s already defined", s.Name), "", existing.Position, "previous declaration")
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
			a.addDiagnostic(e.Position, CategorySymbolError, "E_IDENT_UNDEFINED", fmt.Sprintf("undefined identifier: %s", e.Name), "")
		}

	case *NumberExpr, *StringExpr, *BoolExpr:
		// Literals are fine
	}
}

func (a *SemanticAnalyzer) addDiagnostic(pos Position, category DiagnosticCategory, code, message, file string) {
	a.diagnostics = append(a.diagnostics, Diagnostic{
		Category: category,
		Code:     code,
		Message:  message,
		File:     file,
		Line:     pos.Line,
		Column:   pos.Column,
		Severity: SeverityError,
		Stage:    StageSemantic,
	})
}

func (a *SemanticAnalyzer) addDuplicateDiagnostic(pos Position, category DiagnosticCategory, code, message, file string, previous Position, previousMsg string) {
	d := Diagnostic{
		Category: category,
		Code:     code,
		Message:  message,
		File:     file,
		Line:     pos.Line,
		Column:   pos.Column,
		Severity: SeverityError,
		Stage:    StageSemantic,
	}
	if previous.Line > 0 {
		d.Related = append(d.Related, DiagnosticLocation{
			File:    file,
			Line:    previous.Line,
			Column:  previous.Column,
			Message: previousMsg,
		})
	}
	a.diagnostics = append(a.diagnostics, d)
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
