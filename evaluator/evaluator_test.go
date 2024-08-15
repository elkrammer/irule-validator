package evaluator

import (
	"github.com/elkrammer/irule-validator/lexer"
	"github.com/elkrammer/irule-validator/object"
	"github.com/elkrammer/irule-validator/parser"

	"fmt"
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
		{
			`"hello" - "world"`,
			"unknown operator: STRING - STRING",
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
		expected interface{}
	}{
		{"set a 5; set result $a", 5.0},
		{"set a 5; set b $a; set result $b", 5.0},
		{"set a [expr 5 * 5]; set result $a", 25.0},
		{"set a 5; set b $a; set c [expr $a + $b + 5]; set result $c", 15.0},
	}
	for _, tt := range tests {
		evaluated := testEval(tt.input)
		t.Logf("Input: %s", tt.input)
		t.Logf("Evaluated result: %+v", evaluated)

		switch expected := tt.expected.(type) {
		case float64:
			testNumberObject(t, evaluated, expected)
		default:
			t.Errorf("unexpected result type. got=%T, want=%T", evaluated, expected)
		}
	}
}

func TestFunctionObject(t *testing.T) {
	input := "proc add {x} { $x + 2 }"
	evaluated := testEval(input)

	fn, ok := evaluated.(*object.Function)
	if !ok {
		t.Fatalf("object is not Function. got=%T (%+v)", evaluated, evaluated)
	}

	if len(fn.Parameters) != 1 {
		t.Fatalf("function has wrong parameters. Parameters=%+v", fn.Parameters)
	}

	if fn.Parameters[0].String() != "x" {
		t.Fatalf("parameter is not 'x'. got=%q", fn.Parameters[0])
	}

	expectedBody := "$x + 2"
	if fn.Body == nil {
		t.Fatalf("function body is nil")
	}
	if fn.Body.String() != expectedBody {
		t.Fatalf("body is not %q. got=%q", expectedBody, fn.Body.String())
	}
}

func TestFunctionApplication(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"proc identity {x} {return $x}; identity 5;", 5},
		{"proc double {x} {expr {$x * 2}}; double 5;", 10},
		{"proc anon {x} {return $x}; anon 5", 5},
		// {"proc add {x y} {expr {$x + $y}}; add 5 5;", 10},
		// {"proc add {x y} {expr {$x + $y}}; add [expr {5 + 5}] [add 5 5];", 20},
	}

	for _, tt := range tests {
		testNumberObject(t, testEval(tt.input), tt.expected)
	}
}

func TestStringLiteral(t *testing.T) {
	input := `"howdy!"`

	evaluated := testEval(input)
	str, ok := evaluated.(*object.String)
	if !ok {
		t.Fatalf("object is not string. got=%T (%+v)", evaluated, evaluated)
	}

	if str.Value != "howdy!" {
		t.Errorf("String has wrong value. got=%q", str.Value)
	}
}

func TestStringConcatenation(t *testing.T) {
	input := `"Hello" + " " + "World!"`

	evaluated := testEval(input)
	str, ok := evaluated.(*object.String)
	if !ok {
		t.Fatalf("object is not String. got=%T (%+v)", evaluated, evaluated)
	}

	if str.Value != "Hello World!" {
		t.Errorf("String has wrong value. got=%q", str.Value)
	}
}

func TestBuiltinFunctions(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{`puts("hello", "world!")`, nil},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)

		switch expected := tt.expected.(type) {
		case int:
			testNumberObject(t, evaluated, float64(expected))
		case nil:
			testNullObject(t, evaluated)
		case string:
			errObj, ok := evaluated.(*object.Error)
			if !ok {
				t.Errorf("object is not Error. got=%T (%+v)",
					evaluated, evaluated)
				continue
			}
			if errObj.Message != expected {
				t.Errorf("wrong error message. expected=%q, got=%q",
					expected, errObj.Message)
			}
		case []int:
			array, ok := evaluated.(*object.Array)
			if !ok {
				t.Errorf("obj not Array. got=%T (%+v)", evaluated, evaluated)
				continue
			}

			if len(array.Elements) != len(expected) {
				t.Errorf("wrong num of elements. want=%d, got=%d",
					len(expected), len(array.Elements))
				continue
			}

			for i, expectedElem := range expected {
				testNumberObject(t, array.Elements[i], float64(expectedElem))
			}
		}
	}
}

func TestArrayExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"set arr(0) 10; set arr(1) 20; return $arr(0)", "10"},
		// {"array set arr {0 30 1 40}; return $arr(1)", "40"},
		// {"set arr(foo) bar; return $arr(foo)", "bar"},
		// {"array set arr {a 1 b 2 c 3}; return [array size arr]", "3"},
		// {"array set arr {0 10 1 20 2 30}; return [array names arr]", "0 1 2"},
	}

	for _, tt := range tests {
		evaluated := testEval(tt.input)
		testStringObject(t, evaluated, tt.expected)
	}
}

func testStringObject(t *testing.T, obj object.Object, expected string) {
	// Check if the object is a Number and convert it to a String if true
	if numObj, ok := obj.(*object.Number); ok {
		// Convert the number to a string
		obj = &object.String{Value: fmt.Sprintf("%d", int(numObj.Value))}
	}

	// Now proceed with the original String test
	result, ok := obj.(*object.String)
	if !ok {
		t.Fatalf("object is not String. got=%T (%+v)", obj, obj)
	}

	if result.Value != expected {
		t.Errorf("object has wrong value. got=%q, want=%q",
			result.Value, expected)
	}
}
