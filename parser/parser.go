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
	LOGICAL     // && or ||
	CONTAINS
)

var precedences = map[token.TokenType]int{
	token.EQ:          EQUALS,
	token.NOT_EQ:      EQUALS,
	token.LT:          LESSGREATER,
	token.GT:          LESSGREATER,
	token.PLUS:        SUM,
	token.MINUS:       SUM,
	token.SLASH:       PRODUCT,
	token.ASTERISK:    PRODUCT,
	token.LPAREN:      CALL,
	token.AND:         LOGICAL,
	token.OR:          LOGICAL,
	token.CONTAINS:    CONTAINS,
	token.STARTS_WITH: EQUALS,
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

	braceCount int
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
	p.registerPrefix(token.ASTERISK, p.parsePrefixExpression)
	p.registerPrefix(token.BANG, p.parsePrefixExpression)
	p.registerPrefix(token.DOLLAR, p.parseVariableOrArrayAccess)
	p.registerPrefix(token.FALSE, p.parseBoolean)
	p.registerPrefix(token.IDENT, p.parseIdentifier)
	p.registerPrefix(token.LBRACE, p.parseHashLiteral)
	p.registerPrefix(token.LBRACKET, p.parseArrayLiteral)
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(token.RBRACKET, p.parseArrayLiteral)
	p.registerPrefix(token.RPAREN, p.parseGroupedExpression)
	p.registerPrefix(token.MINUS, p.parsePrefixExpression)
	p.registerPrefix(token.NUMBER, p.parseNumberLiteral)
	p.registerPrefix(token.SET, p.parseSetExpression)
	p.registerPrefix(token.STRING, p.parseStringLiteral)
	p.registerPrefix(token.TRUE, p.parseBoolean)
	p.registerPrefix(token.WHEN, p.parseWhenExpression)
	p.registerPrefix(token.HTTP_COMMAND, p.parseHttpCommand)
	p.registerPrefix(token.HTTP_HEADER, p.parseHttpCommand)
	p.registerPrefix(token.HTTP_METHOD, p.parseHttpCommand)
	p.registerPrefix(token.HTTP_PATH, p.parseHttpCommand)
	p.registerPrefix(token.HTTP_QUERY, p.parseHttpCommand)
	p.registerPrefix(token.HTTP_REDIRECT, p.parseHttpCommand)
	p.registerPrefix(token.HTTP_RESPOND, p.parseHttpCommand)
	p.registerPrefix(token.HTTP_URI, p.parseHttpCommand)
	p.registerPrefix(token.SWITCH, p.parseSwitchExpression)
	p.registerPrefix(token.DEFAULT, p.parseDefaultExpression)

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
	p.registerInfix(token.AND, p.parseInfixExpression)
	p.registerInfix(token.OR, p.parseInfixExpression)

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
	p.braceCount = 0

	for !p.curTokenIs(token.EOF) {
		if config.DebugMode {
			fmt.Printf("DEBUG: Current token: %s, Brace count: %d\n", p.curToken.Type, p.braceCount)
		}
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		} else {
			fmt.Printf("Failed to parse statement at token: %+v\n", p.curToken) // Debug print
		}

		p.nextToken()
	}

	// Handle any remaining open blocks at EOF
	for p.braceCount > 0 {
		p.braceCount--
		if config.DebugMode {
			fmt.Printf("DEBUG: Closing unclosed block at EOF. Brace count: %d\n", p.braceCount)
		}
	}

	if p.braceCount != 0 {
		p.errors = append(p.errors, fmt.Sprintf("Mismatched braces. Final brace count: %d", p.braceCount))
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
		return stmt
	case token.RETURN:
		if config.DebugMode {
			fmt.Printf("DEBUG: Calling parseReturnStatement\n")
		}
		stmt = p.parseReturnStatement()
	case token.IDENT:
		if p.curToken.Literal == "pool" {
			return p.parsePoolStatement()
		}
		return p.parseExpressionStatement()
	case token.WHEN:
		stmt = &ast.ExpressionStatement{
			Token:      p.curToken,
			Expression: p.parseWhenExpression(),
		}
	case token.IF:
		stmt = p.parseIfStatement()
	case token.ELSE:
		stmt = p.parseIfStatement()
	case token.LBRACE:
		stmt = p.parseBlockStatement()
	case token.SWITCH:
		stmt = p.parseSwitchStatement()
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
	if config.DebugMode {
		fmt.Printf("DEBUG: parseSetStatement Start\n")
	}
	stmt := &ast.SetStatement{Token: p.curToken}

	p.nextToken() // Move past 'set'

	// Parse the target (can be an identifier or an expression)
	stmt.Name = p.parseExpression(LOWEST)

	if stmt.Name == nil {
		p.errors = append(p.errors, "ERROR: parseSetStatement: Expected a name for set statement")
		return nil
	}

	// Parse the value
	if !p.peekTokenIs(token.EOF) {
		p.nextToken() // Move to the value
		stmt.Value = p.parseExpression(LOWEST)
	}

	// Consume any remaining tokens until EOF or semicolon
	// for !p.curTokenIs(token.EOF) && !p.curTokenIs(token.SEMICOLON) {
	// 	p.nextToken()
	// }

	if config.DebugMode {
		fmt.Printf("DEBUG: Set statement name type: %T\n", stmt.Name)
		fmt.Printf("DEBUG: Set statement name: %v\n", stmt.Name)
		fmt.Printf("DEBUG: Set statement value type: %T\n", stmt.Value)
		fmt.Printf("DEBUG: Set statement value: %v\n", stmt.Value)
		fmt.Printf("DEBUG: parseSetStatement End\n")
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
		if p.curTokenIs(token.EOF) {
			return nil // Return nil for EOF
		}
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(token.SEMICOLON) && !p.peekTokenIs(token.RBRACE) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			if p.peekTokenIs(token.STARTS_WITH) || p.peekTokenIs(token.EQ) {
				p.nextToken()
				leftExp = p.parseInfixExpression(leftExp)
			} else {
				return leftExp
			}
		}
		p.nextToken()
		leftExp = infix(leftExp)
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

	p.braceCount++
	p.nextToken() // consume opening brace

	if config.DebugMode {
		fmt.Printf("DEBUG: Entering block statement. Brace count: %d\n", p.braceCount)
	}

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
			if config.DebugMode {
				fmt.Printf("DEBUG: Added statement to block, type: %T\n", stmt)
			}
		} else if config.DebugMode {
			fmt.Printf("DEBUG: Failed to parse statement at token: %+v\n", p.curToken)
		}

		p.nextToken()
	}

	if p.curTokenIs(token.RBRACE) {
		p.braceCount--
		if config.DebugMode {
			fmt.Printf("DEBUG: Exiting block statement. Brace count: %d\n", p.braceCount)
		}
	} else if p.curTokenIs(token.EOF) {
		p.braceCount--
		if config.DebugMode {
			fmt.Printf("DEBUG: Reached EOF while parsing block statement. Brace count: %d\n", p.braceCount)
		}
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: End parseBlockStatement, statements: %d\n", len(block.Statements))
	}

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

	// Check if it's an HTTP command
	// if p.curTokenIs(token.HTTP_URI) || p.curTokenIs(token.HTTP_METHOD) || p.curTokenIs(token.HTTP_HEADER) {
	// 	httpExpr := p.parseHttpCommand()
	// 	if httpExpr != nil {
	// 		list = append(list, httpExpr)
	// 	}
	// 	return list
	// }

	// Parse expressions until encountering the end token
	list = append(list, p.parseExpression(LOWEST))

	for p.peekTokenIs(token.COMMA) {
		p.nextToken() // Consume the comma
		p.nextToken() // Move to the next expression
		list = append(list, p.parseExpression(LOWEST))
		if config.DebugMode {
			fmt.Printf("DEBUG: parseExpressionList loop. list = %v\n", list)
		}
	}

	// Ensure that the list is terminated with the end token
	if !p.expectPeek(end) {
		if config.DebugMode {
			fmt.Printf("DEBUG: parseExpressionList - Expected end, got %s\n", p.curToken.Type)
		}
		return nil
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseExpressionList End\n")
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

func (p *Parser) parseSetExpression() ast.Expression {
	stmt := &ast.SetStatement{Token: p.curToken}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseSetExpression - Starting\n")
	}

	p.nextToken() // move past 'set'

	// Parse the target (can be an identifier or an expression)
	stmt.Name = p.parseExpression(LOWEST)

	if stmt.Name == nil {
		return nil
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseSetExpression - Name: %s\n", stmt.Name)
	}

	// Parse the value
	if !p.peekTokenIs(token.EOF) {
		p.nextToken() // Move to the value
		stmt.Value = p.parseExpression(LOWEST)
	}

	// Consume any remaining tokens until EOF or semicolon
	for !p.curTokenIs(token.EOF) && !p.curTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseSetExpression - Value parsed: %T\n", stmt.Value)
		fmt.Printf("DEBUG: parseSetExpression - Completed: %v\n", stmt)
	}

	return stmt
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseArrayLiteral Start\n")
	}

	array := &ast.ArrayLiteral{Token: p.curToken}

	// Check if we have double brackets
	doubleBracket := p.peekTokenIs(token.LBRACKET)
	if doubleBracket {
		p.nextToken() // Consume the second '['
	}

	// Check if the next token is an HTTP-related token
	if p.peekTokenIs(token.HTTP_URI) || p.peekTokenIs(token.HTTP_METHOD) || p.peekTokenIs(token.HTTP_HEADER) {
		p.nextToken() // Move to the HTTP token
		httpExpr := p.parseHttpCommand()
		if httpExpr != nil {
			array.Elements = []ast.Expression{httpExpr}
		}

		// Expect the closing brackets
		if !p.expectPeek(token.RBRACKET) {
			return nil
		}
		if doubleBracket {
			if !p.expectPeek(token.RBRACKET) {
				return nil
			}
		}
	} else {
		array.Elements = p.parseExpressionList(token.RBRACKET)
		if doubleBracket {
			if !p.expectPeek(token.RBRACKET) {
				return nil
			}
		}
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseArrayLiteral End. Array: %s\n", array)
	}
	return array
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

// func (p *Parser) parseBracketExpression() ast.Expression {
// 	if config.DebugMode {
// 		fmt.Printf("DEBUG: Start parseBracketExpression\n")
// 	}
//
// 	expr := &ast.BracketExpression{Token: p.curToken}
//
// 	p.nextToken() // Advance past the `[` token
//
// 	if p.curTokenIs(token.HTTP_URI) {
// 		expr.Expression = p.parseHttpCommand()
// 	} else {
// 		expr.Expression = p.parseExpression(LOWEST)
// 	}
//
// 	// We expect a closing bracket `]` after the expression
// 	if !p.expectPeek(token.RBRACKET) {
// 		p.errors = append(p.errors, fmt.Sprintf("Expected closing bracket, got %s", p.peekToken.Type))
// 		return nil
// 	}
//
// 	if config.DebugMode {
// 		fmt.Printf("DEBUG: End parseBracketExpression - %+v\n", expr)
// 	}
//
// 	return expr
// }

func (p *Parser) parseHttpCommand() ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: Start parseHttpCommand\n")
	}
	expr := &ast.HttpExpression{Token: p.curToken}

	// Check if we're starting with a '[' or '[['
	if p.curTokenIs(token.LBRACKET) {
		if p.peekTokenIs(token.LBRACKET) {
			p.nextToken() // consume second '['
		}
		p.nextToken() // consume '['
	}

	// Parse the HTTP command (e.g., HTTP::header)
	expr.Command = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if expr.Command.String() == "HTTP::header" {
		if !p.expectPeek(token.STRING) {
			return nil
		}

		expr.Argument = &ast.StringLiteral{
			Token: p.curToken,
			Value: p.curToken.Literal,
		}
		if config.DebugMode {
			fmt.Printf("DEBUG: parseHttpCommand successfully parsed http::header arg %+v\n", expr.Argument)
		}
	}

	// If we started with '[' or '[[', expect closing ']' or ']]'
	if p.curToken.Type == token.LBRACKET {
		if !p.expectPeek(token.RBRACKET) {
			p.errors = append(p.errors, fmt.Sprintf("Expected closing bracket after HTTP command, got %s", p.peekToken.Type))
			return nil
		}
		if p.peekTokenIs(token.RBRACKET) {
			p.nextToken() // consume second ']'
		}
	}

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

	// parse the condition
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	p.nextToken() // consume '{'

	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RBRACE) {
		return nil
	}

	// Parse consequence
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	stmt.Consequence = p.parseBlockStatement()

	// Parse else clause if it exists
	if p.peekTokenIs(token.ELSE) {
		p.nextToken() // consume 'else'

		if !p.expectPeek(token.LBRACE) {
			return nil
		}
		stmt.Alternative = p.parseBlockStatement()
	}

	// Handle EOF
	if p.curTokenIs(token.EOF) {
		if config.DebugMode {
			fmt.Printf("DEBUG: Reached EOF while parsing if statement. Brace count: %d\n", p.braceCount)
		}
	}

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

func (p *Parser) parsePoolStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{Token: p.curToken}
	if config.DebugMode {
		fmt.Printf("DEBUG: Start parsePoolStatement\n")
	}

	callExpr := &ast.CallExpression{
		Token:    p.curToken,
		Function: &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal},
	}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	argument := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	callExpr.Arguments = append(callExpr.Arguments, argument)

	stmt.Expression = callExpr
	if config.DebugMode {
		fmt.Printf("DEBUG: End parsePoolStatement\n")
	}
	return stmt
}

func (p *Parser) parseSwitchStatement() *ast.SwitchStatement {
	if config.DebugMode {
		fmt.Printf("DEBUG: Start parseSwitchStatement\n")
	}
	switchStmt := &ast.SwitchStatement{Token: p.curToken}

	// Parse switch options and value
	p.nextToken() // move past 'switch'
	if p.curTokenIs(token.MINUS) {
		switchStmt.Token.Literal += " " + p.curToken.Literal
		p.nextToken() // move past option
		switchStmt.Token.Literal += " " + p.curToken.Literal
		p.nextToken() // move past option value
	}
	switchStmt.Value = p.parseExpression(LOWEST)

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	switchStmt.Cases = []*ast.CaseStatement{}

	p.nextToken() // Move past the opening brace

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		if config.DebugMode {
			fmt.Printf("DEBUG: Switch loop - Current token: %s, Literal: %s\n", p.curToken.Type, p.curToken.Literal)
		}

		if p.curTokenIs(token.DEFAULT) {
			switchStmt.Default = p.parseDefaultCase()
		} else if p.curTokenIs(token.STRING) {
			caseStmt := p.parseCaseStatement()
			if caseStmt != nil {
				switchStmt.Cases = append(switchStmt.Cases, caseStmt)
				if config.DebugMode {
					fmt.Printf("DEBUG: Added case, total cases: %d\n", len(switchStmt.Cases))
				}
			}
		} else {
			// Skip any unexpected tokens
			p.nextToken()
		}
	}

	if !p.curTokenIs(token.RBRACE) {
		p.peekError(token.RBRACE)
		return nil
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: End parseSwitchStatement, total cases: %d\n", len(switchStmt.Cases))
	}
	return switchStmt
}

func (p *Parser) parseSwitchExpression() ast.Expression {
	return p.parseSwitchStatement()
}

func (p *Parser) parseDefaultExpression() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseCaseStatement() *ast.CaseStatement {
	if config.DebugMode {
		fmt.Printf("DEBUG: Start parseCaseStatement\n")
	}
	caseStmt := &ast.CaseStatement{Token: p.curToken}

	caseStmt.Value = p.parseExpression(LOWEST)

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	caseStmt.Consequence = p.parseBlockStatement()

	// Advance to the next token after the closing brace
	p.nextToken()

	if config.DebugMode {
		fmt.Printf("DEBUG: End parseCaseStatement\n")
	}

	return caseStmt
}

func (p *Parser) parseDefaultCase() *ast.CaseStatement {
	if config.DebugMode {
		fmt.Printf("DEBUG: Start parseDefaultCase\n")
	}
	defaultCase := &ast.CaseStatement{Token: p.curToken, Value: nil}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	defaultCase.Consequence = p.parseBlockStatement()

	if config.DebugMode {
		fmt.Printf("DEBUG: End parseDefaultCase\n")
	}
	return defaultCase
}
