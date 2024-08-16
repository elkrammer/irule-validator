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
	// p.registerPrefix(token.ARRAY, p.parseArrayOperation)
	p.registerPrefix(token.ASTERISK, p.parsePrefixExpression)
	p.registerPrefix(token.BANG, p.parsePrefixExpression)
	p.registerPrefix(token.DOLLAR, p.parseVariableOrArrayAccess)
	p.registerPrefix(token.FALSE, p.parseBoolean)
	p.registerPrefix(token.IDENT, p.parseIdentifier)
	p.registerPrefix(token.IF, p.parseIfExpression)
	p.registerPrefix(token.LBRACE, p.parseHashLiteral)
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(token.MINUS, p.parsePrefixExpression)
	p.registerPrefix(token.NUMBER, p.parseNumberLiteral)
	p.registerPrefix(token.RBRACKET, p.parseArrayLiteral)
	p.registerPrefix(token.RPAREN, p.parseGroupedExpression)
	p.registerPrefix(token.SET, p.parseSetExpression)
	p.registerPrefix(token.STRING, p.parseStringLiteral)
	p.registerPrefix(token.TRUE, p.parseBoolean)
	p.registerPrefix(token.WHEN, p.parseWhenExpression)
	p.registerPrefix(token.HTTP_URI, p.parseHttpUri)
	// p.registerPrefix(token.CONTAINS, p.parseContainsExpression)
	// p.registerPrefix(token.HTTP_REQUEST, p.parseHttpRequestEvent)
	// p.registerPrefix(token.HTTP_RESPONSE, p.parseHttpResponseEvent)
	// p.registerPrefix(token.SWITCH, p.parseSwitchExpression)

	p.infixParseFns = make(map[token.TokenType]infixParseFn)
	p.registerInfix(token.ASTERISK, p.parseInfixExpression)
	p.registerInfix(token.EQ, p.parseInfixExpression)
	p.registerInfix(token.GT, p.parseInfixExpression)
	p.registerInfix(token.LBRACKET, p.parseIndexExpression)
	p.registerInfix(token.LPAREN, p.parseCallExpression)
	p.registerInfix(token.LT, p.parseInfixExpression)
	p.registerInfix(token.MINUS, p.parseInfixExpression)
	p.registerInfix(token.NOT_EQ, p.parseInfixExpression)
	p.registerInfix(token.PLUS, p.parseInfixExpression)
	p.registerInfix(token.SLASH, p.parseInfixExpression)
	p.registerInfix(token.STARTS_WITH, p.parseInfixExpression)
	p.registerInfix(token.ENDS_WITH, p.parseInfixExpression)
	p.registerInfix(token.MATCHES, p.parseInfixExpression)
	p.registerInfix(token.AND, p.parseInfixExpression)
	p.registerInfix(token.OR, p.parseInfixExpression)
	p.registerInfix(token.CONTAINS, p.parseContainsExpression)

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
		// Only advance if we're not at EOF and the current token isn't SET
		// This ensures we don't skip the next statement after a semicolon
		// if !p.curTokenIs(token.EOF) && p.curToken.Type != token.SET {
		// 	p.nextToken()
		// }
		// p.nextToken()
		for p.curTokenIs(token.SEMICOLON) {
			p.nextToken()
		}
	}
	if config.DebugMode {
		fmt.Printf("DEBUG: Finished parsing program, total statements: %d\n", len(program.Statements))
	}
	return program
}

func (p *Parser) parseStatement() ast.Statement {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseStatement - Current token: %s, Peek token: %s\n", p.curToken.Type, p.peekToken.Type)
	}
	switch p.curToken.Type {
	case token.SET:
		stmt := p.parseSetStatement()
		if stmt == nil {
			fmt.Printf("Failed to parse SET statement\n")
		}
		return stmt
	case token.RETURN:
		return p.parseReturnStatement()
	case token.SEMICOLON:
		p.nextToken()
		return nil
	case token.IDENT:
		if p.curToken.Literal == "array" && p.peekTokenIs(token.IDENT) && p.peekToken.Literal == "set" {
			fmt.Printf("DEBUG: MAGIC ARRAY")
			return p.parseArraySetStatement()
		}
		return p.parseExpressionStatement()
	default:
		return p.parseExpressionStatement()
	}
}

// func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
//     stmt := &ast.ReturnStatement{Token: p.curToken}
//
//     p.nextToken()
//
//     if p.curTokenIs(token.NUMBER) {
//         value, _ := strconv.Atoi(p.curToken.Literal)
//         stmt.ReturnValue = &ast.IntegerLiteral{Token: p.curToken, Value: int64(value)}
//     } else {
//         stmt.ReturnValue = p.parseExpression(LOWEST)
//     }
//
//     if p.peekTokenIs(token.SEMICOLON) {
//         p.nextToken()
//     }
//
//     if p.peekTokenIs(token.RBRACE) {
//         p.nextToken()
//     }
//
//     return stmt
// }

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.curToken}

	p.nextToken()

	stmt.ReturnValue = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	if p.peekTokenIs(token.RBRACE) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseSetStatement() *ast.SetStatement {
	stmt := &ast.SetStatement{Token: p.curToken}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseSetStatement Start\n")
	}

	if !p.expectPeek(token.IDENT) {
		fmt.Printf("Expected IDENT, got: %+v\n", p.curToken)
		p.errors = append(p.errors, fmt.Sprintf("Expected next token to be IDENT, got %s instead", p.peekToken.Type))
		return nil
	}

	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Check for array index
	if p.peekTokenIs(token.LPAREN) {
		if config.DebugMode {
			fmt.Printf("DEBUG: Parsing array index")
		}
		p.nextToken() // consume '('
		p.nextToken() // move to index
		stmt.Index = p.parseExpression(LOWEST)

		if !p.expectPeek(token.RPAREN) {
			p.errors = append(p.errors, fmt.Sprintf("Expected next token to be ), got %s instead", p.peekToken.Type))
			return nil
		}

		if config.DebugMode {
			fmt.Printf("DEBUG: Finished parsing array index: %+v\n", stmt.Index)
		}
	}

	p.nextToken() // Move to the value

	// Parse the value
	stmt.Value = p.parseExpression(LOWEST)

	if config.DebugMode {
		fmt.Printf("DEBUG: parseSetStatement End: set %s(%v) %v\n", stmt.Name.Value, stmt.Index, stmt.Value)
	}

	return stmt
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseExpressionStatement - Starting\n")
	}
	stmt := &ast.ExpressionStatement{Token: p.curToken}

	leftExp := p.parseExpression(LOWEST)

	// Check if this might be a function call
	if p.peekTokenIs(token.IDENT) || p.peekTokenIs(token.NUMBER) {
		stmt.Expression = p.parseCallExpression(leftExp)
	} else {
		stmt.Expression = leftExp
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseExpressionStatement - Parsed expression: %T\n", stmt.Expression)
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseExpression(precedence int) ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseExpression - Current token: %s, Precedence: %d\n", p.curToken.Type, precedence)
	}

	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		if p.curToken.Type == token.LBRACKET {
			return p.parseGroupedExpression()
		}
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}
		p.nextToken()
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
	// if p.peekTokenIs(token.ELSE) {
	// 	p.nextToken()
	//
	// 	if !p.expectPeek(token.LBRACE) {
	// 		return nil
	// 	}
	//
	// 	alternative := p.parseBlockStatement()
	// 	if alternative == nil {
	// 		p.errors = append(p.errors, "missing closing brace")
	// 		return nil
	// 	}
	// 	expression.Alternative = alternative
	// }

	if config.DebugMode {
		fmt.Printf("DEBUG: Finished parsing if expression\n")
	}
	return expression
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()

	exp := p.parseExpression(LOWEST)

	if p.peekTokenIs(token.RPAREN) {
		if !p.expectPeek(token.RPAREN) {
			return nil
		}
	} else if p.peekTokenIs(token.RBRACKET) {
		if !p.expectPeek(token.RBRACKET) {
			return nil
		}
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
	if config.DebugMode {
		fmt.Println("DEBUG: Starting parsing block statement")
	}

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	return block
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	fmt.Println("DEBUG: Parsing index expression")

	exp := &ast.IndexExpression{Token: p.curToken, Left: left}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	p.nextToken() // move past '(' token
	exp.Index = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
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
	list := []ast.Expression{}

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

func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseCallExpression - Function: %T\n", function)
	}

	exp := &ast.CallExpression{Token: p.curToken, Function: function}
	exp.Arguments = []ast.Expression{}
	if config.DebugMode {
		fmt.Printf("DEBUG: parseCallExpression - Arguments: %T\n", exp.Arguments)
	}

	for !p.peekTokenIs(token.SEMICOLON) && !p.peekTokenIs(token.EOF) {
		p.nextToken()
		arg := p.parseExpression(LOWEST)
		if arg != nil {
			exp.Arguments = append(exp.Arguments, arg)
		}
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseCallExpression - Function: %v, Arguments: %d\n", function, len(exp.Arguments))
	}
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

// func (p *Parser) parseArrayOperation() ast.Expression {
// 	operation := &ast.ArrayOperation{Token: p.curToken}
//
// 	// Expect the next token to be the array command
// 	if !p.expectPeek(token.IDENT) {
// 		return nil
// 	}
// 	operation.Command = p.curToken.Literal
//
// 	// Parse array name
// 	if !p.expectPeek(token.IDENT) {
// 		return nil
// 	}
// 	operation.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
//
// 	// Check if there's an index
// 	if p.peekTokenIs(token.LPAREN) {
// 		p.nextToken() // consume '('
// 		// p.nextToken() // move to index
// 		operation.Index = p.parseExpression(LOWEST)
// 		if !p.expectPeek(token.RPAREN) {
// 			return nil
// 		}
// 	}
//
// 	// If it's a 'set' operation, parse the value
// 	if operation.Command == "set" {
// 		p.nextToken() // move to value
// 		operation.Value = p.parseExpression(LOWEST)
// 	}
//
// 	return operation
// }

func (p *Parser) parseSetExpression() ast.Expression {
	stmt := &ast.SetStatement{Token: p.curToken}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseSetExpression - Starting\n")
	}

	if !p.expectPeek(token.IDENT) {
		if config.DebugMode {
			fmt.Printf("DEBUG: parseSetExpression - Expected IDENT, got %s\n", p.curToken.Type)
		}
		return nil
	}

	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseSetExpression - Name: %s\n", stmt.Name.Value)
	}

	p.nextToken() // move past the variable Name

	if p.peekTokenIs(token.LPAREN) {
		if config.DebugMode {
			fmt.Printf("DEBUG: parseSetExpression - Parsing array index\n")
		}
		p.nextToken() // consume '('
		p.nextToken() // move to the index expression
		stmt.Index = p.parseExpression(LOWEST)
		if !p.expectPeek(token.RPAREN) {
			if config.DebugMode {
				fmt.Printf("DEBUG: parseSetExpression - Expected RPAREN, got %s\n", p.curToken.Type)
			}
			return nil
		} else {
			p.nextToken() // move past the identifier or closing parenthesis
		}
	}

	stmt.Value = p.parseExpression(LOWEST)

	if config.DebugMode {
		fmt.Printf("DEBUG: parseSetExpression - Value parsed: %v\n", stmt.Value)
		fmt.Printf("DEBUG: parseSetExpression - Completed: %v\n", stmt)
	}

	return stmt
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{Token: p.curToken}
	array.Elements = p.parseExpressionList(token.RBRACKET)
	return array
}

func (p *Parser) parseArraySetStatement() *ast.SetStatement {
	fmt.Println("DEBUG: Parsing array set statement")
	stmt := &ast.SetStatement{Token: p.curToken, IsArraySet: true}

	// Consume "array"
	p.nextToken()
	// Consume "set"
	if !p.expectPeek(token.IDENT) || p.curToken.Literal != "set" {
		p.errors = append(p.errors, "Expected 'set' after 'array'")
		return nil
	}
	p.nextToken()

	// Parse the array name
	if !p.expectPeek(token.IDENT) {
		p.errors = append(p.errors, fmt.Sprintf("Expected next token to be IDENT, got %s instead", p.peekToken.Type))
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	fmt.Printf("DEBUG: Array name: %s\n", stmt.Name.Value)

	// Parse the index (if any)
	if p.peekTokenIs(token.LPAREN) {
		p.nextToken() // consume '('
		fmt.Println("DEBUG: Parsing array index")
		p.nextToken() // move to index
		stmt.Index = p.parseExpression(LOWEST)
		if !p.expectPeek(token.RPAREN) {
			p.errors = append(p.errors, fmt.Sprintf("Expected next token to be ), got %s instead", p.peekToken.Type))
			return nil
		}
		fmt.Printf("DEBUG: Array index: %v\n", stmt.Index)
	}

	// Parse the value
	p.nextToken()
	fmt.Println("DEBUG: Parsing array value")
	stmt.Value = p.parseExpression(LOWEST)
	fmt.Printf("DEBUG: Array value: %v\n", stmt.Value)

	return stmt
}

func (p *Parser) parseVariableOrArrayAccess() ast.Expression {
	p.nextToken() // consume '$'
	if !p.curTokenIs(token.IDENT) {
		p.errors = append(p.errors, fmt.Sprintf("Expected identifier after $, got %s instead", p.curToken.Type))
		return nil
	}

	varExp := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal, IsVariable: true}

	if p.peekTokenIs(token.LPAREN) {
		p.nextToken() // consume '('
		p.nextToken() // move to index
		index := p.parseExpression(LOWEST)
		if !p.expectPeek(token.RPAREN) {
			return nil
		}
		return &ast.IndexExpression{Left: varExp, Index: index}
	}

	return varExp
}

func (p *Parser) parseWhenExpression() ast.Expression {
	expression := &ast.WhenExpression{Token: p.curToken}

	if !p.expectPeek(token.HTTP_REQUEST) {
		return nil
	}

	expression.Event = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	expression.Block = p.parseBlockStatement()

	return expression
}

func (p *Parser) parseHttpRequestEvent() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseHttpUri() ast.Expression {
	expression := &ast.HttpUriExpression{Token: p.curToken}

	if !p.expectPeek(token.COLON) {
		return nil
	}

	if !p.expectPeek(token.COLON) {
		return nil
	}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	expression.Method = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	return expression
}

func (p *Parser) parseContainsExpression(left ast.Expression) ast.Expression {
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

// func (p *Parser) parseContainsExpression(left ast.Expression) ast.Expression {
// 	expression := &ast.InfixExpression{
// 		Token:    p.curToken,
// 		Operator: p.curToken.Literal,
// 	}
//
// 	if left == nil {
// 		precedence := PREFIX
// 		p.nextToken()
// 		expression.Right = p.parseExpression(precedence)
// 	} else {
// 		expression.Left = left
// 		precedence := p.curPrecedence()
// 		p.nextToken()
// 		expression.Right = p.parseExpression(precedence)
// 	}
//
// 	return expression
// }
