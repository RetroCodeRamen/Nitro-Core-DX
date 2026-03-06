package main

import "nitro-core-dx/internal/corelx"

func isKeyword(tt corelx.TokenType) bool {
	switch tt {
	case corelx.TOKEN_FUNCTION, corelx.TOKEN_IF, corelx.TOKEN_ELSEIF, corelx.TOKEN_ELSE,
		corelx.TOKEN_WHILE, corelx.TOKEN_FOR, corelx.TOKEN_RETURN, corelx.TOKEN_TYPE,
		corelx.TOKEN_STRUCT, corelx.TOKEN_ASSET, corelx.TOKEN_TRUE, corelx.TOKEN_FALSE,
		corelx.TOKEN_AND, corelx.TOKEN_OR, corelx.TOKEN_NOT:
		return true
	default:
		return false
	}
}

func tokenFallbackLiteral(tt corelx.TokenType) string {
	switch tt {
	case corelx.TOKEN_ASSIGN:
		return ":="
	case corelx.TOKEN_EQUAL:
		return "="
	case corelx.TOKEN_PLUS:
		return "+"
	case corelx.TOKEN_MINUS:
		return "-"
	case corelx.TOKEN_STAR:
		return "*"
	case corelx.TOKEN_SLASH:
		return "/"
	case corelx.TOKEN_PERCENT:
		return "%"
	case corelx.TOKEN_EQUAL_EQUAL:
		return "=="
	case corelx.TOKEN_BANG_EQUAL:
		return "!="
	case corelx.TOKEN_LESS:
		return "<"
	case corelx.TOKEN_LESS_EQUAL:
		return "<="
	case corelx.TOKEN_GREATER:
		return ">"
	case corelx.TOKEN_GREATER_EQUAL:
		return ">="
	case corelx.TOKEN_AMPERSAND:
		return "&"
	case corelx.TOKEN_PIPE:
		return "|"
	case corelx.TOKEN_CARET:
		return "^"
	case corelx.TOKEN_TILDE:
		return "~"
	case corelx.TOKEN_LSHIFT:
		return "<<"
	case corelx.TOKEN_RSHIFT:
		return ">>"
	case corelx.TOKEN_ADDR_OF:
		return "&"
	case corelx.TOKEN_LPAREN:
		return "("
	case corelx.TOKEN_RPAREN:
		return ")"
	case corelx.TOKEN_LBRACKET:
		return "["
	case corelx.TOKEN_RBRACKET:
		return "]"
	case corelx.TOKEN_COMMA:
		return ","
	case corelx.TOKEN_COLON:
		return ":"
	case corelx.TOKEN_ARROW:
		return "->"
	case corelx.TOKEN_DOT:
		return "."
	default:
		return ""
	}
}
