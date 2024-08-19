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
	p.registerPrefix(token.CONTAINS, p.parseContainsExpression)
	p.registerPrefix(token.DOLLAR, p.parseVariableOrArrayAccess)
	p.registerPrefix(token.FALSE, p.parseBoolean)
	p.registerPrefix(token.HTTP_URI, p.parseHttpCommand)
	p.registerPrefix(token.IDENT, p.parseIdentifier)
	p.registerPrefix(token.LBRACE, p.parseHashLiteral)
	p.registerPrefix(token.LBRACKET, p.parseArrayLiteral)
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(token.MINUS, p.parsePrefixExpression)
	p.registerPrefix(token.NUMBER, p.parseNumberLiteral)
	p.registerPrefix(token.RBRACKET, p.parseArrayLiteral)
	p.registerPrefix(token.RPAREN, p.parseGroupedExpression)
	p.registerPrefix(token.SET, p.parseSetExpression)
	p.registerPrefix(token.STRING, p.parseStringLiteral)
	p.registerPrefix(token.TRUE, p.parseBoolean)
	p.registerPrefix(token.WHEN, p.parseWhenExpression)
	p.registerPrefix(token.HTTP_COMMAND, p.parseHttpCommand)
	// p.registerPrefix(token.RBRACE, p.parseBracketExpression)
	// p.registerPrefix(token.HTTP_REQUEST, p.parseWhenExpression)
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
	p.registerInfix(token.CONTAINS, p.parseInfixExpression)

	return p
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) peekError(t token.TokenType) {
	msg := fmt.Sprintf("peekError: Expected next token to be %s, got %s instead", t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
	// if config.DebugMode {
	// fmt.Printf("DEBUG: Advanced tokens - Current: %s, Peek: %s\n", p.curToken.Type, p.peekToken.Type)
	// }
}

func (p *Parser) ParseProgram() *ast.Program {
	if config.DebugMode {
		fmt.Printf("DEBUG: Starting to parse program\n")
	}
	program := &ast.Program{}
	program.Statements = []ast.Statement{}

	for !p.curTokenIs(token.EOF) {
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
		fmt.Printf("DEBUG: parseStatement - Current token: %s, Peek token: %s\n", p.curToken.Type, p.peekToken.Type)
	}

	var stmt ast.Statement

	switch p.curToken.Type {
	case token.SET:
		stmt = p.parseSetStatement()
		if stmt == nil {
			fmt.Printf("DEBUG: parseStatement - Failed to parse SET statement\n")
		}
		return stmt
	case token.RETURN:
		if config.DebugMode {
			fmt.Printf("DEBUG: Calling parseReturnStatement\n")
		}
		stmt = p.parseReturnStatement()
	case token.IDENT:
		stmt = p.parseExpressionStatement()
	case token.WHEN:
		stmt = &ast.ExpressionStatement{
			Token:      p.curToken,
			Expression: p.parseWhenExpression(),
		}
	case token.IF:
		stmt = p.parseIfStatement()
	case token.LBRACE:
		stmt = p.parseBlockStatement()
	default:
		stmt = p.parseExpressionStatement()
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseStatement exit - Parsed: %T\n", stmt)
	}
	return stmt
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	if config.DebugMode {
		fmt.Printf("DEBUG: Start parseReturnStatement\n")
	}

	stmt := &ast.ReturnStatement{Token: p.curToken}

	p.nextToken() // consume the 'return' token

	switch p.curToken.Type {
	case token.STRING, token.NUMBER:
		stmt.ReturnValue = p.parseExpression(LOWEST)
	default:
		p.errors = append(p.errors, fmt.Sprintf("Expected STRING or NUMBER after return, got %s", p.curToken.Type))
		return nil
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: End parseReturnStatement\n")
	}
	return stmt
}

func (p *Parser) parseSetStatement() *ast.SetStatement {
	stmt := &ast.SetStatement{Token: p.curToken}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseSetStatement Start\n")
	}

	if !p.expectPeek(token.IDENT) {
		fmt.Printf("DEBUG: parseSetStatement - Expected IDENT, got: %+v\n", p.curToken)
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
			fmt.Printf("DEBUG: parseSetStatement - Expected RPAREN, got: %+v\n", p.curToken)
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
		if p.curToken.Type == token.RBRACE {
			return nil // Return nil for closing brace
		}
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(token.SEMICOLON) && !p.peekTokenIs(token.RBRACE) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			if p.peekTokenIs(token.CONTAINS) {
				p.nextToken()
				leftExp = p.parseInfixExpression(leftExp)
			} else {
				return leftExp
			}
		} else {
			p.nextToken()
			leftExp = infix(leftExp)
		}
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: Finished parsing expression, type: %T\n", leftExp)
	}

	return leftExp
}

// func (p *Parser) parseExpression(precedence int) ast.Expression {
// 	if config.DebugMode {
// 		fmt.Printf("DEBUG: parseExpression - Current token: %s, Precedence: %d\n", p.curToken.Type, precedence)
// 	}
//
// 	var leftExp ast.Expression
//
// 	switch p.curToken.Type {
// 	case token.IDENT:
// 		return p.parseIdentifier()
// 	case token.NUMBER:
// 		return p.parseNumberLiteral()
// 	case token.LBRACKET:
// 		return p.parseBracketExpression()
// 	case token.HTTP_URI:
// 		leftExp = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
// 	case token.RBRACE:
// 		if config.DebugMode {
// 			fmt.Printf("DEBUG: parseExpression found closing brace\n")
// 		}
// 		return nil
// 	default:
// 		p.noPrefixParseFnError(p.curToken.Type)
// 		return nil
// 	}
//
// 	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
// 		infix := p.infixParseFns[p.peekToken.Type]
// 		if infix == nil {
// 			if config.DebugMode {
// 				fmt.Printf("DEBUG: parseExpression - No infix parse function for %s\n", p.peekToken.Type)
// 			}
// 			return leftExp
// 		}
// 		p.nextToken()
// 		leftExp = infix(leftExp)
// 	}
//
// 	if config.DebugMode {
// 		fmt.Printf("DEBUG: Finished parsing expression, type: %T\n", leftExp)
// 	}
//
// 	return leftExp
// }

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{
		Token:      p.curToken,
		Value:      p.curToken.Literal,
		IsVariable: strings.HasPrefix(p.curToken.Literal, "$"),
	}
}

func (p *Parser) parseNumberLiteral() ast.Expression {
	lit := &ast.NumberLiteral{Token: p.curToken}

	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
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

func (p *Parser) parseIfExpression() *ast.IfStatement {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseIfExpression Start, current token: %s\n", p.curToken.Type)
	}

	stmt := &ast.IfStatement{Token: p.curToken}

	// expect and consume opening brace
	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	p.nextToken()

	condition := p.parseExpression(LOWEST)

	// Check if the next token is 'contains'
	if p.peekTokenIs(token.CONTAINS) {
		p.nextToken() // Move to 'contains'
		containsExp := &ast.InfixExpression{
			Token:    p.curToken,
			Left:     condition,
			Operator: p.curToken.Literal,
		}
		p.nextToken() // Move past 'contains'
		containsExp.Right = p.parseExpression(LOWEST)
		condition = containsExp
	}

	stmt.Condition = condition

	// expect and consume closing brace
	if !p.expectPeek(token.RBRACE) {
		return nil
	}

	// expect and consume opening brace for consequence block
	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	stmt.Consequence = p.parseBlockStatement()

	if config.DebugMode {
		fmt.Printf("DEBUG: Finished parseifexpression\n")
	}

	return stmt
}

func (p *Parser) parseIfCondition() ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: Parsing if condition, current token: %s\n", p.curToken.Type)
	}

	var condition ast.Expression

	if p.curTokenIs(token.LBRACKET) {
		condition = p.parseHttpExpression()
	} else {
		condition = p.parseExpression(LOWEST)
	}

	if p.peekTokenIs(token.CONTAINS) || p.peekTokenIs(token.EQ) || p.peekTokenIs(token.NOT_EQ) {
		p.nextToken()
		condition = &ast.InfixExpression{
			Token:    p.curToken,
			Left:     condition,
			Operator: p.curToken.Literal,
			Right:    p.parseExpression(LOWEST),
		}
	}

	return condition
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()

	exp := p.parseExpression(LOWEST)

	if p.peekTokenIs(token.RPAREN) {
		if !p.expectPeek(token.RPAREN) {
			if config.DebugMode {
				fmt.Printf("DEBUG: parseGroupedExpression - Expected RPAREN, got %s\n", p.curToken.Type)
			}
			return nil
		}
	} else if p.peekTokenIs(token.RBRACKET) {
		if !p.expectPeek(token.RBRACKET) {
			if config.DebugMode {
				fmt.Printf("DEBUG: parseGroupedExpression - Expected RBRACKET, got %s\n", p.curToken.Type)
			}
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
	if config.DebugMode {
		fmt.Printf("DEBUG: Start parseBlockStatement\n")
	}
	block := &ast.BlockStatement{Token: p.curToken}
	block.Statements = []ast.Statement{}

	p.nextToken() // consume opening brace

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		if config.DebugMode {
			fmt.Printf("DEBUG: parseBlockStatement - Current token: %s\n", p.curToken.Type)
		}
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		// if !p.peekTokenIs(token.RBRACE) {
		p.nextToken()
		// 	return nil
		// }
	}

	// if !p.expectPeek(token.RBRACE) {
	// 	return nil
	// }

	if config.DebugMode {
		fmt.Printf("DEBUG: End parseBlockStatement, statements: %d\n", len(block.Statements))
	}

	// p.nextToken() // consume closing brace

	return block
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	fmt.Printf("DEBUG: Parsing index expression\n")

	exp := &ast.IndexExpression{Token: p.curToken, Left: left}

	if !p.expectPeek(token.LPAREN) {
		if config.DebugMode {
			fmt.Printf("DEBUG: parseIndexExpression - Expected LPAREN, got %s\n", p.curToken.Type)
		}
		return nil
	}

	p.nextToken() // move past '(' token
	exp.Index = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		if config.DebugMode {
			fmt.Printf("DEBUG: parseIndexExpression - Expected RPAREN, got %s\n", p.curToken.Type)
		}
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
			if config.DebugMode {
				fmt.Printf("DEBUG: parseHashLiteral - Expected STRING, got %s\n", p.curToken.Type)
			}
			return nil
		}
		key := &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}

		// Parse value
		if !p.expectPeek(token.STRING) {
			if config.DebugMode {
				fmt.Printf("DEBUG: parseHashLiteral - Expected STRING, got %s\n", p.curToken.Type)
			}
			return nil
		}
		value := &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}

		hash.Pairs[*key] = value

		if !p.peekTokenIs(token.RBRACE) && !p.expectPeek(token.COMMA) {
			if config.DebugMode {
				fmt.Printf("DEBUG: parseHashLiteral - Expected COMMA for Peek token, got %s\n", p.curToken.Type)
			}
			return nil
		}
	}

	if !p.expectPeek(token.RBRACE) {
		if config.DebugMode {
			fmt.Printf("DEBUG: parseHashLiteral - Expected RBRACE, got %s\n", p.curToken.Type)
		}
		return nil
	}

	return hash
}

func (p *Parser) parseExpressionList(end token.TokenType) []ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseExpressionList Start\n")
	}
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
		fmt.Printf("DEBUG: parseExpressionList loop. list = %v\n", list)
	}

	// Ensure that the list is terminated with the end token
	if !p.expectPeek(end) {
		if config.DebugMode {
			fmt.Printf("DEBUG: parseExpressionList - Expected end, got %s\n", p.curToken.Type)
		}
		return nil
	}

	fmt.Printf("DEBUG: parseExpressionList End\n")
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
		if config.DebugMode {
			fmt.Printf("DEBUG: parseParenthesizedExpression - Expected end, got %s\n", p.curToken.Type)
		}
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
	fmt.Printf("DEBUG: ParseArraySetStatement")
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
		p.errors = append(p.errors, fmt.Sprintf("ParseArraySetStatemen - tExpected next token to be IDENT, got %s instead", p.peekToken.Type))
		return nil
	}
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	fmt.Printf("DEBUG: Array name: %s\n", stmt.Name.Value)

	// Parse the index (if any)
	if p.peekTokenIs(token.LPAREN) {
		p.nextToken() // consume '('
		fmt.Printf("DEBUG: Parsing array index")
		p.nextToken() // move to index
		stmt.Index = p.parseExpression(LOWEST)
		if !p.expectPeek(token.RPAREN) {
			p.errors = append(p.errors, fmt.Sprintf("ParseArraySetStatemen - tExpected next token to be ), got %s instead", p.peekToken.Type))
			return nil
		}
		fmt.Printf("DEBUG: Array index: %v\n", stmt.Index)
	}

	// Parse the value
	p.nextToken()
	fmt.Printf("DEBUG: Parsing array value")
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
			if config.DebugMode {
				fmt.Printf("DEBUG: parseVariableOrArrayAccess:  Expected RPAREN, got %s\n", p.curToken.Type)
			}
			return nil
		}
		return &ast.IndexExpression{Left: varExp, Index: index}
	}

	return varExp
}

func (p *Parser) parseHttpRequestEvent() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseHttpUri() ast.Expression {
	expression := &ast.HttpUriExpression{Token: p.curToken}
	if config.DebugMode {
		fmt.Printf("DEBUG: Parsing HTTP::URI expression\n")
	}

	if !p.expectPeek(token.IDENT) {
		if config.DebugMode {
			fmt.Printf("DEBUG: parseHttpRequestEvent - Expected IDENT, got %s\n", p.curToken.Type)
		}
		return nil
	}

	expression.Method = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	// Handle the closing bracket
	if !p.expectPeek(token.RBRACKET) {
		if config.DebugMode {
			fmt.Printf("DEBUG: parseHttpRequestEvent - Expected RBRACKET, got %s\n", p.curToken.Type)
		}
		return nil
	}

	return expression
}

func (p *Parser) parseContainsExpression() ast.Expression {

	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		// Left:      p.parseExpression(LOWEST)
	}

	if config.DebugMode {
		fmt.Printf("Start parseContainsExpression: %T\n", expression)
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	if config.DebugMode {
		fmt.Printf("End parseContainsExpression\n")
	}

	return expression
}

func (p *Parser) ParseIRule() *ast.IRuleNode {
	if config.DebugMode {
		fmt.Printf("DEBUG: Start ParseIRule\n")
	}
	irule := &ast.IRuleNode{}

	if !p.curTokenIs(token.WHEN) {
		return nil
	}

	irule.When = p.parseWhenNode()
	if irule.When == nil {
		return nil
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: End ParseIRule\n")
	}
	return irule
}

func (p *Parser) parseWhenNode() *ast.WhenNode {
	if config.DebugMode {
		fmt.Printf("DEBUG: Start parseWhenNode\n")
	}
	when := &ast.WhenNode{}

	if !p.expectPeek(token.HTTP_REQUEST) {
		if config.DebugMode {
			fmt.Printf("DEBUG: parseWhenNode - Expected HTTP_REQUEST, got %s\n", p.curToken.Type)
		}
		return nil
	}
	when.Event = p.curToken.Literal

	if !p.expectPeek(token.LBRACE) {
		if config.DebugMode {
			fmt.Printf("DEBUG: parseWhenNode - Expected LBRACE, got %s\n", p.curToken.Type)
		}
		return nil
	}

	when.Statements = p.parseBlockStatements()

	if config.DebugMode {
		fmt.Printf("DEBUG: End parseWhenNode\n")
	}
	return when
}

func (p *Parser) parseHttpExpression() ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: Start ParseHttpExpression\n")
	}

	expr := &ast.HttpExpression{Token: p.curToken}

	p.nextToken() // consume '['
	if !p.curTokenIs(token.HTTP_URI) {
		p.errors = append(p.errors, fmt.Sprintf("Expected HTTP::uri, got %s", p.curToken.Type))
		return nil
	}

	expr.Command = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if p.peekTokenIs(token.IDENT) {
		p.nextToken()
		expr.Method = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	}

	if !p.expectPeek(token.RBRACKET) {
		if config.DebugMode {
			fmt.Printf("DEBUG: parseHttpExpression - Expected RBRACKET, got %s\n", p.curToken.Type)
		}
		return nil
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: End ParseHttpExpression\n")
	}

	return expr
}

func (p *Parser) parseBlockStatements() []ast.Statement {
	if config.DebugMode {
		fmt.Printf("DEBUG: Start parseBlockStatementS (with an S)\n")
	}
	statements := []ast.Statement{}

	p.nextToken()

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			statements = append(statements, stmt)
		}
		p.nextToken()
	}

	if config.DebugMode {
		fmt.Printf("DEBUG:End parseBlockStatementS (with an S)\n")
	}

	return statements
}

func (p *Parser) parseBracketExpression() ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: Start parseBracketExpression\n")
	}

	expr := &ast.BracketExpression{Token: p.curToken}

	p.nextToken() // Advance past the `[` token

	if p.curTokenIs(token.HTTP_URI) {
		expr.Expression = p.parseHttpCommand()
	} else {
		expr.Expression = p.parseExpression(LOWEST)
	}

	// We expect a closing bracket `]` after the expression
	if !p.expectPeek(token.RBRACKET) {
		p.errors = append(p.errors, fmt.Sprintf("Expected closing bracket, got %s", p.peekToken.Type))
		return nil
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: End parseBracketExpression - %+v\n", expr)
	}

	return expr
}

func (p *Parser) parseHttpCommand() ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: Start parseHttpCommand\n")
	}
	expr := &ast.HttpExpression{Token: p.curToken}

	// if !p.expectPeek(token.IDENT) {
	// 	return nil
	// }
	expr.Command = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if config.DebugMode {
		fmt.Printf("DEBUG: End parseHttpCommand\n")
	}
	return expr
}

func (p *Parser) parseIfStatement() *ast.IfStatement {
	if config.DebugMode {
		fmt.Printf("DEBUG: Start parseIfStatement\n")
	}
	stmt := &ast.IfStatement{Token: p.curToken}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	p.nextToken() // consume '{'

	// Parse the condition
	var condition ast.Expression
	condition = p.parseExpression(LOWEST)

	// Check if there's a 'contains' part
	if p.peekTokenIs(token.CONTAINS) {
		p.nextToken() // Move to 'contains'
		containsExp := &ast.InfixExpression{
			Token:    p.curToken,
			Left:     condition,
			Operator: p.curToken.Literal,
		}
		p.nextToken() // Move past 'contains'
		containsExp.Right = p.parseExpression(LOWEST)
		condition = containsExp
	}

	stmt.Condition = condition

	if !p.expectPeek(token.RBRACE) {
		return nil
	}

	// parse consequence
	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	// p.nextToken() // Move past '{'
	stmt.Consequence = p.parseBlockStatement()
	if config.DebugMode {
		fmt.Printf("DEBUG: End parseIfStatement\n")
	}

	return stmt
}

func (p *Parser) parseWhenExpression() ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: Start parseWhenExpression\n")
	}
	expr := &ast.WhenExpression{Token: p.curToken}

	if !p.expectPeek(token.HTTP_REQUEST) {
		return nil
	}
	expr.Event = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	expr.Block = p.parseBlockStatement()

	if config.DebugMode {
		fmt.Printf("DEBUG: End parseWhenExpression\n")
	}

	return expr
}

func (p *Parser) parseBlockContents() []ast.Statement {
	statements := []ast.Statement{}

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			statements = append(statements, stmt)
		}
		p.nextToken()
	}

	return statements
}
