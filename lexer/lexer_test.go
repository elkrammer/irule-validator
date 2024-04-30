package lexer

import (
	"github.com/elkrammer/irule-validator/token"
	"testing"
)

// TestVariable does simple variable testing.
func TestVariable(t *testing.T) {
	input := `$a + $b`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.IDENT, "$a"},
		{token.PLUS, "+"},
		{token.IDENT, "$b"},
		{token.EOF, ""},
	}

	l := New(input)
	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong, expected=%q, got=%q: %v", i, tt.expectedType, tok.Type, tok)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - Literal wrong, expected=%q, got=%q: %v", i, tt.expectedLiteral, tok.Literal, tok)
		}
	}
}

// func TestNextToken(t *testing.T) {
// 	input := `when HTTP_REQUEST {
// 	   if { [HTTP::uri] starts_with "/oldpath" } {
// 	       HTTP::redirect "/newpath"
// 	   }
//      return true;
// 	 }`
//
// 	tests := []struct {
// 		expectedType    token.TokenType
// 		expectedLiteral string
// 	}{
// 		{token.WHEN, "when"},
// 		{token.HTTP_REQUEST, "HTTP_REQUEST"},
// 		{token.LBRACE, "{"},
//
// 		{token.IF, "if"},
// 		{token.LBRACE, "{"},
// 		{token.LBRACKET, "["},
// 		{token.HTTP_URI, "HTTP::uri"},
// 		{token.RBRACKET, "]"},
// 		{token.STARTS_WITH, "starts_with"},
// 		{token.STRING, "/oldpath"},
// 		{token.RBRACE, "}"},
// 		{token.LBRACE, "{"},
//
// 		{token.HTTP_REDIRECT, "HTTP::redirect"},
// 		{token.STRING, "/newpath"},
// 		{token.RBRACE, "}"},
// 		{token.RETURN, "return"},
// 		{token.TRUE, "true"},
// 		{token.SEMICOLON, ";"},
// 		{token.RBRACE, "}"},
// 		{token.EOF, ""},
// 	}
//
// 	l := New(input)
//
// 	for i, tt := range tests {
// 		tok := l.NextToken()
//
// 		if tok.Type != tt.expectedType {
// 			t.Fatalf("tests[%d] - tokentype wrong. Expected = %q, got = %q", i, tt.expectedType, tok.Type)
// 		}
//
// 		if tok.Literal != tt.expectedLiteral {
// 			t.Fatalf("tests[%d] - tokentype wrong. Expected = %q, got = %q", i, tt.expectedLiteral, tok.Literal)
// 		}
// 	}
// }
