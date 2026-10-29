package corelx

import (
	"fmt"
	"strings"
)

func (a *SemanticAnalyzer) isNamespace(name string) bool {
	switch name {
	case "ppu", "sprite", "oam", "apu", "gfx", "input", "mem", "bg", "matrix",
		"matrix_plane", "raster", "text", "ym", "music", "boot":
		return true
	}
	return a.isRequestedModule(name)
}

func (a *SemanticAnalyzer) isDeclaredStruct(name string) bool {
	if name == "Sprite" {
		return true
	}
	for _, td := range a.program.Types {
		if td.Name == name {
			_, ok := td.Type.(*StructType)
			return ok
		}
	}
	return false
}

func (a *SemanticAnalyzer) acceptGlobalType(name string) error {
	switch name {
	case "u8", "i8", "int", "i16", "u16", "fixed":
		return nil
	default:
		if a.isDeclaredStruct(name) {
			return nil
		}
		return fmt.Errorf("unsupported global type %q (supported: int, u8, u16, fixed, and struct types)", name)
	}
}

func (a *SemanticAnalyzer) resolveArrayLen(t TypeExpr) {
	arr, ok := t.(*ArrayType)
	if !ok || arr == nil || arr.Len != 0 || arr.LenName == "" {
		return
	}
	v, ok := a.constVals[arr.LenName]
	if !ok || v <= 0 || v > 0x4000 {
		a.addDiagnostic(arr.Position, CategoryTypeError, "E_ARRAY_LEN", fmt.Sprintf("array length %s must be a constant between 1 and 16384", arr.LenName), "")
		return
	}
	arr.Len = int(v)
}

func (a *SemanticAnalyzer) pushScope() {
	a.scopes = append(a.scopes, scopeLayer{names: map[string]savedBinding{}})
}

func (a *SemanticAnalyzer) popScope() {
	top := a.scopes[len(a.scopes)-1]
	a.scopes = a.scopes[:len(a.scopes)-1]
	for name, b := range top.names {
		if b.had {
			a.symbols[name] = b.prev
		} else {
			delete(a.symbols, name)
		}
	}
}

func (a *SemanticAnalyzer) withScope(stmts []Stmt) {
	a.pushScope()
	for _, stmt := range stmts {
		a.analyzeStmt(stmt)
	}
	a.popScope()
}

func (a *SemanticAnalyzer) declareLocal(pos Position, name string, sym *Symbol) {
	if isButtonName(name) {
		a.addDiagnostic(pos, CategorySymbolError, "E_VAR_DUPLICATE", fmt.Sprintf("%s is a controller button name and cannot be redeclared", name), "")
		return
	}
	if len(a.scopes) == 0 {
		a.pushScope()
	}
	top := &a.scopes[len(a.scopes)-1]
	if _, exists := top.names[name]; exists {
		existing := a.symbols[name]
		prev := Position{}
		if existing != nil {
			prev = existing.Position
		}
		a.addDuplicateDiagnostic(pos, CategorySymbolError, "E_VAR_DUPLICATE", fmt.Sprintf("variable %s already defined", name), "", prev, "previous declaration")
		return
	}
	prev, had := a.symbols[name]
	top.names[name] = savedBinding{prev: prev, had: had}
	a.symbols[name] = sym
}

func (a *SemanticAnalyzer) isStructName(name string) bool {
	if name == "Sprite" || name == "Vec2" {
		return true
	}
	for _, td := range a.program.Types {
		if td.Name == name {
			return true
		}
	}
	return false
}

func (a *SemanticAnalyzer) functionByName(name string) *FunctionDecl {
	for _, fn := range a.program.Functions {
		if fn.Name == name {
			return fn
		}
	}
	return nil
}

func (a *SemanticAnalyzer) lookupUserFunc(name string) *FunctionDecl {
	if fn := a.functionByName(name); fn != nil {
		return fn
	}
	if i := strings.LastIndex(name, "."); i >= 0 && a.isRequestedModule(name[:i]) {
		return a.functionByName(name[i+1:])
	}
	return nil
}

func (a *SemanticAnalyzer) declType(s *VarDeclStmt) TypeExpr {
	if s.Type != nil {
		return s.Type
	}
	if call, ok := s.Value.(*CallExpr); ok {
		if ident, ok := call.Func.(*IdentExpr); ok && a.isStructName(ident.Name) && len(call.Args) == 0 {
			return &NamedType{Name: ident.Name}
		}
	}
	switch a.exprArithType(s.Value) {
	case typeFixed:
		return &NamedType{Name: "fixed"}
	case typeInt:
		return &NamedType{Name: "int"}
	case typeBool:
		return &NamedType{Name: "bool"}
	default:
		// Unknown stays unknown. Guessing u16 made a later fixed add a false mix error.
		return &NamedType{Name: ""}
	}
}

func (a *SemanticAnalyzer) checkInitType(pos Position, typeName string, value Expr, what string) {
	if typeName == "" || value == nil || a.isStructName(typeName) {
		return
	}
	declared := arithTypeForName(typeName)
	got := a.exprArithType(value)
	if (declared == typeFixed && got == typeInt) || (declared == typeInt && got == typeFixed) {
		a.addDiagnostic(pos, CategoryTypeError, "E_TYPE_MIX", fmt.Sprintf("cannot initialize %s %s with a %s value — convert explicitly with int(x) or fixed(x)", typeName, what, got), "")
	}
}

func (a *SemanticAnalyzer) checkAssign(s *AssignStmt) {
	targetName := ""
	switch t := s.Target.(type) {
	case *IdentExpr:
		if sym, ok := a.symbols[t.Name]; ok {
			targetName = typeExprName(sym.Type)
		}
	case *MemberExpr:
		if ident, ok := t.Object.(*IdentExpr); ok {
			if sym, ok := a.symbols[ident.Name]; ok && a.isStructName(typeExprName(sym.Type)) {
				targetName = a.structFieldType(typeExprName(sym.Type), t.Member)
			}
		}
	case *IndexExpr:
		if ident, ok := t.Array.(*IdentExpr); ok {
			if sym, ok := a.symbols[ident.Name]; ok {
				targetName = typeExprName(sym.Type)
			}
		}
	}
	if targetName == "" || a.isStructName(targetName) {
		return
	}
	declared := arithTypeForName(targetName)
	got := a.exprArithType(s.Value)
	if (declared == typeFixed && got == typeInt) || (declared == typeInt && got == typeFixed) {
		a.addDiagnostic(s.Position, CategoryTypeError, "E_TYPE_MIX", fmt.Sprintf("cannot assign %s value to %s — convert explicitly with int(x) or fixed(x)", got, declared), "")
	}
}

func (a *SemanticAnalyzer) structFieldType(structName, field string) string {
	for _, td := range a.program.Types {
		if td.Name != structName {
			continue
		}
		st, ok := td.Type.(*StructType)
		if !ok {
			return ""
		}
		for _, f := range st.Fields {
			if f.Name == field {
				return typeExprName(f.Type)
			}
		}
	}
	return ""
}

func (a *SemanticAnalyzer) checkReturn(s *ReturnStmt) {
	fn := a.currentFunc
	if fn == nil {
		return
	}
	if fn.ReturnType == nil {
		if s.Value != nil {
			a.addDiagnostic(s.Position, CategoryTypeError, "E_RETURN_VALUE", fmt.Sprintf("function %s does not return a value", fn.Name), "")
		}
		return
	}
	wantName := typeExprName(fn.ReturnType)
	if s.Value == nil {
		a.addDiagnostic(s.Position, CategoryTypeError, "E_RETURN_VALUE", fmt.Sprintf("function %s must return %s", fn.Name, wantName), "")
		return
	}
	a.checkTypeMatch(s.Position, "return", wantName, s.Value)
}

func (a *SemanticAnalyzer) checkCall(call *CallExpr) {
	name := callFuncName(call)
	if name == "" {
		return
	}
	if sym, ok := a.symbols[name]; ok && sym.IsBuiltin {
		return
	}
	if fn := a.lookupUserFunc(name); fn != nil {
		if a.currentFunc != nil {
			a.calls = append(a.calls, callEdge{from: a.currentFunc.Name, to: fn.Name, pos: call.Pos()})
		}
		if len(call.Args) != len(fn.Params) {
			a.addDiagnostic(call.Pos(), CategoryTypeError, "E_CALL_ARITY", fmt.Sprintf("%s expects %d arguments, got %d", name, len(fn.Params), len(call.Args)), "")
			return
		}
		for i, param := range fn.Params {
			if arr, ok := param.Type.(*ArrayType); ok {
				a.checkArrayArg(call.Pos(), name, i, arr, call.Args[i])
				continue
			}
			a.checkTypeMatch(call.Pos(), fmt.Sprintf("argument %d of %s", i+1, name), typeExprName(param.Type), call.Args[i])
		}
		return
	}
	if strings.Contains(name, ".") && !a.isRequestedModule(strings.SplitN(name, ".", 2)[0]) {
		a.addDiagnostic(call.Pos(), CategorySymbolError, "E_CALL_UNKNOWN", fmt.Sprintf("unknown function %s", name), "")
	}
}

func (a *SemanticAnalyzer) checkArrayArg(pos Position, fn string, index int, want *ArrayType, value Expr) {
	what := fmt.Sprintf("argument %d of %s", index+1, fn)
	ident, ok := value.(*IdentExpr)
	if !ok {
		a.addDiagnostic(pos, CategoryTypeError, "E_TYPE_MISMATCH", fmt.Sprintf("%s expects an array name", what), "")
		return
	}
	sym, ok := a.symbols[ident.Name]
	if !ok {
		return
	}
	got, ok := sym.Type.(*ArrayType)
	if !ok || typeExprName(got.Elem) != typeExprName(want.Elem) || got.Len != want.Len {
		a.addDiagnostic(pos, CategoryTypeError, "E_TYPE_MISMATCH", fmt.Sprintf("%s expects %s[%d]", what, typeExprName(want.Elem), want.Len), "")
	}
}

func (a *SemanticAnalyzer) checkTypeMatch(pos Position, what, wantName string, value Expr) {
	if wantName == "" {
		return
	}
	if a.isStructName(wantName) {
		got := a.valueTypeName(value)
		if got != wantName {
			a.addDiagnostic(pos, CategoryTypeError, "E_TYPE_MISMATCH", fmt.Sprintf("%s expects %s", what, wantName), "")
		}
		return
	}
	want := arithTypeForName(wantName)
	got := a.exprArithType(value)
	if want == typeUnknown || got == typeUnknown {
		return
	}
	if want != got {
		a.addDiagnostic(pos, CategoryTypeError, "E_TYPE_MISMATCH", fmt.Sprintf("%s expects %s, got %s", what, want, got), "")
	}
}

func (a *SemanticAnalyzer) valueTypeName(expr Expr) string {
	switch e := expr.(type) {
	case *IdentExpr:
		if sym, ok := a.symbols[e.Name]; ok {
			return typeExprName(sym.Type)
		}
	case *CallExpr:
		if ident, ok := e.Func.(*IdentExpr); ok && a.isStructName(ident.Name) {
			return ident.Name
		}
		if fn := a.lookupUserFunc(callFuncName(e)); fn != nil {
			return typeExprName(fn.ReturnType)
		}
	case *NumberExpr:
		if e.IsFixed {
			return "fixed"
		}
		return "int"
	}
	return ""
}

func (a *SemanticAnalyzer) exprArithType(expr Expr) string {
	if expr == nil {
		return typeUnknown
	}
	switch e := expr.(type) {
	case *NumberExpr:
		if e.IsFixed {
			return typeFixed
		}
		return typeInt
	case *BoolExpr:
		return typeBool
	case *IdentExpr:
		if sym, ok := a.symbols[e.Name]; ok {
			if a.constIsFixed(e.Name) {
				return typeFixed
			}
			return arithTypeForName(typeExprName(sym.Type))
		}
		return typeUnknown
	case *UnaryExpr:
		if e.Op == TOKEN_NOT {
			return typeBool
		}
		return a.exprArithType(e.Operand)
	case *BinaryExpr:
		switch e.Op {
		case TOKEN_PLUS, TOKEN_MINUS, TOKEN_STAR, TOKEN_SLASH, TOKEN_PERCENT,
			TOKEN_AMPERSAND, TOKEN_PIPE, TOKEN_CARET, TOKEN_LSHIFT, TOKEN_RSHIFT:
			lt, rt := a.exprArithType(e.Left), a.exprArithType(e.Right)
			if (lt == typeFixed && rt == typeInt) || (lt == typeInt && rt == typeFixed) {
				a.addDiagnostic(e.Position, CategoryTypeError, "E_TYPE_MIX", fmt.Sprintf("cannot mix fixed and int operands in '%s' — convert explicitly with int(x) or fixed(x)", tokenOpName(e.Op)), "")
			}
			if lt == typeFixed || rt == typeFixed {
				return typeFixed
			}
			if lt == typeUnknown && rt == typeUnknown {
				return typeUnknown
			}
			return typeInt
		default:
			return typeBool
		}
	case *IndexExpr:
		ident, ok := e.Array.(*IdentExpr)
		if !ok {
			return typeUnknown
		}
		sym, ok := a.symbols[ident.Name]
		if !ok {
			return typeUnknown
		}
		if arr, ok := sym.Type.(*ArrayType); ok {
			return arithTypeForName(typeExprName(arr.Elem))
		}
		return arithTypeForName(typeExprName(sym.Type))
	case *MemberExpr:
		ident, ok := e.Object.(*IdentExpr)
		if !ok {
			return typeUnknown
		}
		sym, ok := a.symbols[ident.Name]
		if !ok {
			return typeUnknown
		}
		ft := a.structFieldType(typeExprName(sym.Type), e.Member)
		if ft == "" {
			return typeUnknown
		}
		return arithTypeForName(ft)
	case *CallExpr:
		switch callFuncName(e) {
		case "int":
			return typeInt
		case "fixed":
			return typeFixed
		}
		if fn := a.lookupUserFunc(callFuncName(e)); fn != nil {
			return arithTypeForName(typeExprName(fn.ReturnType))
		}
		return typeUnknown
	default:
		return typeUnknown
	}
}

type callEdge struct {
	from string
	to   string
	pos  Position
}

func buttonNames() []string {
	return []string{"UP", "DOWN", "LEFT", "RIGHT", "A", "B", "X", "Y", "L", "R", "START", "Z"}
}

func isButtonName(name string) bool {
	for _, b := range buttonNames() {
		if b == name {
			return true
		}
	}
	return false
}

func duplicateNameMessage(name string) string {
	if isButtonName(name) {
		return fmt.Sprintf("%s is a controller button name and cannot be redeclared", name)
	}
	return fmt.Sprintf("duplicate declaration of %s", name)
}

func (a *SemanticAnalyzer) reportRecursion() {
	adj := map[string][]callEdge{}
	for _, e := range a.calls {
		adj[e.from] = append(adj[e.from], e)
	}
	color := map[string]int{}
	var stack []string
	var visit func(string)
	visit = func(name string) {
		color[name] = 1
		stack = append(stack, name)
		for _, e := range adj[name] {
			switch color[e.to] {
			case 1:
				a.addDiagnostic(e.pos, CategoryValidationError, "E_RECURSION",
					fmt.Sprintf("recursion is not supported (%s)", strings.Join(cyclePath(stack, e.to), " -> ")), "")
			case 0:
				visit(e.to)
			}
		}
		stack = stack[:len(stack)-1]
		color[name] = 2
	}
	for name := range adj {
		if color[name] == 0 {
			visit(name)
		}
	}
}

func cyclePath(stack []string, back string) []string {
	start := 0
	for i, name := range stack {
		if name == back {
			start = i
			break
		}
	}
	path := append([]string{}, stack[start:]...)
	return append(path, back)
}

func (a *SemanticAnalyzer) constIsFixed(name string) bool {
	for _, c := range a.program.Consts {
		if c.Name != name {
			continue
		}
		if n, ok := c.Value.(*NumberExpr); ok && n.IsFixed {
			return true
		}
	}
	return false
}

func listReturns(stmts []Stmt) bool {
	if len(stmts) == 0 {
		return false
	}
	return stmtReturns(stmts[len(stmts)-1])
}

func stmtReturns(s Stmt) bool {
	switch s := s.(type) {
	case *ReturnStmt:
		return true
	case *IfStmt:
		if len(s.Else) == 0 || !listReturns(s.Then) || !listReturns(s.Else) {
			return false
		}
		for _, c := range s.ElseIf {
			if !listReturns(c.Body) {
				return false
			}
		}
		return true
	case *WhileStmt:
		if !isAlwaysTrue(s.Condition) || listHasBreak(s.Body) {
			return false
		}
		return true
	default:
		return false
	}
}

func isAlwaysTrue(e Expr) bool {
	b, ok := e.(*BoolExpr)
	return ok && b.Value
}

func listHasBreak(stmts []Stmt) bool {
	for _, s := range stmts {
		switch s := s.(type) {
		case *BreakStmt:
			return true
		case *IfStmt:
			if listHasBreak(s.Then) || listHasBreak(s.Else) || listHasBreakClauses(s.ElseIf) {
				return true
			}
		}
	}
	return false
}

func listHasBreakClauses(clauses []*ElseIfClause) bool {
	for _, c := range clauses {
		if listHasBreak(c.Body) {
			return true
		}
	}
	return false
}
