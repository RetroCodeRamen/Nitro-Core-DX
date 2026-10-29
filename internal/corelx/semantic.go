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
	calls       []callEdge
	constVals   map[string]int64
	scopes      []scopeLayer
}

type savedBinding struct {
	prev *Symbol
	had  bool
}

type scopeLayer struct {
	names map[string]savedBinding
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

	// Predefined button-name constants for the input builtins.
	for _, b := range buttonNames() {
		analyzer.symbols[b] = &Symbol{Name: b}
	}

	// Analyze types
	for _, typeDecl := range program.Types {
		analyzer.analyzeType(typeDecl)
	}

	// Analyze assets
	for _, asset := range program.Assets {
		analyzer.analyzeAsset(asset)
	}

	// Register top-level constants and globals so identifier references and
	// constant expressions validate. Errors surface as diagnostics.
	constVals := make(map[string]int64)
	for _, c := range program.Consts {
		if _, dup := analyzer.symbols[c.Name]; dup {
			analyzer.addDiagnostic(c.Position, CategorySymbolError, "E_CONST_DUPLICATE", duplicateNameMessage(c.Name), "")
			continue
		}
		if _, err := evalConstExpr(c.Value, constVals); err != nil {
			analyzer.addDiagnostic(c.Position, CategoryValidationError, "E_CONST_NOT_CONSTANT", fmt.Sprintf("const %s: %v", c.Name, err), "")
		} else {
			v, _ := evalConstExpr(c.Value, constVals)
			constVals[c.Name] = v
		}
		analyzer.symbols[c.Name] = &Symbol{Name: c.Name, Position: c.Position}
	}
	analyzer.constVals = constVals
	for _, g := range program.Globals {
		if _, dup := analyzer.symbols[g.Name]; dup {
			analyzer.addDiagnostic(g.Position, CategorySymbolError, "E_GLOBAL_DUPLICATE", duplicateNameMessage(g.Name), "")
			continue
		}
		if g.ArrayLenName != "" {
			v, ok := constVals[g.ArrayLenName]
			if !ok || v <= 0 || v > 0x4000 {
				analyzer.addDiagnostic(g.Position, CategoryTypeError, "E_ARRAY_LEN", fmt.Sprintf("array length %s must be a constant between 1 and 16384", g.ArrayLenName), "")
			} else {
				g.ArrayLen = int(v)
			}
		}
		if analyzer.isDeclaredStruct(g.TypeName) {
			if g.Init != nil || len(g.InitList) > 0 {
				analyzer.addDiagnostic(g.Position, CategoryTypeError, "E_GLOBAL_TYPE", fmt.Sprintf("struct global %s starts at zero; assign its fields", g.Name), "")
			}
		} else if g.Init != nil {
			analyzer.checkInitType(g.Position, g.TypeName, g.Init, "global "+g.Name)
		}
		if err := analyzer.acceptGlobalType(g.TypeName); err != nil {
			analyzer.addDiagnostic(g.Position, CategoryTypeError, "E_GLOBAL_TYPE", fmt.Sprintf("global %s: %v", g.Name, err), "")
		}
		var gType TypeExpr = &NamedType{Name: g.TypeName}
		if g.ArrayLen > 0 {
			gType = &ArrayType{Elem: &NamedType{Name: g.TypeName}, Len: g.ArrayLen}
		}
		analyzer.symbols[g.Name] = &Symbol{Name: g.Name, Type: gType, Position: g.Position}
	}

	// Register/analyze function declarations (name collisions first)
	analyzer.registerFunctionDecls()

	// Analyze functions
	for _, fn := range program.Functions {
		analyzer.analyzeFunction(fn)
	}
	analyzer.reportRecursion()

	// Check for entry point function (Start or __Boot)
	hasEntry := false
	for _, fn := range program.Functions {
		if fn.Name == "Start" || fn.Name == "__Boot" {
			hasEntry = true
			break
		}
	}
	if !hasEntry {
		analyzer.addDiagnostic(Position{}, CategoryValidationError, "E_MISSING_ENTRYPOINT", "missing required function: Start() or __Boot()", "")
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
		"int", "fixed", // charter D4 numeric conversions
		"text.draw", "text.draw_int", // HUD text via the text port
		"wait_vblank", "frame_counter",
		"sprite.set_pos", "sprite.set_size", "oam.write", "oam.write_sprite_data", "oam.clear_sprite", "oam.flush",
		// LEGACY (scaffolding): apu.* drives the legacy 4-channel synth and is
		// transitional only. The final audio subsystem is YM2608/OPNA; these
		// builtins are pending replacement by the future music.* YM2608 API.
		"apu.enable", "apu.set_channel_wave", "apu.set_channel_freq",
		"apu.set_channel_volume", "apu.note_on", "apu.note_off",
		// YM2608 audio subsystem: low-level register escape hatch (port 0 / port 1).
		"ym.write", "ym.write_port1",
		// YM2608 audio subsystem: high-level song playback over an external
		// .ncdxmusic asset (registered here only as each builtin's codegen
		// case lands, to avoid a registered-but-uncodegen'd gap).
		"music.play", "music.play_loop", "music.stop", "music.set_volume", "music.fade_to", "music.play_jingle",
		"ppu.enable_display", "gfx.load_tiles", "gfx.set_palette", "gfx.set_palette_color", "gfx.init_default_palettes",
		"boot.show_default",
		"input.read", "input.poll", "input.held", "input.pressed", "input.released",
		// Sprite attribute helpers. Implemented in codegen_builtin_sprite.go.
		"SPR_PAL", "SPR_HFLIP", "SPR_VFLIP", "SPR_PRI",
		"SPR_ENABLE", "SPR_SIZE_8", "SPR_SIZE_16",
		"SPR_SIZE_32X16", "SPR_SIZE_32X32", "SPR_SIZE_64X32", "SPR_SIZE_64X64", "SPR_SIZE_128X64", "SPR_SIZE_128X128",
		"SPR_BLEND", "SPR_ALPHA",
		"mem.write", "mem.read", "mem.write16", "mem.read16",
		"bg.set_scroll", "bg.enable", "bg.disable", "bg.set_priority", "bg.set_tilemap_base", "bg.load_tilemap", "bg.set_source_mode", "bg.bind_transform", "bg.set_tile_size",
		"bg.set_tile", "bg.fill_span", "bg.clear",
		"matrix_plane.enable", "matrix_plane.disable", "matrix_plane.load_bitmap", "matrix_plane.set_projection", "matrix_plane.set_depth", "matrix_plane.set_camera", "matrix_plane.set_surface", "matrix_plane.set_flags", "matrix_plane.load_tiles", "matrix_plane.load_tilemap", "matrix_plane.set_tile", "matrix_plane.fill_rect", "matrix_plane.clear",
		"raster.enable", "raster.disable",
		"raster.set_scanline_scroll", "raster.set_scanline_matrix", "raster.set_scanline_center", "raster.set_scanline_tilemap_base",
		"raster.set_scanline_rebind", "raster.set_scanline_priority", "raster.set_scanline_source_mode",
		"matrix.enable", "matrix.disable", "matrix.bind", "matrix.set_matrix", "matrix.set_center", "matrix.identity", "matrix.set_flags",
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
	// Also register the plain asset name (charter: no ASSET_ prefix) so
	// `load_bitmap(ParkFloor, 0)` and similar resolve.
	if _, exists := a.symbols[asset.Name]; !exists {
		a.symbols[asset.Name] = &Symbol{
			Name:      asset.Name,
			Type:      &NamedType{Name: "u16"},
			IsBuiltin: false,
			Position:  asset.Position,
		}
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
		if len(fn.Params) > 0 {
			a.addDiagnostic(fn.Position, CategoryValidationError, "E_ENTRY_PARAMS", fmt.Sprintf("function %s() must have no parameters", fn.Name), "")
		}
	}

	oldFunc := a.currentFunc
	a.currentFunc = fn
	// Save and restore symbol table so function-scoped locals don't leak.
	savedSymbols := make(map[string]*Symbol, len(a.symbols))
	for k, v := range a.symbols {
		savedSymbols[k] = v
	}
	defer func() {
		a.currentFunc = oldFunc
		a.symbols = savedSymbols
	}()

	// Parameters and the function body share one scope. Blocks inside the
	// body push their own, so a name in an if does not escape that if.
	a.pushScope()
	defer a.popScope()
	for _, param := range fn.Params {
		a.resolveArrayLen(param.Type)
		a.declareLocal(param.Position, param.Name, &Symbol{
			Name:     param.Name,
			Type:     param.Type,
			Position: param.Position,
		})
	}

	for _, stmt := range fn.Body {
		a.analyzeStmt(stmt)
	}
	if fn.ReturnType != nil && !listReturns(fn.Body) {
		a.addDiagnostic(fn.Position, CategoryTypeError, "E_RETURN_MISSING", fmt.Sprintf("function %s must return %s on every path", fn.Name, typeExprName(fn.ReturnType)), "")
	}
}

func (a *SemanticAnalyzer) analyzeStmt(stmt Stmt) {
	switch s := stmt.(type) {
	case *VarDeclStmt:
		a.resolveArrayLen(s.Type)
		a.declareLocal(s.Position, s.Name, &Symbol{
			Name:     s.Name,
			Type:     a.declType(s),
			Position: s.Position,
		})
		if s.Type != nil {
			a.checkInitType(s.Position, typeExprName(s.Type), s.Value, "variable "+s.Name)
		}
		a.analyzeExpr(s.Value)

	case *AssignStmt:
		a.analyzeExpr(s.Target)
		a.analyzeExpr(s.Value)
		a.checkAssign(s)

	case *IfStmt:
		a.analyzeExpr(s.Condition)
		a.withScope(s.Then)
		for _, clause := range s.ElseIf {
			a.analyzeExpr(clause.Condition)
			a.withScope(clause.Body)
		}
		a.withScope(s.Else)

	case *WhileStmt:
		a.analyzeExpr(s.Condition)
		a.withScope(s.Body)

	case *ForStmt:
		// The loop variable exists for the bounds and the body, then goes away.
		a.pushScope()
		a.declareLocal(s.Position, s.VarName, &Symbol{
			Name:     s.VarName,
			Type:     &NamedType{Name: "int"},
			Position: s.Position,
		})
		a.analyzeExpr(s.Start)
		a.analyzeExpr(s.End)
		if s.Step != nil {
			a.analyzeExpr(s.Step)
		}
		for _, stmt := range s.Body {
			a.analyzeStmt(stmt)
		}
		a.popScope()

	case *ReturnStmt:
		if s.Value != nil {
			a.analyzeExpr(s.Value)
		}
		a.checkReturn(s)

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
		a.checkCall(e)
		a.analyzeExpr(e.Func)
		for _, arg := range e.Args {
			a.analyzeExpr(arg)
		}

	case *MemberExpr:
		if ident, ok := e.Object.(*IdentExpr); !ok || !a.isNamespace(ident.Name) {
			a.analyzeExpr(e.Object)
		}

	case *IndexExpr:
		a.analyzeExpr(e.Array)
		a.analyzeExpr(e.Index)

	case *IdentExpr:
		if a.isNamespace(e.Name) {
			a.addDiagnostic(e.Position, CategoryTypeError, "E_NAMESPACE_VALUE", fmt.Sprintf("%s is a namespace, not a value", e.Name), "")
			return
		}
		if _, exists := a.symbols[e.Name]; !exists {
			a.addDiagnostic(e.Position, CategorySymbolError, "E_IDENT_UNDEFINED", fmt.Sprintf("undefined identifier: %s", e.Name), "")
		}

	case *NumberExpr, *StringExpr, *BoolExpr:
		// Literals are fine
	}
}

// isRequestedModule reports whether name is one of the program's `--!
// modules:` namespaces (charter D1).
func (a *SemanticAnalyzer) isRequestedModule(name string) bool {
	for _, m := range a.program.Modules {
		if m == name {
			return true
		}
	}
	return false
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
		if e.IsFixed {
			return &NamedType{Name: "fixed"}
		}
		return &NamedType{Name: "int"}
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
