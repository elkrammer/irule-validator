package ast

import (
	"github.com/elkrammer/irule-validator/token"
	"testing"
)

func TestString(t *testing.T) {
	program := &Program{
		Statements: []Statement{
			&ExpressionStatement{
				Expression: &CallExpression{
					Function: &Identifier{
						Token: token.Token{Type: token.IDENT, Literal: "puts"},
						Value: "puts",
					},
					Arguments: []Expression{
						&StringLiteral{
							Token: token.Token{Type: token.STRING, Literal: "Hello, world!"},
							Value: "Hello, world!",
						},
					},
				},
			},
		},
	}

	expected := `puts("Hello, world!")`

	if program.String() != expected {
		t.Errorf("program.String() wrong. Got=%q, Expected=%q", program.String(), expected)
	}
}
