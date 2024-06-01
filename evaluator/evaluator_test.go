package evaluator

import (
	"github.com/elkrammer/irule-validator/lexer"
	"github.com/elkrammer/irule-validator/object"
	"github.com/elkrammer/irule-validator/parser"

	"testing"
)

func TestEvalNumberExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"5", 5},
		{"10", 10},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testNumberObject(t, evaluated, tt.expected)
	}
}

func testEval(input string) object.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	return Eval(program)
}

func testNumberObject(t *testing.T, obj object.Object, expected float64) bool {
	result, ok := obj.(*object.Number)
	if !ok {
		t.Errorf("object is not a number. got=%T (%v))", obj, obj)
		return false
	}

	if result.Value != expected {
		t.Errorf("object has wrong value. got %f, want=%f", result.Value, expected)
		return false
	}

	return true
}
