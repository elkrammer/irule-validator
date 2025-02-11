package parser

import (
	"fmt"
	"github.com/elkrammer/irule-validator/ast"
	"github.com/elkrammer/irule-validator/lexer"
	"strings"
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

func TestSetStatements(t *testing.T) {
	tests := []struct {
		input              string
		expectedIdentifier string
		expectedValue      interface{}
		isHttpUri          bool
	}{
		{"set x 5", "x", 5, false},
		{"set y true", "y", true, false},
		{"set foobar y", "foobar", "y", false},
	}

	for i, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != 1 {
			t.Fatalf("program.Statements does not contain 1 statements. got=%d",
				len(program.Statements))
		}

		stmt := program.Statements[0]
		if tt.isHttpUri {
			if !testSetStatementWithHttpUri(t, stmt, tt.expectedIdentifier, tt.expectedValue.(string)) {
				t.Errorf("Test case %d failed", i)
				return
			}
		} else {
			if !testSetStatement(t, stmt, tt.expectedIdentifier) {
				t.Errorf("Test case %d failed", i)
				return
			}

			val := stmt.(*ast.SetStatement).Value
			if !testLiteralExpression(t, val, tt.expectedValue) {
				t.Errorf("Test case %d failed", i)
				return
			}
		}
	}
}

func testSetStatement(t *testing.T, s ast.Statement, name string) bool {
	if s.TokenLiteral() != "set" {
		t.Errorf("s.TokenLiteral not 'set'. got=%q", s.TokenLiteral())
		return false
	}

	setStmt, ok := s.(*ast.SetStatement)
	if !ok {
		t.Errorf("s not *ast.SetStatement. got=%T", s)
		return false
	}

	return testComplexExpression(t, setStmt.Name, name)
}

func testBooleanLiteral(t *testing.T, exp ast.Expression, value bool) bool {
	bo, ok := exp.(*ast.Boolean)
	if !ok {
		t.Errorf("exp not *ast.BooleanLiteral. got=%T", exp)
		return false
	}

	if bo.Value != value {
		t.Errorf("bo.Value not %t. got=%t", value, bo.Value)
		return false
	}

	if bo.TokenLiteral() != fmt.Sprintf("%t", value) {
		t.Errorf("bo.TokenLiteral not %t. got=%s", value, bo.TokenLiteral())
		return false
	}

	return true
}

func TestReturnStatements(t *testing.T) {
	input := `
    when HTTP_REQUEST {
      if {[HTTP::uri] contains "forbidden"} {
        return 403
      }
    }
  `
	l := lexer.New(input)
	p := New(l)

	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. Got=%d", len(program.Statements))
	}

	whenStmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not ast.ExpressionStatement. Got=%T", program.Statements[0])
	}

	whenExp, ok := whenStmt.Expression.(*ast.WhenExpression)
	if !ok {
		t.Fatalf("stmt.Expression is not ast.WhenExpression. Got=%T", whenStmt.Expression)
	}

	if len(whenExp.Block.Statements) != 1 {
		t.Fatalf("when block does not contain 1 statement. Got=%d", len(whenExp.Block.Statements))
	}

	ifStmt, ok := whenExp.Block.Statements[0].(*ast.IfStatement)
	if !ok {
		t.Fatalf("when block statement is not ast.IfStatement. Got=%T", whenExp.Block.Statements[0])
	}

	if len(ifStmt.Consequence.Statements) != 1 {
		t.Fatalf("if consequence does not contain 1 statement. Got=%d", len(ifStmt.Consequence.Statements))
	}

	returnStmt, ok := ifStmt.Consequence.Statements[0].(*ast.ReturnStatement)
	if !ok {
		t.Fatalf("if consequence statement is not ast.ReturnStatement. Got=%T", ifStmt.Consequence.Statements[0])
	}

	if returnStmt.TokenLiteral() != "return" {
		t.Errorf("returnStmt.TokenLiteral not 'return', got=%q", returnStmt.TokenLiteral())
	}
}

func TestParsingPrefixExpressions(t *testing.T) {
	prefixTests := []struct {
		input    string
		operator string
		value    interface{}
	}{
		{"! 5", "!", 5},   // Boolean negation with whitespace
		{"-0", "-", 0},    // Boolean negation with 0 (false)
		{"!1", "!", 1},    // Boolean negation with 1 (true)
		{"- 15", "-", 15}, // Arithmetic negation with whitespace
	}

	for _, tt := range prefixTests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != 1 {
			t.Fatalf("program.Statements does not contain %d statements. Got=%d", 1, len(program.Statements))
		}

		stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
		if !ok {
			t.Fatalf("program.Statements[0] is not ast.ExpressionStatement. Got=%T", program.Statements[0])
		}

		exp, ok := stmt.Expression.(*ast.PrefixExpression)
		if !ok {
			t.Fatalf("stmt is not ast.PrefixExpression. Got=%T", stmt.Expression)
		}
		if exp.Operator != tt.operator {
			t.Fatalf("exp.Operator is not '%s'. Got=%s", tt.operator, exp.Operator)
		}

		if !testLiteralExpression(t, exp.Right, tt.value) {
			return
		}
	}
}

func testIdentifier(t *testing.T, ident *ast.Identifier, expectedValue string) bool {
	if ident.Value != expectedValue {
		t.Errorf("ident.Value not %s. got=%s", expectedValue, ident.Value)
		return false
	}
	return true
}

func testLiteralExpression(t *testing.T, exp ast.Expression, expected interface{}) bool {
	switch v := expected.(type) {
	case int:
		return testNumberLiteral(t, exp, int64(v))
	case int64:
		return testNumberLiteral(t, exp, v)
	case string:
		return testStringOrIdentifierLiteral(t, exp, v)
	case bool:
		return testBooleanLiteral(t, exp, v)
	}

	t.Errorf("type of exp not handled. Got=%T", exp)
	return false
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
		{
			"2 * (3 + 4) - 5 / 2",
			"2 * (3 + 4) - 5 / 2",
		},
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

func testNumberLiteral(t *testing.T, nl ast.Expression, value int64) bool {
	num, ok := nl.(*ast.NumberLiteral)
	if !ok {
		t.Errorf("nl not *ast.NumberLiteral. got=%T", nl)
		return false
	}

	if num.Value != value {
		t.Errorf("num.Value not %d. got=%d", value, num.Value)
		return false
	}

	if num.TokenLiteral() != fmt.Sprintf("%d", value) {
		t.Errorf("num.TokenLiteral not %d. got=%s", value, num.TokenLiteral())
		return false
	}

	return true
}

func testStringOrIdentifierLiteral(t *testing.T, exp ast.Expression, value string) bool {
	switch v := exp.(type) {
	case *ast.ArrayLiteral:
		if len(v.Elements) == 0 {
			t.Errorf("ArrayLiteral does not contain at least 1 element. got=%d", len(v.Elements))
			return false
		}
		switch elem := v.Elements[0].(type) {
		case *ast.HttpExpression:
			if elem.String() != value {
				t.Errorf("HttpExpression.String() not %q. got=%q", value, elem.String())
				return false
			}
		case *ast.IpExpression:
			if elem.String() != value {
				t.Errorf("IpExpression.String() not %q. got=%q", value, elem.String())
				return false
			}
		default:
			t.Errorf("ArrayLiteral element is not HttpExpression or IpExpression. got=%T", elem)
			return false
		}
	}
	return true
}

func testComplexExpression(t *testing.T, exp ast.Expression, expectedName string) bool {
	switch target := exp.(type) {
	case *ast.Identifier:
		return testIdentifier(t, target, expectedName)
	case *ast.BracketExpression:
		return testComplexExpression(t, target.Expression, expectedName)
	case *ast.InfixExpression:
		return testInfixExpression(t, target, expectedName)
	case *ast.ArrayLiteral:
		return testArrayLiteral(t, target, expectedName)
	default:
		t.Errorf("Expression is not Identifier, BracketExpression, or InfixExpression. got=%T", exp)
		return false
	}
}

func testInfixExpression(t *testing.T, exp *ast.InfixExpression, expectedName string) bool {
	// Handle different types of infix expressions
	switch exp.Operator {
	case "[":
		return testComplexExpression(t, exp.Left, expectedName)
	case "/":
		// This could be a path or part of an HTTP::uri expression
		if arrayLit, ok := exp.Left.(*ast.ArrayLiteral); ok {
			return testArrayLiteral(t, arrayLit, expectedName)
		}
		// Otherwise, it's likely a path
		return testPath(t, exp, expectedName)
	default:
		t.Errorf("Unexpected operator in InfixExpression: %s", exp.Operator)
		return false
	}
}

func testArrayLiteral(t *testing.T, arr *ast.ArrayLiteral, expectedName string) bool {
	if len(arr.Elements) != 1 {
		t.Errorf("ArrayLiteral does not contain 1 element. got=%d", len(arr.Elements))
		return false
	}

	switch elem := arr.Elements[0].(type) {
	case *ast.HttpExpression:
		return testIdentifier(t, elem.Command, expectedName)
	case *ast.Identifier:
		return testIdentifier(t, elem, expectedName)
	default:
		t.Errorf("Unexpected type in ArrayLiteral: %T", elem)
		return false
	}
}

func testPath(t *testing.T, exp *ast.InfixExpression, expectedPath string) bool {
	if exp.Operator != "/" {
		t.Errorf("Operator is not '/'. got=%s", exp.Operator)
		return false
	}

	left, leftOk := exp.Left.(*ast.Identifier)
	right, rightOk := exp.Right.(*ast.Identifier)

	if !leftOk || !rightOk {
		t.Errorf("Left or Right of InfixExpression is not an Identifier. Left: %T, Right: %T", exp.Left, exp.Right)
		return false
	}

	actualPath := left.Value + "/" + right.Value
	if actualPath != expectedPath {
		t.Errorf("Path not %s. got=%s", expectedPath, actualPath)
		return false
	}

	return true
}

func testSetStatementWithHttpUri(t *testing.T, s ast.Statement, expectedName string, expectedPath string) bool {
	setStmt, ok := s.(*ast.SetStatement)
	if !ok {
		t.Errorf("s not *ast.SetStatement. got=%T", s)
		return false
	}

	outerInfix, ok := setStmt.Name.(*ast.InfixExpression)
	if !ok {
		t.Errorf("setStmt.Name not *ast.InfixExpression. got=%T", setStmt.Name)
		return false
	}
	// fmt.Printf("DEBUG: Outer InfixExpression: %+v\n", outerInfix)

	if outerInfix.Operator != "/" {
		t.Errorf("Outer operator is not '/'. got=%s", outerInfix.Operator)
		return false
	}

	// Check the left side (should be [[HTTP::uri]])
	leftInfix, ok := outerInfix.Left.(*ast.InfixExpression)
	if !ok {
		t.Errorf("Left of outer InfixExpression is not an InfixExpression. got=%T", outerInfix.Left)
		return false
	}
	// fmt.Printf("DEBUG: Left InfixExpression: %+v\n", leftInfix)

	if leftInfix.Operator != "/" {
		t.Errorf("Left inner operator is not '/'. got=%s", leftInfix.Operator)
		return false
	}

	arrayLit, ok := leftInfix.Left.(*ast.ArrayLiteral)
	if !ok {
		t.Errorf("Left of left inner InfixExpression is not an ArrayLiteral. got=%T", leftInfix.Left)
		return false
	}
	// fmt.Printf("DEBUG: ArrayLiteral: %+v\n", arrayLit)

	if len(arrayLit.Elements) != 1 {
		t.Errorf("ArrayLiteral does not contain 1 element. got=%d", len(arrayLit.Elements))
		return false
	}

	httpExp, ok := arrayLit.Elements[0].(*ast.HttpExpression)
	if !ok {
		t.Errorf("Element is not an HttpExpression. got=%T", arrayLit.Elements[0])
		return false
	}
	// fmt.Printf("DEBUG: HttpExpression: %+v\n", httpExp)

	if httpExp.Command.Value != expectedName {
		t.Errorf("HttpExpression.Command.Value not '%s'. got=%s", expectedName, httpExp.Command.Value)
		return false
	}

	// Check the path
	path1, ok := leftInfix.Right.(*ast.Identifier)
	if !ok {
		t.Errorf("Right of left InfixExpression is not an Identifier. got=%T", leftInfix.Right)
		return false
	}

	path2, ok := outerInfix.Right.(*ast.Identifier)
	if !ok {
		t.Errorf("Right of outer InfixExpression is not an Identifier. got=%T", outerInfix.Right)
		return false
	}

	actualPath := path1.Value + "/" + path2.Value
	// fmt.Printf("DEBUG: Actual Path: %s\n", actualPath)

	if actualPath != expectedPath {
		t.Errorf("Path not '%s'. got=%s", expectedPath, actualPath)
		return false
	}

	return true
}

func TestInfixExpressions(t *testing.T) {
	tests := []struct {
		input      string
		leftValue  interface{}
		operator   string
		rightValue interface{}
	}{
		{"5 + 5;", 5, "+", 5},
		{"5 - 5;", 5, "-", 5},
		{"5 * 5;", 5, "*", 5},
		{"5 / 5;", 5, "/", 5},
		{"5 > 5;", 5, ">", 5},
		{"5 < 5;", 5, "<", 5},
		{"5 == 5;", 5, "==", 5},
		{"5 != 5;", 5, "!=", 5},
		{"true == true", true, "==", true},
		{"true != false", true, "!=", false},
		{"false == false", false, "==", false},
		{"[HTTP::uri] contains \"admin\"", "[HTTP::uri]", "contains", "admin"},
		{"[IP::client_addr] equals 10.0.0.1", "IP::client_addr", "equals", "10.0.0.1"},
		{"[HTTP::header \"User-Agent\"] starts_with \"Mozilla\"", "[HTTP::header]", "starts_with", "Mozilla"},
		// {"[TCP::local_port] != 443", "TCP::local_port", "!=", 443},
		// {"$current_users <= $max_users", "$current_users", "<=", "$max_users"},
		// {"$static::max_connections > 100", "$static::max_connections", ">", 100},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
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

		exp, ok := stmt.Expression.(*ast.InfixExpression)
		if !ok {
			t.Fatalf("stmt.Expression is not ast.InfixExpression. got=%T",
				stmt.Expression)
		}

		if !testInfixExpressionComponents(t, exp, tt.leftValue, tt.operator, tt.rightValue) {
			return
		}
	}
}

func testInfixExpressionComponents(t *testing.T, exp *ast.InfixExpression, left interface{}, operator string, right interface{}) bool {
	if !testLiteralExpression(t, exp.Left, left) {
		return false
	}

	if exp.Operator != operator {
		t.Errorf("exp.Operator is not '%s'. got=%q", operator, exp.Operator)
		return false
	}

	if !testLiteralExpression(t, exp.Right, right) {
		return false
	}

	return true
}

func TestF5IRuleConstructs(t *testing.T) {
	tests := []struct {
		input              string
		expectedStatements int
		checkFunc          func(*testing.T, ast.Statement)
	}{
		{
			input:              "when HTTP_REQUEST { }",
			expectedStatements: 1,
			checkFunc:          checkWhenExpression,
		},
		{
			input:              "HTTP::respond 200 content \"Hello, World!\"",
			expectedStatements: 4,
			checkFunc:          checkHttpRespond,
		},
		{
			input:              "pool my_pool",
			expectedStatements: 1,
			checkFunc:          checkPoolCommand,
		},
		{
			input: `
		              if { [HTTP::uri] starts_with "/api" } {
		                  pool api_pool
		              } else {
		                  pool default_pool
		              }
		          `,
			expectedStatements: 1,
			checkFunc:          checkIfStatement,
		},
		{
			input: `
		              switch -glob [HTTP::uri] {
		                  "/images/*" { pool image_pool }
		                  "/videos/*" { pool video_pool }
		                  default { pool default_pool }
		              }
		          `,
			expectedStatements: 1,
			checkFunc:          checkSwitchStatement,
		},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
		program := p.ParseProgram()
		checkParserErrors(t, p)

		if len(program.Statements) != tt.expectedStatements {
			t.Fatalf("program has wrong number of statements. got=%d, want=%d",
				len(program.Statements), tt.expectedStatements)
		}

		tt.checkFunc(t, program.Statements[0])
	}
}

func checkWhenExpression(t *testing.T, stmt ast.Statement) {
	exprStmt, ok := stmt.(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("stmt not *ast.ExpressionStatement. got=%T", stmt)
	}

	whenExpr, ok := exprStmt.Expression.(*ast.WhenExpression)
	if !ok {
		t.Fatalf("exprStmt.Expression not *ast.WhenExpression. got=%T", exprStmt.Expression)
	}

	if whenExpr.TokenLiteral() != "when" {
		t.Errorf("whenExpr.TokenLiteral not 'when'. got=%q", whenExpr.TokenLiteral())
	}

	eventIdent, ok := whenExpr.Event.(*ast.Identifier)
	if !ok {
		t.Fatalf("whenExpr.Event not *ast.Identifier. got=%T", whenExpr.Event)
	}

	if eventIdent.Value != "HTTP_REQUEST" {
		t.Errorf("eventIdent.Value not 'HTTP_REQUEST'. got=%q", eventIdent.Value)
	}

	if whenExpr.Block == nil {
		t.Fatalf("whenExpr.Block is nil")
	}
}

func checkHttpRespond(t *testing.T, stmt ast.Statement) {
	exprStmt, ok := stmt.(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("stmt not *ast.ExpressionStatement. got=%T", stmt)
	}

	expr, ok := exprStmt.Expression.(*ast.HttpExpression)
	if !ok {
		t.Fatalf("stmt.Expression not *ast.HttpExpression. got=%T", exprStmt.Expression)
	}

	if expr.Command.String() != "HTTP::respond" {
		t.Errorf("callExpr.Function not 'HTTP::respond'. got=%q", expr.Token.Literal)
	}
}

func checkPoolCommand(t *testing.T, stmt ast.Statement) {
	exprStmt, ok := stmt.(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("stmt not *ast.ExpressionStatement. got=%T", stmt)
	}

	callExpr, ok := exprStmt.Expression.(*ast.CallExpression)
	if !ok {
		t.Fatalf("stmt.Expression not *ast.CallExpression. got=%T", exprStmt.Expression)
	}

	if callExpr.Function.String() != "pool" {
		t.Errorf("callExpr.Function not 'pool'. got=%q", callExpr.Function)
	}

	if len(callExpr.Arguments) != 1 {
		t.Fatalf("wrong number of arguments. got=%d, want=1", len(callExpr.Arguments))
	}
}

func checkIfStatement(t *testing.T, stmt ast.Statement) {
	ifStmt, ok := stmt.(*ast.IfStatement)
	if !ok {
		t.Fatalf("stmt not *ast.IfStatement. got=%T", stmt)
	}

	if ifStmt.Condition == nil {
		t.Fatalf("ifStmt.Condition is nil")
	}

	if ifStmt.Consequence == nil {
		t.Fatalf("ifStmt.Consequence is nil")
	}

	if ifStmt.Alternative == nil {
		t.Fatalf("ifStmt.Alternative is nil")
	}
}

func checkSwitchStatement(t *testing.T, stmt ast.Statement) {
	switchStmt, ok := stmt.(*ast.SwitchStatement)
	if !ok {
		t.Fatalf("stmt not *ast.ExpressionStatement. got=%T", stmt)
	}

	if switchStmt.Value == nil {
		t.Fatalf("switchStmt.Value is nil")
	}

	if len(switchStmt.Cases) < 2 {
		fmt.Printf("switchStmt = %+v", switchStmt)
		t.Fatalf("switchExpr has too few cases. got=%d, want at least 2", len(switchStmt.Cases))
	}

	if switchStmt.Default == nil {
		t.Fatalf("switchExpr.Default is nil")
	}
}

func TestSwitchStatementPatternValidation(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedErrors []string
	}{
		{
			name: "Valid regex patterns",
			input: `
		      when HTTP_REQUEST {
		          switch -regex [string tolower [HTTP::uri]] {
		              "^/api/v1/users/\d+$" { }
                  "^/api/v2/(?:users|groups)/[a-zA-Z0-9-]+$" { }
		              "^/search\?q=([^&]+)&page=(\d+)$" { }
		              "^/products/([a-z0-9-]+)(?:/reviews)?(?:\?sort=(asc|desc))?$" { }
		              "^/files/([^/]+\.(?:pdf|docx?|xlsx?))$" { }
		              "^/(\w+)/(\d+)(?:/(\w+))?(?:/history)?$" { }
                  "^/reports/(daily|weekly|monthly)/(\d{4}-\d{2}-\d{2})$}" { }
		              default { }
		          }
		      }
		  `,
			expectedErrors: []string{},
		},
		{
			name: "Invalid glob pattern in regex switch",
			input: `
				when HTTP_REQUEST {
					switch -regex [string tolower [HTTP::uri]] {
						"^/api/v1/users.*" { }
						"/api*" { }
						default { }
					}
				}
			`,
			expectedErrors: []string{"   Invalid regex pattern (looks like a glob pattern): /api*"},
		},
		{
			name: "Invalid regex pattern in glob switch",
			input: `
				when HTTP_REQUEST {
					switch -glob [string tolower [HTTP::uri]] {
						"/api*" - "/api" { }
						"^/api/v1/users.*" { }
						default { }
					}
				}
			`,
			expectedErrors: []string{"   Invalid glob pattern (looks like a regex pattern): ^/api/v1/users.*"},
		},
		{
			name: "Mixed invalid patterns",
			input: `
				when HTTP_REQUEST {
					switch -regex [string tolower [HTTP::uri]] {
						"^/api/v1/users.*" { }
						"/api*" { }
					}
				}
			`,
			expectedErrors: []string{
				"   Invalid regex pattern (looks like a glob pattern): /api*",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input)
			p := New(l)
			program := p.ParseProgram()

			errors := p.Errors()
			if len(errors) != len(tt.expectedErrors) {
				t.Fatalf("Expected %d errors, got %d", len(tt.expectedErrors), len(errors))
			}

			for i, expectedError := range tt.expectedErrors {
				if !strings.Contains(errors[i], expectedError) {
					t.Errorf("Expected error to contain: %q, got: %q", expectedError, errors[i])
				}
			}

			if len(errors) > 0 {
				return
			}

			if program == nil || len(program.Statements) == 0 {
				t.Fatalf("ParseProgram() returned nil or empty program")
			}

			whenExpr, ok := program.Statements[0].(*ast.ExpressionStatement)
			if !ok {
				t.Fatalf("First statement is not ast.ExpressionStatement. got=%T", program.Statements[0])
			}

			whenStmt, ok := whenExpr.Expression.(*ast.WhenExpression)
			if !ok {
				t.Fatalf("Expression is not ast.WhenExpression. got=%T", whenExpr.Expression)
			}

			if whenStmt.Block == nil || len(whenStmt.Block.Statements) == 0 {
				t.Fatalf("When block is nil or empty")
			}

			switchStmt, ok := whenStmt.Block.Statements[0].(*ast.SwitchStatement)
			if !ok {
				t.Fatalf("When block does not contain a SwitchStatement. got=%T", whenStmt.Block.Statements[0])
			}

			if len(switchStmt.Cases) < 3 {
				t.Fatalf("Switch statement should have at least 3 cases, got %d", len(switchStmt.Cases))
			}

			if switchStmt.Default == nil {
				t.Fatalf("Switch statement should have a default case")
			}
		})
	}
}

func TestMatchesRegexExpression(t *testing.T) {
	input := `
when HTTP_REQUEST {
    if { [HTTP::uri] matches_regex {^/api/v2/users/\d+$} } {
        log local0. "Matched user API endpoint"
    }
}
`

	l := lexer.New(input)
	p := New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("program.Statements does not contain 1 statement. got=%d",
			len(program.Statements))
	}

	exprStmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("program.Statements[0] is not ast.ExpressionStatement. got=%T",
			program.Statements[0])
	}

	whenExpr, ok := exprStmt.Expression.(*ast.WhenExpression)
	if !ok {
		t.Fatalf("exprStmt.Expression is not ast.WhenExpression. got=%T",
			exprStmt.Expression)
	}

	if whenExpr.Event.String() != "HTTP_REQUEST" {
		t.Errorf("whenExpr.Event is not 'HTTP_REQUEST'. got=%s", whenExpr.Event)
	}

	if len(whenExpr.Block.Statements) != 1 {
		t.Fatalf("whenExpr.Block does not contain 1 statement. got=%d",
			len(whenExpr.Block.Statements))
	}

	ifStmt, ok := whenExpr.Block.Statements[0].(*ast.IfStatement)
	if !ok {
		t.Fatalf("whenExpr.Block.Statements[0] is not ast.IfStatement. got=%T",
			whenExpr.Block.Statements[0])
	}

	matchesRegexExpr, ok := ifStmt.Condition.(*ast.InfixExpression)
	if !ok {
		t.Fatalf("ifStmt.Condition is not ast.InfixExpression. got=%T",
			ifStmt.Condition)
	}

	if matchesRegexExpr.Operator != "matches_regex" {
		t.Fatalf("matchesRegexExpr.Operator is not 'matches_regex'. got=%s",
			matchesRegexExpr.Operator)
	}

	arrayLiteral, ok := matchesRegexExpr.Left.(*ast.ArrayLiteral)
	if !ok {
		t.Fatalf("matchesRegexExpr.Left is not ast.ArrayLiteral. got=%T",
			matchesRegexExpr.Left)
	}

	if len(arrayLiteral.Elements) != 1 {
		t.Fatalf("arrayLiteral does not contain 1 element. got=%d",
			len(arrayLiteral.Elements))
	}

	httpExpr, ok := arrayLiteral.Elements[0].(*ast.HttpExpression)
	if !ok {
		t.Fatalf("arrayLiteral.Elements[0] is not ast.HttpExpression. got=%T",
			arrayLiteral.Elements[0])
	}

	if httpExpr.Command.Value != "HTTP::uri" {
		t.Errorf("httpExpr.Command is not 'HTTP::uri'. got=%s", httpExpr.Command)
	}

	regexPattern, ok := matchesRegexExpr.Right.(*ast.RegexPattern)
	if !ok {
		t.Fatalf("matchesRegexExpr.Right is not ast.StringLiteral. got=%T",
			matchesRegexExpr.Right)
	}

	expectedPattern := "^/api/v2/users/\\d+$"
	if regexPattern.Value != expectedPattern {
		t.Fatalf("regexPattern.Value is not %q. got=%q",
			expectedPattern, regexPattern.Value)
	}
}
