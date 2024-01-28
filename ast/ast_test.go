package ast

import (
	"testing"

	"github.com/elkrammer/irule-validator/token"
)

func TestString(t *testing.T) {
	program := &Program{
		Statements: []Statement{
			&ReturnStatement{
				Token: token.Token{Type: token.RETURN, Literal: "return"},
				ReturnValue: &Identifier{
					Token: token.Token{Type: token.STRING, Literal: "true"},
					Value: "true",
				},
				// Value: &Identifier{
				// 	Token: token.Token{Type: token.IDENT, Literal: "anotherVar"},
				// 	Value: "anotherVar",
				// },
			},
		},
	}

	if program.String() != "return true;" {
		t.Errorf("program.String() wrong. Got=%q", program.String())
	}
}
