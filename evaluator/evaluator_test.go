package evaluator

import (
	"github.com/elkrammer/irule-validator/lexer"
	"github.com/elkrammer/irule-validator/object"
	"github.com/elkrammer/irule-validator/parser"
	"strings"
	"testing"
)

func TestEvalNumberExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"5", 5},
		{"10", 10},
		{"-5", -5},
		{"-10", -10},
		{"5 + 5 + 5 + 5 - 10", 10},
		{"2 * 2 * 2 * 2 * 2", 32},
		{"-50 + 100 + -50", 0},
		{"5 * 2 + 10", 20},
		{"5 + 2 * 10", 25},
		{"20 + 2 * -10", 0},
		{"50 / 2 * 2 + 10", 60},
		{"2 * (5 + 10)", 30},
		{"3 * 3 * 3 + 10", 37},
		{"3 * (3 * 3) + 10", 37},
		{"(5 + 10 * 2 + 15 / 3) * 2 + -10", 50},
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
	env := object.NewEnvironment()

	return Eval(program, env)
}

func testNumberObject(t *testing.T, obj object.Object, expected float64) bool {

	if obj == nil {
		t.Errorf("object is nil. expected=%.2f", expected)
		return false
	}

	result, ok := obj.(*object.Number)
	if !ok {
		t.Errorf("object is not a number. got=%T (%+v)", obj, obj)
		return false
	}
	if result.Value != expected {
		t.Errorf("object has wrong value. got=%f, want=%f",
			result.Value, expected)
		return false
	}

	return true
}

func TestEvalBooleanExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"false", false},
		{"1 < 2", true},
		{"1 > 2", false},
		{"1 < 1", false},
		{"1 > 1", false},
		{"1 == 1", true},
		{"1 == 2", false},
		{"1 != 2", true},
		{"1 != 1", false},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testBooleanObject(t, evaluated, tt.expected)
	}
}

func testBooleanObject(t *testing.T, obj object.Object, expected bool) bool {
	result, ok := obj.(*object.Boolean)
	if !ok {
		t.Errorf("object is not a boolean. got=%T (%+v)", obj, obj)
		return false
	}

	if result.Value != expected {
		t.Errorf("object has wrong value. got %t, want=%t", result.Value, expected)
		return false
	}

	return true
}

func TestBangOperator(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"!true", false},
		{"!false", true},
		{"!5", false},
		{"!!true", true},
		{"!!false", false},
		{"!!5", true},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testBooleanObject(t, evaluated, tt.expected)
	}
}

func TestIfElseExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{"if {1} { 10 }", 10},
		{"if {0} { 10 }", nil},
		{"if {1} { 10 }", 10},
		{"if {1 < 2} { 10 }", 10},
		{"if {1 > 2} { 10 }", nil},
		{"if {1 > 2} { 10 } else { 20 }", 20},
		{"if {1 < 2} { 10 } else { 20 }", 10},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		integer, ok := tt.expected.(int)
		if ok {
			testNumberObject(t, evaluated, float64(integer))
		} else {
			testNullObject(t, evaluated)
		}
	}
}

func testNullObject(t *testing.T, obj object.Object) bool {
	if obj != NULL {
		t.Errorf("object is not NULL. got=%T (%+v)", obj, obj)
		return false
	}
	return true
}

func TestReturnStatements(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"return 10", 10},
		{"return 10; 9", 10},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testNumberObject(t, evaluated, tt.expected)
	}
}

func TestErrorHandling(t *testing.T) {
	tests := []struct {
		input           string
		expectedMessage string
		isParserError   bool
	}{
		{
			"5 + true;",
			"type mismatch: NUMBER + BOOLEAN",
			false,
		},
		{
			"5 + true; 5;",
			"type mismatch: NUMBER + BOOLEAN",
			false,
		},
		{
			"-true",
			"invalid command name '-true'",
			false,
		},
		{
			"if {1 + 1 == 2} {",
			"missing closing brace",
			true,
		},
		{
			"foobar",
			"identifier not found: foobar",
			false,
		},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := parser.New(l)
		program := p.ParseProgram()

		if tt.isParserError {
			errors := p.Errors()
			if len(errors) == 0 {
				t.Errorf("parser.ParseProgram() returned no errors, expected: %q", tt.expectedMessage)
				continue
			}
			if !strings.Contains(errors[0], tt.expectedMessage) {
				t.Errorf("wrong error message. expected=%q, got=%q", tt.expectedMessage, errors[0])
			}
		} else {
			// Check for unexpected parsing errors
			if len(p.Errors()) != 0 {
				t.Errorf("parser had %d errors", len(p.Errors()))
				for _, msg := range p.Errors() {
					t.Errorf("parser error: %q", msg)
				}
				continue
			}

			// Evaluate the program
			env := object.NewEnvironment()
			evaluated := Eval(program, env)

			// Check if the result is an error
			if errObj, ok := evaluated.(*object.Error); ok {
				if errObj.Message != tt.expectedMessage {
					t.Errorf("wrong error message. expected=%q, got=%q", tt.expectedMessage, errObj.Message)
				}
			} else {
				t.Errorf("no error object returned. got=%T(%+v)", evaluated, evaluated)
			}
		}
	}
}

func TestSetStatements(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		// {"set a 5; set result $a", 5},
		// {"set a 5; set b $a; set result $b", 5},
		{"set a [expr 5 * 5]; set result $a", 25},
		// {"set a 5; set b $a; set c [expr $a + $b + 5]; set result $c", 15},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		t.Logf("Evaluated result: %+v", evaluated) // Add this line
		testNumberObject(t, testEval(tt.input), tt.expected)
	}
}
