package parser

import (
	// "fmt"
	"github.com/elkrammer/irule-validator/ast"
	"github.com/elkrammer/irule-validator/lexer"
	"testing"
)

func checkParserErrors(t *testing.T, p *Parser) {
	errors := p.Errors()
	if len(errors) == 0 {
		return
	}

	t.Errorf("parser has %d errors", len(errors))

	for _, msg := range errors {
		t.Errorf("parser error: %q", msg)
	}
	t.FailNow()
}

func TestStringLiteralExpression(t *testing.T) {
	input := `"hello world";`

	l := lexer.New(input)
	p := New(l)

	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	literal, ok := stmt.Expression.(*ast.StringLiteral)
	if !ok {
		t.Fatalf("exp not *ast.StringLiteral. Got=%T", stmt.Expression)
	}

	if literal.Value != "hello world" {
		t.Errorf("literal.Value not %q. Got=%q", "hello world", literal.Value)
	}
}

// func TestLetStatements(t *testing.T) {
// 	input := `
//     let x = 5;
//     let y = 10;
//     let satan = 666;
//   `
//
// 	l := lexer.New(input)
// 	p := New(l)
//
// 	program := p.ParseProgram()
// 	checkParserErrors(t, p)
//
// 	if program == nil {
// 		t.Fatalf("ParseProgram() returned nil")
// 	}
//
// 	if len(program.Statements) != 3 {
// 		t.Fatalf("program.Statements does not contain 3 statements. Got: %d", len(program.Statements))
// 	}
//
// 	tests := []struct {
// 		expectedIdentifier string
// 	}{
// 		{"x"},
// 		{"y"},
// 		{"satan"},
// 	}
//
// 	for i, tt := range tests {
// 		stmt := program.Statements[i]
// 		if !testLetStatement(t, stmt, tt.expectedIdentifier) {
// 			return
// 		}
// 	}
// }
//
// func testLetStatement(t *testing.T, s ast.Statement, name string) bool {
// 	if s.TokenLiteral() != "let" {
// 		t.Errorf("s.TokenLiteral not 'let'. got=%q", s.TokenLiteral())
// 		return false
// 	}
//
// 	letStmt, ok := s.(*ast.LetStatement)
// 	if !ok {
// 		t.Errorf("s not *ast.LetStatement. Got=%T", s)
// 		return false
// 	}
//
// 	if letStmt.Name.Value != name {
// 		t.Errorf("letStmt.Name.Value nnot '%s'. Got=%s", name, letStmt.Name.Value)
// 		return false
// 	}
//
// 	if letStmt.Name.TokenLiteral() != name {
// 		t.Errorf("letStmt.Name.TokenLiteral() not '%s'. Got=%s", name, letStmt.TokenLiteral())
// 		return false
// 	}
//
// 	return true
// }

//	func TestReturnStatements(t *testing.T) {
//		input := `
//	    return 5;
//	    return 10;
//	    return 666;
//	  `
//
//		l := lexer.New(input)
//		p := New(l)
//
//		program := p.ParseProgram()
//		checkParserErrors(t, p)
//
//		if len(program.Statements) != 3 {
//			t.Fatalf("program.Statements does not contain 3 statements. Got=%d", len(program.Statements))
//		}
//
//		for _, stmt := range program.Statements {
//			returnStmt, ok := stmt.(*ast.ReturnStatement)
//			if !ok {
//				t.Errorf("stmt not *ast.ReturnStatement. Got=%T", stmt)
//				continue
//			}
//
//			if returnStmt.TokenLiteral() != "return" {
//				t.Errorf("returnStmt.TokenLiteral not 'return', got Got=%q", returnStmt.TokenLiteral())
//			}
//		}
//	}

// func TestIdentifierExpression(t *testing.T) {
// 	input := "foobar;"
//
// 	l := lexer.New(input)
// 	p := New(l)
// 	program := p.ParseProgram()
// 	checkParserErrors(t, p)
//
// 	if len(program.Statements) != 1 {
// 		t.Fatalf("program has not enough statements. got=%d", len(program.Statements))
// 	}
//
// 	stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
// 	if !ok {
// 		t.Fatalf("program.Statements[0] is not ast.ExpressionStatement. Got=%T", program.Statements[0])
// 	}
//
// 	ident, ok := stmt.Expression.(*ast.Identifier)
// 	if !ok {
// 		t.Fatalf("exp not *ast.Identifier. Got=%T", stmt.Expression)
// 	}
// 	if ident.Value != "foobar" {
// 		t.Errorf("ident.Value not %s. Got=%s", "foobar", ident.Value)
// 	}
// 	if ident.TokenLiteral() != "foobar" {
// 		t.Errorf("ident.TokenLiteral not %s. Got=%s", "foobar", ident.TokenLiteral())
// 	}
// }

// func TestIntegerLiteralExpression(t *testing.T) {
// 	input := "5;"
//
// 	l := lexer.New(input)
// 	p := New(l)
// 	program := p.ParseProgram()
// 	checkParserErrors(t, p)
//
// 	if len(program.Statements) != 1 {
// 		t.Fatalf("program has not enough statements. got=%d", len(program.Statements))
// 	}
//
// 	stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
// 	if !ok {
// 		t.Fatalf("program.Statements[0] is not ast.ExpressionStatement. Got=%T", program.Statements[0])
// 	}
//
// 	literal, ok := stmt.Expression.(*ast.NumberLiteral)
// 	if !ok {
// 		t.Fatalf("exp not *ast.NumberLiteral. Got=%T", stmt.Expression)
// 	}
// 	if literal.Value != 5.0 {
// 		t.Errorf("literal.Value not %f. Got=%f", 5.0, literal.Value)
// 	}
// 	if literal.TokenLiteral() != "5.0" {
// 		t.Errorf("ident.TokenLiteral not %s. Got=%s", "5", literal.TokenLiteral())
// 	}
// }

// func TestParsingPrefixExpressions(t *testing.T) {
// 	prefixTests := []struct {
// 		input    string
// 		operator string
// 		value    interface{}
// 	}{
// 		{"!5", "!", 5},
// 		{"-15", "-", 15},
// 		// {"!1", "!", 1}, // Tcl uses 1 for true
// 		// {"!0", "!", 0}, // Tcl uses 0 for false
// 		// {"! 1", "!", 1}, // Test whitespace tolerance
// 		// {"!   0", "!", 0}, // Test whitespace tolerance
// 		// {"!", "!", nil}, // Edge case: Only operator
// 		// {"", "", nil}, // Edge case: Empty input
// 		// Add more test cases for error handling if applicable
// 	}
//
// 	for _, tt := range prefixTests {
// 		l := lexer.New(tt.input)
// 		p := New(l)
// 		program := p.ParseProgram()
// 		checkParserErrors(t, p)
//
// 		if len(program.Statements) != 1 {
// 			t.Fatalf("program.Statements does not contain %d statements. Got=%d", 1, len(program.Statements))
// 		}
//
// 		stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
// 		if !ok {
// 			t.Fatalf("program.Statements[0] is not ast.ExpressionStatement. Got=%T", program.Statements[0])
// 		}
//
// 		exp, ok := stmt.Expression.(*ast.PrefixExpression)
// 		if !ok {
// 			t.Fatalf("stmt is not ast.PrefixExpression. Got=%T", stmt.Expression)
// 		}
// 		if exp.Operator != tt.operator {
// 			t.Fatalf("exp.Operator is not '%s'. Got=%s", tt.operator, exp.Operator)
// 		}
//
// 		if !testLiteralExpression(t, exp.Right, tt.value) {
// 			return
// 		}
// 	}
// }

// func testBooleanLiteral(t *testing.T, exp ast.Expression, value bool) bool {
// 	bo, ok := exp.(*ast.Boolean)
// 	if !ok {
// 		t.Errorf("exp not *ast.Boolean. got=%T", exp)
// 		return false
// 	}
//
// 	if bo.Value != value {
// 		t.Errorf("bo.Value not %t. Got=%t", value, bo.Value)
// 		return false
// 	}
//
// 	if bo.TokenLiteral() != fmt.Sprintf("%t", value) {
// 		t.Errorf("bo.TokenLiteral not %t. Got=%s", value, bo.TokenLiteral())
// 		return false
// 	}
//
// 	return true
// }

// func testNumberLiteral(t *testing.T, il ast.Expression, value float64) bool {
// 	num, ok := il.(*ast.NumberLiteral)
//
// 	if !ok {
// 		t.Errorf("il not *ast.IntegerLiteral. Got=%T", il)
// 		return false
// 	}
//
// 	if num.Value != value {
// 		t.Errorf("integ.Value not  %f. Got=%f", value, num.Value)
// 		return false
// 	}
//
// 	if num.TokenLiteral() != fmt.Sprintf("%f", value) {
// 		t.Errorf("integ.TokenLiteral not %f. Got=%s", value, num.TokenLiteral())
// 		return false
// 	}
//
// 	return true
// }

// func TestParsingInfixExpressions(t *testing.T) {
//
// 	infixTests := []struct {
// 		input      string
// 		leftValue  interface{}
// 		operator   string
// 		rightValue interface{}
// 	}{
// 		{"5 + 5;", 5, "+", 5},
// 		{"5 - 5;", 5, "-", 5},
// 		{"5 * 5;", 5, "*", 5},
// 		{"5 / 5;", 5, "/", 5},
// 		{"5 > 5;", 5, ">", 5},
// 		{"5 < 5;", 5, "<", 5},
// 		{"5 == 5;", 5, "==", 5},
// 		{"5 != 5;", 5, "!=", 5},
// 		{"true == true", true, "==", true},
// 		{"true != false", true, "!=", false},
// 		{"false == false", false, "==", false},
// 	}
//
// 	for _, tt := range infixTests {
// 		l := lexer.New(tt.input)
// 		p := New(l)
// 		program := p.ParseProgram()
// 		checkParserErrors(t, p)
//
// 		if len(program.Statements) != 1 {
// 			t.Fatalf("program.Statements does not contain %d statements. Got=%d", 1, len(program.Statements))
// 		}
//
// 		stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
// 		if !ok {
// 			t.Fatalf("program.Statements[0] is not an ast.ExpressionStatement. Got=%T", program.Statements[0])
// 		}
//
// 		if !testInfixExpression(t, stmt.Expression, tt.leftValue, tt.operator, tt.rightValue) {
// 			return
// 		}
// 	}
// }

func testIdentifier(t *testing.T, exp ast.Expression, value string) bool {
	ident, ok := exp.(*ast.Identifier)
	if !ok {
		t.Errorf("exp is not *ast.Identifier. Got=%T", exp)
		return false
	}

	// Construct the expected token literal including the '$' symbol
	expectedTokenLiteral := "$" + value

	// Expected value should not include the '$' symbol
	if ident.Value != value {
		t.Errorf("ident.Value not %s. Got=%s", value, ident.Value)
		return false
	}

	if ident.TokenLiteral() != expectedTokenLiteral {
		t.Errorf("ident.TokenLiteral not %s. Got=%s", value, ident.TokenLiteral())
		return false
	}

	return true
}

func testLiteralExpression(t *testing.T, exp ast.Expression, expected interface{}) bool {
	switch v := expected.(type) {
	// case int:
	// 	// return testNumberLiteral(t, exp, float64(v))
	// case float64:
	// 	return testNumberLiteral(t, exp, v)
	case string:
		return testIdentifier(t, exp, v)
		// case bool:
		// 	return testBooleanLiteral(t, exp, v)
	}

	t.Errorf("type of exp not handled. Got=%T", exp)
	return false
}

// func TestBooleanExpression(t *testing.T) {
// 	tests := []struct {
// 		input           string
// 		expectedBoolean bool
// 	}{
// 		{"true;", true},
// 		{"false;", false},
// 	}
//
// 	for _, tt := range tests {
// 		l := lexer.New(tt.input)
// 		p := New(l)
// 		program := p.ParseProgram()
// 		checkParserErrors(t, p)
//
// 		if len(program.Statements) != 1 {
// 			t.Fatalf("program has not enough statements. got=%d",
// 				len(program.Statements))
// 		}
//
// 		stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
// 		if !ok {
// 			t.Fatalf("program.Statements[0] is not ast.ExpressionStatement. got=%T",
// 				program.Statements[0])
// 		}
//
// 		boolean, ok := stmt.Expression.(*ast.Boolean)
// 		if !ok {
// 			t.Fatalf("exp not *ast.Boolean. got=%T", stmt.Expression)
// 		}
// 		if boolean.Value != tt.expectedBoolean {
// 			t.Errorf("boolean.Value not %t. got=%t", tt.expectedBoolean,
// 				boolean.Value)
// 		}
// 	}
// }

func testInfixExpression(t *testing.T, exp ast.Expression, left interface{}, operator string, right interface{}) bool {
	opExp, ok := exp.(*ast.InfixExpression)
	if !ok {
		t.Errorf("exp is not ast.InfixExpression. Got=%T(%s)", exp, exp)
		return false
	}

	if !testLiteralExpression(t, opExp.Left, left) {
		return false
	}

	if opExp.Operator != operator {
		t.Errorf("exp.Operator is not '%s'. Got=%q", operator, opExp.Operator)
		return false
	}

	if !testLiteralExpression(t, opExp.Right, right) {
		return false
	}

	return true
}

func TestOperatorPrecedenceParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			"(1 + 2) * 3",
			"1 + 2 * 3",
		},
		{
			"1 + 2 * 3",
			"1 + 2 * 3",
		},
		// {
		// 	"2 * (3 + 4) - 5 / 2",
		// 	"2 * (3 + 4) - 5 / 2",
		// },
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		actual := program.String()
		if actual != tt.expected {
			t.Errorf("expected=%q, got=%q", tt.expected, actual)
		}
	}
}

func TestIfExpression(t *testing.T) {
	input := `if {$x < $y} { x }`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Body does not contain %d statements. got=%d\n", 1, len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not ast.ExpressionStatement. got=%T", program.Statements[0])
	}

	exp, ok := stmt.Expression.(*ast.IfExpression)
	if !ok {
		t.Fatalf("stmt.Expression is not ast.IfExpression. got=%T", stmt.Expression)
	}

	if !testInfixExpression(t, exp.Condition, "x", "<", "y") {
		return
	}

	if len(exp.Consequence.Statements) != 1 {
		t.Errorf("consequence is not 1 statements. got=%d\n", len(exp.Consequence.Statements))
	}

	consequence, ok := exp.Consequence.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("Statements[0] is not ast.ExpressionStatement. got=%T", exp.Consequence.Statements[0])
	}

	if !testIdentifier(t, consequence.Expression, "x") {
		return
	}

	if exp.Alternative != nil {
		t.Errorf("exp.Alternative.Statements was not nil. got=%+v", exp.Alternative)
	}
}

func TestIfElseExpression(t *testing.T) {
	input := `if {$x < $y} { x } else { y }`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain %d statements. got=%d\n",
			1, len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not ast.ExpressionStatement. got=%T",
			program.Statements[0])
	}

	exp, ok := stmt.Expression.(*ast.IfExpression)
	if !ok {
		t.Fatalf("stmt.Expression is not ast.IfExpression. got=%T", stmt.Expression)
	}

	if !testInfixExpression(t, exp.Condition, "x", "<", "y") {
		return
	}

	if len(exp.Consequence.Statements) != 1 {
		t.Errorf("consequence is not 1 statements. got=%d\n",
			len(exp.Consequence.Statements))
	}

	consequence, ok := exp.Consequence.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("Statements[0] is not ast.ExpressionStatement. got=%T",
			exp.Consequence.Statements[0])
	}

	if !testIdentifier(t, consequence.Expression, "x") {
		return
	}

	if len(exp.Alternative.Statements) != 1 {
		t.Errorf("exp.Alternative.Statements does not contain 1 statements. got=%d\n",
			len(exp.Alternative.Statements))
	}

	alternative, ok := exp.Alternative.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("Statements[0] is not ast.ExpressionStatement. got=%T",
			exp.Alternative.Statements[0])
	}

	if !testIdentifier(t, alternative.Expression, "y") {
		return
	}
}

func TestFunctionLiteralParsing(t *testing.T) {
	input := "proc add {x, y} { return $x + $y }"
	expectedParams := []string{"x", "y"}

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	proc := stmt.Expression.(*ast.FunctionLiteral)

	if len(proc.Parameters) != len(expectedParams) {
		t.Fatalf("length of parameters wrong. want %d, got %d\n",
			len(expectedParams), len(proc.Parameters))
	}

	for i, ident := range expectedParams {
		if proc.Parameters[i].Value != ident {
			t.Errorf("parameter %d wrong. want %s, got %s\n",
				i, ident, proc.Parameters[i].Value)
		}
	}
}

func TestFunctionParameterParsing(t *testing.T) {
	tests := []struct {
		input          string
		expectedParams []string
	}{
		// {input: "proc add {} {}", expectedParams: []string{}},
		// {input: "proc add {x} {}", expectedParams: []string{"x"}},
		// {input: "proc add {x y z} {}", expectedParams: []string{"x", "y", "z"}},
		{"proc add {x y z} {}", []string{"x", "y", "z"}},
		{"proc add {\"x\" \"y\" \"z\"} {}", []string{"x", "y", "z"}},
		{"proc add {x, y, z} {}", []string{"x", "y", "z"}},
		{"proc add {\"x\", \"y\", \"z\"} {}", []string{"x", "y", "z"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			program := p.ParseProgram()
			checkParserErrors(t, p)

			stmt := program.Statements[0].(*ast.ExpressionStatement)
			proc := stmt.Expression.(*ast.FunctionLiteral)

			if len(proc.Parameters) != len(tt.expectedParams) {
				t.Fatalf("length of parameters wrong. want %d, got %d\n",
					len(tt.expectedParams), len(proc.Parameters))
			}

			for i, ident := range tt.expectedParams {
				if proc.Parameters[i].Value != ident {
					t.Errorf("parameter %d wrong. want %s, got %s\n",
						i, ident, proc.Parameters[i].Value)
				}
			}
		})
	}
}
