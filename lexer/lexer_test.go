package lexer

import (
	"github.com/elkrammer/irule-validator/token"
	"testing"
)

func TestNextToken(t *testing.T) {
	// input := `when HTTP_REQUEST {
	//    if { [HTTP::uri] starts_with "/oldpath" } {
	//        HTTP::redirect "/newpath[HTTP::uri]"
	//    }
	//  }`
	input := `when HTTP_REQUEST {`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.WHEN, "when"},
		{token.HTTP_REQUEST, "HTTP_REQUEST"},
		{token.LBRACE, "{"},

		// {token.IF, "if"},
		// {token.LBRACE, "{"},
		// {token.LBRACKET, "["},
		// {token.HTTP_URI, "HTTP::uri"},
		// {token.RBRACKET, "]"},
		// {token.STARTS_WITH, "starts_with"},
		//
		// {token.LBRACKET, "["},
		// {token.STRING, `"/oldpath"`},
		// {token.RBRACKET, "]"},
		// {token.RPAREN, ")"},
		// {token.LBRACE, "{"},
		// {token.HTTP_RESPONSE, "HTTP_RESPONSE"},
		// {token.DOUBLE_COLON, "::"},
		// {token.REDIRECT, "redirect"},
		// {token.STRING, `"/newpath`},
		// {token.LBRACKET, "["},
		// {token.HTTP_URI, "HTTP::uri"},
		// {token.RBRACKET, "]"},
		// {token.STRING, `"`},
		// {token.RPAREN, ")"},
		// {token.SEMICOLON, ";"},
		// {token.RBRACE, "}"},
		// {token.RBRACE, "}"},
		// {token.EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. Expected = %q, got = %q", i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - tokentype wrong. Expected = %q, got = %q", i, tt.expectedLiteral, tok.Literal)
		}
	}
}
