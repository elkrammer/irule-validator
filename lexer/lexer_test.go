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

func TestNextToken(t *testing.T) {
	input := `set five 5;
set ten 10;

proc add {x y} {
  return [expr {$x + $y}]
}

set result [add $five $ten];
expr {-5};
expr {5 < 10 > 5};

if {5 < 10} {
	return true;
} else {
	return false;
}

expr {10 == 10};
expr {10 != 9};
"foobar"
"foo bar"
{1 2}
{"foo" "bar"}
`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.SET, "set"},
		{token.IDENT, "five"},
		{token.NUMBER, "5"},
		{token.SEMICOLON, ";"},
		{token.SET, "set"},
		{token.IDENT, "ten"},
		{token.NUMBER, "10"},
		{token.SEMICOLON, ";"},
		{token.FUNCTION, "proc"},
		{token.IDENT, "add"},
		{token.LBRACE, "{"},
		{token.IDENT, "x"},
		{token.IDENT, "y"},
		{token.RBRACE, "}"},
		{token.LBRACE, "{"},
		{token.RETURN, "return"},
		{token.LBRACKET, "["},
		{token.EXPR, "expr"},
		{token.LBRACE, "{"},
		{token.IDENT, "$x"},
		{token.PLUS, "+"},
		{token.IDENT, "$y"},
		{token.RBRACE, "}"},
		{token.RBRACKET, "]"},
		{token.RBRACE, "}"},
		{token.SET, "set"},
		{token.IDENT, "result"},
		{token.LBRACKET, "["},
		{token.IDENT, "add"},
		{token.IDENT, "$five"},
		{token.IDENT, "$ten"},
		{token.RBRACKET, "]"},
		{token.SEMICOLON, ";"},
		{token.EXPR, "expr"},
		{token.LBRACE, "{"},
		{token.MINUS, "-"},
		{token.NUMBER, "5"},
		{token.RBRACE, "}"},
		{token.SEMICOLON, ";"},
		{token.EXPR, "expr"},
		{token.LBRACE, "{"},
		{token.NUMBER, "5"},
		{token.LT, "<"},
		{token.NUMBER, "10"},
		{token.GT, ">"},
		{token.NUMBER, "5"},
		{token.RBRACE, "}"},
		{token.SEMICOLON, ";"},
		{token.IF, "if"},
		{token.LBRACE, "{"},
		{token.NUMBER, "5"},
		{token.LT, "<"},
		{token.NUMBER, "10"},
		{token.RBRACE, "}"},
		{token.LBRACE, "{"},
		{token.RETURN, "return"},
		{token.TRUE, "true"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},
		{token.ELSE, "else"},
		{token.LBRACE, "{"},
		{token.RETURN, "return"},
		{token.FALSE, "false"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},
		{token.EXPR, "expr"},
		{token.LBRACE, "{"},
		{token.NUMBER, "10"},
		{token.EQ, "=="},
		{token.NUMBER, "10"},
		{token.RBRACE, "}"},
		{token.SEMICOLON, ";"},
		{token.EXPR, "expr"},
		{token.LBRACE, "{"},
		{token.NUMBER, "10"},
		{token.NOT_EQ, "!="},
		{token.NUMBER, "9"},
		{token.RBRACE, "}"},
		{token.SEMICOLON, ";"},
		{token.STRING, "foobar"},
		{token.STRING, "foo bar"},
		{token.LBRACE, "{"},
		{token.NUMBER, "1"},
		{token.NUMBER, "2"},
		{token.RBRACE, "}"},
		{token.LBRACE, "{"},
		{token.STRING, "foo"},
		{token.STRING, "bar"},
		{token.RBRACE, "}"},
		{token.EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}
