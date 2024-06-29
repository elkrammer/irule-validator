package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/elkrammer/irule-validator/ast"
	"github.com/elkrammer/irule-validator/config"
	"github.com/elkrammer/irule-validator/lexer"
	"github.com/elkrammer/irule-validator/token"
)

const (
	_ int = iota
	LOWEST
	EQUALS      // ==
	LESSGREATER // > or <
	SUM         // +
	PRODUCT     // *
	PREFIX      // -X or !X
	CALL        // myFunction(X)
)

var precedences = map[token.TokenType]int{
	token.EQ:       EQUALS,
	token.NOT_EQ:   EQUALS,
	token.LT:       LESSGREATER,
	token.GT:       LESSGREATER,
	token.PLUS:     SUM,
	token.MINUS:    SUM,
	token.SLASH:    PRODUCT,
	token.ASTERISK: PRODUCT,
	token.LPAREN:   CALL,
}

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

type Parser struct {
	l      *lexer.Lexer
	errors []string

	curToken  token.Token
	peekToken token.Token

	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}
	// read two tokens so curToken and peekToken are both set
	p.nextToken()
	p.nextToken()

	p.prefixParseFns = make(map[token.TokenType]prefixParseFn)
	p.registerPrefix(token.IDENT, p.parseIdentifier)
	p.registerPrefix(token.NUMBER, p.parseNumberLiteral)
	p.registerPrefix(token.STRING, p.parseStringLiteral)
	p.registerPrefix(token.TRUE, p.parseBoolean)
	p.registerPrefix(token.FALSE, p.parseBoolean)
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(token.IF, p.parseIfExpression)
	p.registerPrefix(token.MINUS, p.parsePrefixExpression)
	p.registerPrefix(token.BANG, p.parsePrefixExpression)
	p.registerPrefix(token.FUNCTION, p.parseFunctionLiteral)
	p.registerPrefix(token.LBRACE, p.parseHashLiteral)
	p.registerPrefix(token.LBRACKET, p.parseArrayLiteral)
	p.registerPrefix(token.ASTERISK, p.parsePrefixExpression)
	p.registerPrefix(token.EXPR, p.parseExpr)

	p.infixParseFns = make(map[token.TokenType]infixParseFn)
	p.registerInfix(token.EQ, p.parseInfixExpression)
	p.registerInfix(token.NOT_EQ, p.parseInfixExpression)
	p.registerInfix(token.LT, p.parseInfixExpression)
	p.registerInfix(token.GT, p.parseInfixExpression)
	p.registerInfix(token.PLUS, p.parseInfixExpression)
	p.registerInfix(token.MINUS, p.parseInfixExpression)
	p.registerInfix(token.SLASH, p.parseInfixExpression)
	p.registerInfix(token.ASTERISK, p.parseInfixExpression)
	p.registerInfix(token.LPAREN, p.parseCallExpression)
	p.registerInfix(token.LBRACKET, p.parseIndexExpression)

	return p
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) peekError(t token.TokenType) {
	msg := fmt.Sprintf("Expected next token to be %s, got %s instead", t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
	if config.DebugMode {
		fmt.Printf("DEBUG: Advanced tokens - Current: %s, Peek: %s\n", p.curToken.Type, p.peekToken.Type)
	}
}

func (p *Parser) ParseProgram() *ast.Program {
	if config.DebugMode {
		fmt.Printf("DEBUG: Starting to parse program\n")
	}
	program := &ast.Program{}
	program.Statements = []ast.Statement{}

	for !p.curTokenIs(token.EOF) {
		if config.DebugMode {
			fmt.Printf("DEBUG: Parsing statement, current token: %s\n", p.curToken.Type)
		}
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		} else {
			fmt.Printf("Failed to parse statement at token: %+v\n", p.curToken) // Debug print
		}
		p.nextToken()
	}
	if config.DebugMode {
		fmt.Printf("DEBUG: Finished parsing program, total statements: %d\n", len(program.Statements))
	}
	return program
}

func (p *Parser) parseStatement() ast.Statement {
	if config.DebugMode {
		fmt.Printf("DEBUG: Parsing statement, token type: %s\n", p.curToken.Type)
	}
	switch p.curToken.Type {
	case token.SET:
		return p.parseSetStatement()
	case token.RETURN:
		return p.parseReturnStatement()
	case token.SEMICOLON:
		p.nextToken()
		return nil
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.curToken}

	p.nextToken()

	stmt.ReturnValue = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseSetStatement() *ast.SetStatement {
	stmt := &ast.SetStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		fmt.Printf("Expected IDENT, got: %+v\n", p.curToken)
		return nil
	}

	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	p.nextToken() // Move past the variable name

	// Check if the next token is a '['
	if p.curTokenIs(token.LBRACKET) {
		arrayLiteral := p.parseArrayLiteral()
		if arrayLiteral == nil {
			fmt.Printf("Failed to parse array literal: %s\n", strings.Join(p.errors, ", "))
			return nil
		}
		stmt.Value = arrayLiteral
	} else {
		// Parse the value as an expression
		stmt.Value = p.parseExpression(LOWEST)
	}

	if stmt.Value == nil {
		fmt.Printf("Failed to parse set statement value: %s\n", strings.Join(p.errors, ", "))
		return nil
	}

	// Consume the semicolon if it's there
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{Token: p.curToken}
	stmt.Expression = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.NEWLINE) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseExpression(precedence int) ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: Parsing expression with precedence %d, current token: %s\n", precedence, p.curToken.Type)
	}
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()
	if config.DebugMode {
		fmt.Printf("DEBUG: Parsed left expression: %T\n", leftExp)
	}

	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
		if config.DebugMode {
			fmt.Printf("DEBUG: Continuing expression, peek token: %s\n", p.peekToken.Type)
		}
		if p.peekTokenIs(token.LPAREN) {
			p.nextToken() // Consume opening parenthesis
			leftExp = p.parseCallExpression(leftExp)
			continue
		}

		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken() // Consume infix token
		leftExp = infix(leftExp)
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: Finished parsing expression\n")
	}
	return leftExp
}

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{
		Token:      p.curToken,
		Value:      p.curToken.Literal,
		IsVariable: strings.HasPrefix(p.curToken.Literal, "$"),
	}
}

func (p *Parser) parseNumberLiteral() ast.Expression {
	lit := &ast.NumberLiteral{Token: p.curToken}

	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as integer", p.curToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}

	lit.Value = value
	return lit
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	expression := &ast.PrefixExpression{
		Token: p.curToken,
	}

	// Handle the BANG operator
	if p.curToken.Type == token.BANG {
		expression.Operator = p.curToken.Literal
		p.nextToken()
		expression.Right = p.parseExpression(PREFIX)
		return expression
	}

	// Handle the MINUS operator
	if p.curToken.Type == token.MINUS {
		expression.Operator = p.curToken.Literal
		p.nextToken() // Consume the MINUS token
		expression.Right = p.parseExpression(PREFIX)
		return expression
	}

	return nil
}

func (p *Parser) parseBoolean() ast.Expression {
	return &ast.Boolean{Token: p.curToken, Value: p.curTokenIs(token.TRUE)}
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseFunctionLiteral() ast.Expression {
	proc := &ast.FunctionLiteral{Token: p.curToken}

	// Expect the function name after 'proc'
	if !p.expectPeek(token.IDENT) {
		fmt.Println("Error: Expected function name after 'proc'")
		return nil
	}

	// Expect an opening brace '{' after the function name.
	if !p.expectPeek(token.LBRACE) {
		fmt.Println("Error: Expected opening brace after 'proc'")
		return nil
	}

	proc.Parameters = p.parseFunctionParameters()

	// The next token should be the opening brace for the function body
	if !p.expectPeek(token.LBRACE) {
		fmt.Println("Error: Expected opening brace for function body")
		return nil
	}

	proc.Body = p.parseBlockStatement()

	return proc
}

func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	identifiers := []*ast.Identifier{}

	// If the next token is '}', then there are no parameters
	if p.peekTokenIs(token.RBRACE) {
		p.nextToken() // Consume the '}' token
		return identifiers
	}

	p.nextToken() // Consume the opening brace

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		if p.curTokenIs(token.IDENT) {
			identifier := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
			identifiers = append(identifiers, identifier)
		}
		p.nextToken()

		if p.curTokenIs(token.COMMA) {
			p.nextToken() // Skip comma
		}
	}

	if !p.curTokenIs(token.RBRACE) {
		fmt.Println("Error: Expected closing brace for parameters")
	}

	return identifiers
}

func (p *Parser) parseIfExpression() ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: Parsing if expression\n")
	}
	expression := &ast.IfExpression{Token: p.curToken}

	// Expect an opening brace after 'if'
	if !p.expectPeek(token.LBRACE) {
		p.errors = append(p.errors, fmt.Sprintf("expected '{' after 'if', but got %s", p.peekToken.Type))
		return nil
	}

	p.nextToken() // Consume the '{'
	if config.DebugMode {
		fmt.Printf("DEBUG: Parsing if condition\n")
	}
	expression.Condition = p.parseExpression(LOWEST)

	// Expect a closing brace after the condition
	if !p.expectPeek(token.RBRACE) {
		return nil
	}

	// Expect an opening brace for the consequence block
	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	// Parse the consequence block
	if config.DebugMode {
		fmt.Printf("DEBUG: Parsing if consequence\n")
	}
	consequence := p.parseBlockStatement()
	if consequence == nil {
		p.errors = append(p.errors, "missing closing brace")
		return nil
	}
	expression.Consequence = consequence

	// Handle the 'else' part
	if p.peekTokenIs(token.ELSE) {
		p.nextToken()

		if !p.expectPeek(token.LBRACE) {
			return nil
		}

		alternative := p.parseBlockStatement()
		if alternative == nil {
			p.errors = append(p.errors, "missing closing brace")
			return nil
		}
		expression.Alternative = alternative
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: Finished parsing if expression\n")
	}
	return expression
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()

	exp := p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return exp
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	} else {
		p.peekError(t)
		return false
	}
}

func (p *Parser) registerPrefix(tokenType token.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType token.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func (p *Parser) noPrefixParseFnError(t token.TokenType) {
	msg := fmt.Sprintf("no prefix parse function for %s found", t)
	p.errors = append(p.errors, msg)
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}

	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}

	return LOWEST
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.curToken}
	block.Statements = []ast.Statement{}

	p.nextToken()

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	if p.curTokenIs(token.EOF) {
		p.errors = append(p.errors, "missing closing brace")
		return nil
	}

	return block
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	exp := &ast.IndexExpression{Token: p.curToken, Left: left}

	p.nextToken() // consume '[' token
	exp.Index = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RBRACKET) {
		return nil
	}

	return exp
}

func (p *Parser) parseHashLiteral() ast.Expression {
	hash := &ast.HashLiteral{Token: p.curToken}
	hash.Pairs = make(map[ast.StringLiteral]ast.Expression)

	for !p.peekTokenIs(token.RBRACE) {
		p.nextToken()

		// Parse key
		if !p.expectPeek(token.STRING) {
			return nil
		}
		key := &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}

		// Parse value
		if !p.expectPeek(token.STRING) {
			return nil
		}
		value := &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}

		hash.Pairs[*key] = value

		if !p.peekTokenIs(token.RBRACE) && !p.expectPeek(token.COMMA) {
			return nil
		}
	}

	if !p.expectPeek(token.RBRACE) {
		return nil
	}

	return hash
}

func (p *Parser) parseExpressionList(end token.TokenType) []ast.Expression {
	var list []ast.Expression

	// If the next token is the end token, return an empty list
	if p.peekTokenIs(end) {
		p.nextToken() // Consume the end token
		return list
	}

	p.nextToken() // Move past the opening bracket

	// Parse expressions until encountering the end token
	list = append(list, p.parseExpression(LOWEST))
	for p.peekTokenIs(token.COMMA) {
		p.nextToken() // Consume the comma
		p.nextToken() // Move to the next expression
		list = append(list, p.parseExpression(LOWEST))
	}

	// Ensure that the list is terminated with the end token
	if !p.expectPeek(end) {
		return nil
	}

	return list
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{Token: p.curToken}
	p.nextToken() // consume the '['

	if p.curTokenIs(token.IDENT) && p.curToken.Literal == "expr" {
		p.nextToken() // consume 'expr'
		expr := p.parseExpression(LOWEST)
		if expr == nil {
			return nil
		}
		array.Elements = []ast.Expression{&ast.ExprExpression{Token: p.curToken, Expression: expr}}

		// Consume tokens until we reach the closing bracket
		for !p.curTokenIs(token.RBRACKET) && !p.curTokenIs(token.EOF) {
			p.nextToken()
		}
	} else {
		array.Elements = p.parseExpressionList(token.RBRACKET)
	}

	if !p.curTokenIs(token.RBRACKET) {
		p.errors = append(p.errors, fmt.Sprintf("expected ], got %s instead", p.curToken.Type))
		return nil
	}
	p.nextToken() // consume the ']'

	return array
}

func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	exp := &ast.CallExpression{Token: p.curToken, Function: function}
	p.nextToken() // Consume the '('

	args := []ast.Expression{}
	if p.peekTokenIs(token.RPAREN) {
		p.nextToken() // Consume the ')'
	} else {
		for {
			arg := p.parseExpression(LOWEST)
			if arg == nil {
				return nil
			}
			args = append(args, arg)

			if !p.peekTokenIs(token.COMMA) {
				break
			}
			p.nextToken() // Consume the ','
		}

		if !p.expectPeek(token.RPAREN) {
			return nil
		}
		p.nextToken() // Consume the ')'
	}

	exp.Arguments = args
	return exp
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()

	expression.Right = p.parseExpression(precedence)
	return expression
}

func (p *Parser) parseExprExpression(left ast.Expression) ast.Expression {
	exprExpr := &ast.ExprExpression{
		Token:      p.curToken,
		Expression: left,
	}

	p.nextToken() // Consume the 'expr' token

	// Parse the expression inside the 'expr' command
	exprExpr.Expression = p.parseExpression(LOWEST)

	return exprExpr
}

func (p *Parser) parseExpr() ast.Expression {
	exprExpr := &ast.ExprExpression{
		Token: p.curToken,
	}

	p.nextToken() // Consume the 'expr' token

	exprExpr.Expression = p.parseExprBody()

	return exprExpr
}

func (p *Parser) parseExprBody() ast.Expression {
	p.nextToken() // Consume the '{'

	expr := p.parseExpression(LOWEST)

	if !p.expectPeek(token.RBRACE) {
		return nil
	}

	p.nextToken() // Consume the '}'

	return expr
}

func (p *Parser) parseParenthesizedExpression() ast.Expression {
	p.nextToken() // Consume the opening parenthesis '('

	expr := p.parseExpression(LOWEST)
	if expr == nil {
		return nil
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	p.nextToken() // Consume the closing parenthesis ')'
	return &ast.ParenthesizedExpression{Expression: expr}
}

func (p *Parser) parseBinaryExpression(precedence int, left ast.Expression) ast.Expression {
	op := p.curToken

	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
		p.nextToken()
		right := p.parseExpression(p.peekPrecedence())
		if right == nil {
			return nil
		}
		left = &ast.InfixExpression{
			Token:    op,
			Left:     left,
			Operator: op.Literal,
			Right:    right,
		}
	}

	return left
}
