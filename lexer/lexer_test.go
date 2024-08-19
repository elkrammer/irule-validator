package lexer

import (
	"github.com/elkrammer/irule-validator/token"
	"testing"
)

func TestIRuleVariables(t *testing.T) {
	input := `
when HTTP_REQUEST {
    set client_ip [IP::client_addr]
    set host [HTTP::host]
    if { $host equals "example.com" } {
        log local0. "Request from $client_ip to $host"
    }
}
`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.WHEN, "when"},
		{token.HTTP_REQUEST, "HTTP_REQUEST"},
		{token.LBRACE, "{"},

		{token.SET, "set"},
		{token.IDENT, "client_ip"},
		{token.LBRACKET, "["},
		{token.IP_CLIENT_ADDR, "IP::client_addr"},
		{token.RBRACKET, "]"},

		{token.SET, "set"},
		{token.IDENT, "host"},
		{token.LBRACKET, "["},
		{token.HTTP_HOST, "HTTP::host"},
		{token.RBRACKET, "]"},

		{token.IF, "if"},
		{token.LBRACE, "{"},
		{token.IDENT, "$host"},
		{token.EQ, "equals"},
		{token.STRING, "example.com"},
		{token.RBRACE, "}"},
		{token.LBRACE, "{"},

		{token.IDENT, "log"},
		{token.IDENT, "local0."},
		{token.STRING, "Request from $client_ip to $host"},

		{token.RBRACE, "}"},
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

func TestNextToken(t *testing.T) {
	input := `
when HTTP_REQUEST {
    set uri [HTTP::uri]
    if { [HTTP::uri] starts_with "/api" } {
        log local0. "API request received"
        pool api_pool
    } elseif { [class match [IP::client_addr] eq "internal_network"] } {
        log local0. "Internal network access"
        pool internal_pool
    } else {
        HTTP::redirect "https://www.example.com"
    }
}
`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.WHEN, "when"},
		{token.HTTP_REQUEST, "HTTP_REQUEST"},
		{token.LBRACE, "{"},

		{token.SET, "set"},
		{token.IDENT, "uri"},
		{token.LBRACKET, "["},
		{token.HTTP_URI, "HTTP::uri"},
		{token.RBRACKET, "]"},

		{token.IF, "if"},
		{token.LBRACE, "{"},
		{token.LBRACKET, "["},
		{token.HTTP_URI, "HTTP::uri"},
		{token.RBRACKET, "]"},
		{token.STARTS_WITH, "starts_with"},
		{token.STRING, "/api"},
		{token.RBRACE, "}"},
		{token.LBRACE, "{"},

		{token.IDENT, "log"},
		{token.IDENT, "local0."},
		{token.STRING, "API request received"},

		{token.IDENT, "pool"},
		{token.IDENT, "api_pool"},

		{token.RBRACE, "}"},

		{token.ELSEIF, "elseif"},
		{token.LBRACE, "{"},
		{token.LBRACKET, "["},
		{token.IDENT, "class"},
		{token.MATCH, "match"},
		{token.LBRACKET, "["},
		{token.IP_CLIENT_ADDR, "IP::client_addr"},
		{token.RBRACKET, "]"},
		{token.EQ, "=="},
		{token.STRING, "internal_network"},
		{token.RBRACKET, "]"},
		{token.RBRACE, "}"},
		{token.LBRACE, "{"},

		{token.IDENT, "log"},
		{token.IDENT, "local0."},
		{token.STRING, "Internal network access"},

		{token.IDENT, "pool"},
		{token.IDENT, "internal_pool"},

		{token.RBRACE, "}"},

		{token.ELSE, "else"},
		{token.LBRACE, "{"},

		{token.HTTP_REDIRECT, "HTTP::redirect"},
		{token.STRING, "https://www.example.com"},

		{token.RBRACE, "}"},
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

func TestEdgeCaseTokens(t *testing.T) {
	input := `
    set uri    [HTTP::uri ]
    set host  [  HTTP::host   ]
    if { [HTTP::uri ] eq "/test" } {
        log local0. "Matched"
    }
    `

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.SET, "set"},
		{token.IDENT, "uri"},
		{token.LBRACKET, "["},
		{token.HTTP_URI, "HTTP::uri"},
		{token.RBRACKET, "]"},

		// More expected tokens...
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
