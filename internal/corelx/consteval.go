package corelx

import "fmt"

// evalConstExpr evaluates a compile-time constant expression. Supported:
// integer literals, true/false (1/0), references to previously defined
// constants, unary minus / bitwise not, and integer binary arithmetic.
func evalConstExpr(expr Expr, consts map[string]int64) (int64, error) {
	switch e := expr.(type) {
	case *NumberExpr:
		return int64(e.Value), nil
	case *BoolExpr:
		if e.Value {
			return 1, nil
		}
		return 0, nil
	case *IdentExpr:
		if v, ok := consts[e.Name]; ok {
			return v, nil
		}
		return 0, fmt.Errorf("constant expression references %q, which is not a previously defined constant", e.Name)
	case *UnaryExpr:
		v, err := evalConstExpr(e.Operand, consts)
		if err != nil {
			return 0, err
		}
		switch e.Op {
		case TOKEN_MINUS:
			return -v, nil
		case TOKEN_TILDE:
			return ^v, nil
		default:
			return 0, fmt.Errorf("unsupported unary operator in constant expression")
		}
	case *BinaryExpr:
		l, err := evalConstExpr(e.Left, consts)
		if err != nil {
			return 0, err
		}
		r, err := evalConstExpr(e.Right, consts)
		if err != nil {
			return 0, err
		}
		switch e.Op {
		case TOKEN_PLUS:
			return l + r, nil
		case TOKEN_MINUS:
			return l - r, nil
		case TOKEN_STAR:
			return l * r, nil
		case TOKEN_SLASH:
			if r == 0 {
				return 0, fmt.Errorf("division by zero in constant expression")
			}
			return idiv16(l, r), nil
		case TOKEN_PERCENT:
			if r == 0 {
				return 0, fmt.Errorf("modulo by zero in constant expression")
			}
			return imod16(l, r), nil
		case TOKEN_AMPERSAND:
			return l & r, nil
		case TOKEN_PIPE:
			return l | r, nil
		case TOKEN_CARET:
			return l ^ r, nil
		case TOKEN_LSHIFT:
			return l << uint(r&63), nil
		case TOKEN_RSHIFT:
			return sar16(l, r), nil
		default:
			return 0, fmt.Errorf("unsupported binary operator in constant expression")
		}
	default:
		return 0, fmt.Errorf("expression is not a compile-time constant")
	}
}

// foldProgramConstsTyped evaluates all top-level const declarations in
// order, tracking which constants are fixed-point. Fixed const arithmetic
// supports + - between fixed values, fixed*fixed (rescaled), and
// fixed/fixed (rescaled); mixing fixed and int is an error (charter D4).
func foldProgramConstsTyped(prog *Program) (map[string]int64, map[string]bool, error) {
	consts := make(map[string]int64)
	fixed := make(map[string]bool)
	for _, c := range prog.Consts {
		if _, dup := consts[c.Name]; dup {
			return nil, nil, fmt.Errorf("line %d: duplicate constant %q", c.Position.Line, c.Name)
		}
		v, isFixed, err := evalConstExprTyped(c.Value, consts, fixed)
		if err != nil {
			return nil, nil, fmt.Errorf("line %d: const %s: %w", c.Position.Line, c.Name, err)
		}
		consts[c.Name] = v
		fixed[c.Name] = isFixed
	}
	return consts, fixed, nil
}

// evalConstExprTyped evaluates a constant expression and reports whether
// the result is fixed-point.
func evalConstExprTyped(expr Expr, consts map[string]int64, fixed map[string]bool) (int64, bool, error) {
	switch e := expr.(type) {
	case *NumberExpr:
		if e.IsFixed {
			return int64(int16(uint16(e.Value))), true, nil
		}
		return int64(e.Value), false, nil
	case *IdentExpr:
		if v, ok := consts[e.Name]; ok {
			return v, fixed[e.Name], nil
		}
		return 0, false, fmt.Errorf("constant expression references %q, which is not a previously defined constant", e.Name)
	case *UnaryExpr:
		v, isF, err := evalConstExprTyped(e.Operand, consts, fixed)
		if err != nil {
			return 0, false, err
		}
		switch e.Op {
		case TOKEN_MINUS:
			return -v, isF, nil
		case TOKEN_TILDE:
			return ^v, isF, nil
		}
		return 0, false, fmt.Errorf("unsupported unary operator in constant expression")
	case *BinaryExpr:
		l, lf, err := evalConstExprTyped(e.Left, consts, fixed)
		if err != nil {
			return 0, false, err
		}
		r, rf, err := evalConstExprTyped(e.Right, consts, fixed)
		if err != nil {
			return 0, false, err
		}
		if lf != rf {
			return 0, false, fmt.Errorf("cannot mix fixed and int in a constant expression — convert explicitly")
		}
		isF := lf
		switch e.Op {
		case TOKEN_PLUS:
			return l + r, isF, nil
		case TOKEN_MINUS:
			return l - r, isF, nil
		case TOKEN_STAR:
			if isF {
				// Must match the runtime __fixmul routine bit-for-bit: it
				// works on magnitudes (toward-zero truncation) then re-applies
				// sign, NOT Go's arithmetic >> (which rounds toward -inf and
				// diverges from hardware on negative truncating products).
				return fixmulFold(l, r), true, nil
			}
			return l * r, false, nil
		case TOKEN_SLASH:
			if r == 0 {
				return 0, false, fmt.Errorf("division by zero in constant expression")
			}
			if isF {
				return fixdivFold(l, r), true, nil
			}
			return idiv16(l, r), false, nil
		default:
			if isF {
				return 0, false, fmt.Errorf("operator not supported on fixed constants")
			}
		}
		// Fall through to untyped integer evaluation for bitwise ops.
		v, err2 := evalConstExpr(expr, consts)
		return v, false, err2
	default:
		v, err := evalConstExpr(expr, consts)
		return v, false, err
	}
}

// fixmulFold computes an 8.8 fixed multiply identically to the runtime
// __fixmul routine: operate on magnitudes (toward-zero), then re-sign. This
// keeps compile-time constant folding bit-identical to on-hardware results.
func fixmulFold(a, b int64) int64 {
	neg := (a < 0) != (b < 0)
	ua, ub := a, b
	if ua < 0 {
		ua = -ua
	}
	if ub < 0 {
		ub = -ub
	}
	res := (ua * ub) >> 8 // toward zero (both operands non-negative)
	res &= 0xFFFF         // 16-bit datapath truncation
	if neg {
		res = (-res) & 0xFFFF
	}
	return int64(int16(uint16(res)))
}

// idiv16 is signed 16-bit division, toward zero, wrapped to int16.
// A zero divisor yields 0xFFFF, matching hardware DIV and __divs.
func idiv16(a, b int64) int64 {
	ai := int32(int16(a))
	bi := int32(int16(b))
	if bi == 0 {
		return -1
	}
	return int64(int16(ai / bi))
}

// imod16 is the remainder paired with idiv16: a - (a/b)*b, sign follows a.
func imod16(a, b int64) int64 {
	ai := int32(int16(a))
	bi := int32(int16(b))
	if bi == 0 {
		return int64(int16(a))
	}
	q := int16(ai / bi)
	return int64(int16(ai - int32(q)*bi))
}

// sar16 is a 16-bit arithmetic right shift. The count is masked to 0..15,
// matching the CPU's SAR.
func sar16(a, n int64) int64 {
	count := uint(uint16(n) & 15)
	return int64(int16(a) >> count)
}

// fixdivFold is signed 8.8 division, identical to __fixdiv: magnitudes,
// toward zero, then the sign is restored. (a<<8)/b is split into
// ((a/b)<<8)+((a%b)<<8)/b so the quotient stays inside 16 bits the same
// way the helper's shifts do.
func fixdivFold(a, b int64) int64 {
	if int16(b) == 0 {
		return -1
	}
	neg := (int16(a) < 0) != (int16(b) < 0)
	ua := mag16(a)
	ub := mag16(b)
	if ub == 0 {
		return -1
	}
	qhi := uint16(uint32(ua) / uint32(ub))
	rem := uint16(uint32(ua) - uint32(qhi)*uint32(ub))
	qlo := uint16((uint32(rem) << 8) / uint32(ub))
	mag := uint16((uint32(qhi) << 8) + uint32(qlo))
	if neg {
		mag = uint16(-int16(mag))
	}
	return int64(int16(mag))
}

func mag16(v int64) uint16 {
	x := int16(v)
	if x < 0 {
		x = -x
	}
	return uint16(x)
}
