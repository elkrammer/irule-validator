package ast

import (
	"testing"

	"github.com/elkrammer/irule-validator/token"
)

// func TestString(t *testing.T) {
// 	program := &Program{
// 		Statements: []Statement{
// 			&ReturnStatement{
// 				Token: token.Token{Type: token.RETURN, Literal: "return"},
// 				ReturnValue: &Identifier{
// 					Token: token.Token{Type: token.STRING, Literal: "true"},
// 					Value: "true",
// 				},
// 				// Value: &Identifier{
// 				// 	Token: token.Token{Type: token.IDENT, Literal: "anotherVar"},
// 				// 	Value: "anotherVar",
// 				// },
// 			},
// 		},
// 	}
//
// 	if program.String() != "return true;" {
// 		t.Errorf("program.String() wrong. Got=%q", program.String())
// 	}
// }

func TestString(t *testing.T) {
	program := &Program{
		Statements: []Statement{
			&ExpressionStatement{
				Token: token.Token{Type: token.WHEN, Literal: "when"},
				Expression: &InfixExpression{
					Left: &Identifier{
						Token: token.Token{Type: token.HTTP_REQUEST, Literal: "HTTP_REQUEST"},
						Value: "HTTP_REQUEST",
					},
					Operator: token.PLUS,
					Right: &StringLiteral{
						Token: token.Token{Type: token.LBRACE, Literal: "{"},
						Value: "{",
					},
				},
			},
		},
	}

	expectedOutput := "when HTTP_REQUEST + {"
	actualOutput := program.String()

	if actualOutput != expectedOutput {
		t.Errorf("program.String() wrong. Got:\n%s\nExpected:\n%s", actualOutput, expectedOutput)
	}
}
