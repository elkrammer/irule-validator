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

var validWhenEvents = []token.TokenType{
	token.HTTP_REQUEST,
	token.HTTP_RESPONSE,
	token.LB_SELECTED,
	token.CLIENT_ACCEPTED,
	token.SERVER_CONNECTED,
	token.CLIENTSSL_HANDSHAKE,
	token.SERVERSSL_HANDSHAKE,
	token.TCP_REQUEST,
	token.TCP_RESPONSE,
	token.USER_REQUEST,
	token.USER_RESPONSE,
	token.RULE_INIT,
	token.DNS_REQUEST,
	token.DNS_RESPONSE,
	token.SSL_CLIENTHELLO,
	token.SSL_SERVERHELLO,
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

	// http commands
	p.registerPrefix(token.HTTP_HEADER, p.parseHttpCommand)
	p.registerPrefix(token.HTTP_METHOD, p.parseHttpCommand)
	p.registerPrefix(token.HTTP_PATH, p.parseHttpCommand)
	p.registerPrefix(token.HTTP_QUERY, p.parseHttpCommand)
	p.registerPrefix(token.HTTP_REDIRECT, p.parseHttpCommand)
	p.registerPrefix(token.HTTP_RESPOND, p.parseHttpCommand)
	p.registerPrefix(token.HTTP_URI, p.parseHttpCommand)
	p.registerPrefix(token.HTTP_HOST, p.parseHttpCommand)

	// load balancer commands
	p.registerPrefix(token.LB_SELECTED, p.parseLoadBalancerCommand)
	p.registerPrefix(token.LB_FAILED, p.parseLoadBalancerCommand)
	p.registerPrefix(token.LB_QUEUED, p.parseLoadBalancerCommand)
	p.registerPrefix(token.LB_COMPLETED, p.parseLoadBalancerCommand)
	p.registerPrefix(token.LB_MODE, p.parseLoadBalancerCommand)
	p.registerPrefix(token.LB_SELECT, p.parseLoadBalancerCommand)
	p.registerPrefix(token.LB_RESELECT, p.parseLoadBalancerCommand)
	p.registerPrefix(token.LB_DETACH, p.parseLoadBalancerCommand)
	p.registerPrefix(token.LB_SERVER, p.parseLoadBalancerCommand)
	p.registerPrefix(token.LB_POOL, p.parseLoadBalancerCommand)
	p.registerPrefix(token.LB_STATUS, p.parseLoadBalancerCommand)
	p.registerPrefix(token.LB_ALIVE, p.parseLoadBalancerCommand)
	p.registerPrefix(token.LB_PERSIST, p.parseLoadBalancerCommand)
	p.registerPrefix(token.LB_METHOD, p.parseLoadBalancerCommand)
	p.registerPrefix(token.LB_SCORE, p.parseLoadBalancerCommand)
	p.registerPrefix(token.LB_PRIORITY, p.parseLoadBalancerCommand)
	p.registerPrefix(token.LB_CONNECT, p.parseLoadBalancerCommand)
	p.registerPrefix(token.LB_BIAS, p.parseLoadBalancerCommand)
	p.registerPrefix(token.LB_SNAT, p.parseLoadBalancerCommand)
	p.registerPrefix(token.LB_LIMIT, p.parseLoadBalancerCommand)
	p.registerPrefix(token.LB_CLASS, p.parseLoadBalancerCommand)

	// SSL Commands
	p.registerPrefix(token.SSL_CIPHER, p.parseSSLCommand)
	p.registerPrefix(token.SSL_CIPHER_BITS, p.parseSSLCommand)
	p.registerPrefix(token.SSL_CLIENTHELLO, p.parseSSLCommand)
	p.registerPrefix(token.SSL_SERVERHELLO, p.parseSSLCommand)

	p.registerPrefix(token.SWITCH, p.parseSwitchExpression)
	p.registerPrefix(token.DEFAULT, p.parseDefaultExpression)
	p.registerPrefix(token.IP_CLIENT_ADDR, p.parseIpExpression)
	p.registerPrefix(token.IP_SERVER_ADDR, p.parseIpExpression)
	p.registerPrefix(token.IP_ADDRESS, p.parseIpAddressLiteral)

	p.infixParseFns = make(map[token.TokenType]infixParseFn)
	p.registerInfix(token.ASTERISK, p.parseInfixExpression)
	p.registerInfix(token.EQ, p.parseInfixExpression)
	p.registerInfix(token.LBRACKET, p.parseIndexExpression)
	p.registerInfix(token.LPAREN, p.parseCallExpression)
	p.registerInfix(token.GT, p.parseInfixExpression)
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

	if stmt == nil {
		p.errors = append(p.errors, fmt.Sprintf("ERROR: parseStatement - Unexpected token: %s", p.curToken.Literal))
		p.nextToken() // Skip problematic token
		return nil
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseStatement End - Parsed: %T\n", stmt)
	}
	return stmt
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	if config.DebugMode {
		fmt.Printf("DEBUG: Start parseReturnStatement\n")
	}

	stmt := &ast.ReturnStatement{Token: p.curToken}

	p.nextToken() // consume the 'return' token

	// Check if the next token is a semicolon or a closing brace
	// If so, it's a bare return statement
	if p.curTokenIs(token.SEMICOLON) || p.curTokenIs(token.RBRACE) {
		return stmt
	}

	stmt.ReturnValue = p.parseExpression(LOWEST)

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
		fmt.Printf("DEBUG: parseExpressionStatement Start, current token: %s\n", p.curToken.Type)
	}
	stmt := &ast.ExpressionStatement{Token: p.curToken}

	if p.curTokenIs(token.IDENT) && p.curToken.Literal == "pool" {
		stmt.Expression = p.parsePoolStatement()
	} else {
		stmt.Expression = p.parseExpression(LOWEST)
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseExpressionStatement - Parsed expression: %T\n", stmt.Expression)
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseExpressionStatement End, expression type: %T\n", stmt.Expression)
	}

	return stmt
}

func (p *Parser) parseExpression(precedence int) ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseExpression Start - Current token: %s, Precedence: %d\n", p.curToken.Type, precedence)
	}

	if p.curTokenIs(token.IDENT) && p.curToken.Literal == "string" {
		return p.parseStringOperation()
	}

	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		if config.DebugMode {
			fmt.Printf("ERROR: parseExpression - no prefix parse function for %s\n", p.curToken.Literal)
		}

		// Handle closing braces and brackets gracefully
		if p.curTokenIs(token.RBRACE) || p.curTokenIs(token.RBRACKET) || p.curTokenIs(token.EOF) {
			return nil
		}

		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}

	leftExp := prefix()

	for !p.peekTokenIs(token.SEMICOLON) && !p.peekTokenIs(token.EOF) && precedence < p.peekPrecedence() {
		if config.DebugMode {
			fmt.Printf("DEBUG: parseExpression loop, current: %s, peek: %s, precedence: %d, peek precedence: %d\n",
				p.curToken.Type, p.peekToken.Type, precedence, p.peekPrecedence())
		}

		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			// Check if we've reached the end of the expression
			if p.peekTokenIs(token.RBRACE) || p.peekTokenIs(token.RBRACKET) {
				break
			}
			return leftExp
		}

		p.nextToken()
		if config.DebugMode {
			fmt.Printf("DEBUG: Parsing infix expression, operator: %s\n", p.curToken.Literal)
		}
		leftExp = infix(leftExp)
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseExpression End, result type: %T\n", leftExp)
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
				fmt.Printf("ERROR: parseGroupedExpression - Expected RPAREN, got %s\n", p.curToken.Type)
			}
			return nil
		}
	} else if p.peekTokenIs(token.RBRACKET) {
		if !p.expectPeek(token.RBRACKET) {
			if config.DebugMode {
				fmt.Printf("ERROR: parseGroupedExpression - Expected RBRACKET, got %s\n", p.curToken.Type)
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
		fmt.Printf("DEBUG: parseBlockStatement Start\n")
	}
	block := &ast.BlockStatement{Token: p.curToken}
	block.Statements = []ast.Statement{}

	p.braceCount++
	p.nextToken() // consume opening brace

	if config.DebugMode {
		fmt.Printf("DEBUG: Entering block statement. Brace count: %d\n", p.braceCount)
	}

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		if config.DebugMode {
			fmt.Printf("DEBUG: parseBlockStatement loop. Current token: %s\n", p.curToken.Literal)
		}
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
			if config.DebugMode {
				fmt.Printf("DEBUG: parseBlockStatement: Added statement to block, type: %T\n", stmt)
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
		fmt.Printf("DEBUG: parseBlockStatement End, statements: %d\n", len(block.Statements))
	}

	return block
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	fmt.Printf("DEBUG: Parsing index expression\n")

	exp := &ast.IndexExpression{Token: p.curToken, Left: left}

	if !p.expectPeek(token.LPAREN) {
		if config.DebugMode {
			fmt.Printf("ERROR: parseIndexExpression - Expected LPAREN, got %s\n", p.curToken.Type)
		}
		return nil
	}

	p.nextToken() // move past '(' token
	exp.Index = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		if config.DebugMode {
			fmt.Printf("ERROR: parseIndexExpression - Expected RPAREN, got %s\n", p.curToken.Type)
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
				fmt.Printf("ERROR: parseHashLiteral - Expected STRING, got %s\n", p.curToken.Type)
			}
			return nil
		}
		key := &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}

		// Parse value
		if !p.expectPeek(token.STRING) {
			if config.DebugMode {
				fmt.Printf("ERROR: parseHashLiteral - Expected STRING, got %s\n", p.curToken.Type)
			}
			return nil
		}
		value := &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}

		hash.Pairs[*key] = value

		if !p.peekTokenIs(token.RBRACE) && !p.expectPeek(token.COMMA) {
			if config.DebugMode {
				fmt.Printf("ERROR: parseHashLiteral - Expected COMMA for Peek token, got %s\n", p.curToken.Type)
			}
			return nil
		}
	}

	if !p.expectPeek(token.RBRACE) {
		if config.DebugMode {
			fmt.Printf("ERROR: parseHashLiteral - Expected RBRACE, got %s\n", p.curToken.Type)
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
	for !p.curTokenIs(end) && !p.curTokenIs(token.EOF) {
		expr := p.parseExpression(LOWEST)
		if expr != nil {
			list = append(list, expr)
		}

		if p.peekTokenIs(token.COMMA) {
			p.nextToken() // Consume the comma
		} else if !p.peekTokenIs(end) {
			if config.DebugMode {
				fmt.Printf("ERROR: parseExpressionList - Expected comma or end, got %s\n", p.peekToken.Type)
			}
			return nil
		}

		p.nextToken() // Move to the next token
	}

	// Ensure that the list is terminated with the end token
	if !p.expectPeek(end) {
		if config.DebugMode {
			fmt.Printf("ERROR: parseExpressionList - Expected end, got %s\n", p.curToken.Type)
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
	if config.DebugMode {
		fmt.Printf("DEBUG: parseInfixExpression Start - Operator: %s\n", p.curToken.Literal)
	}
	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	if config.DebugMode {
		fmt.Printf("DEBUG: parseInfixExpression End - Operator: %s\n", expression.Operator)
	}

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
		fmt.Printf("DEBUG: parseArrayLiteral Start. Current token: %s\n", p.curToken.Literal)
	}

	array := &ast.ArrayLiteral{Token: p.curToken}
	array.Elements = []ast.Expression{}

	p.nextToken() // Move past the opening bracket

	// Handle nested array or expression
	for !p.curTokenIs(token.RBRACKET) && !p.curTokenIs(token.EOF) {
		var expr ast.Expression

		if p.curTokenIs(token.IDENT) && p.curToken.Literal == "string" {
			expr = p.parseStringOperation()
		} else if p.curTokenIs(token.LBRACKET) {
			expr = p.parseArrayLiteral() // Handle nested array
		} else if p.isHttpKeyword(p.curToken.Type) {
			expr = p.parseHttpCommand()
		} else if p.isSSLKeyword(p.curToken.Type) {
			expr = p.parseSSLCommand()
		} else if p.isLbKeyword(p.curToken.Type) {
			expr = p.parseLoadBalancerCommand()
		} else {
			expr = p.parseExpression(LOWEST)
		}

		if expr != nil {
			array.Elements = append(array.Elements, expr)
			if config.DebugMode {
				fmt.Printf("DEBUG: parseArrayLiteral - Added element: %T\n", expr)
			}
		} else {
			if config.DebugMode {
				fmt.Printf("DEBUG: parseArrayLiteral Error - Failed to parse element\n")
			}
			return nil
		}

		// Break if we've reached the end of the array
		if p.peekTokenIs(token.RBRACKET) {
			break
		}

		// Move to next token if it's not the closing bracket
		if !p.curTokenIs(token.RBRACKET) {
			p.nextToken()
		}
	}

	if !p.expectPeek(token.RBRACKET) {
		p.errors = append(p.errors, fmt.Sprintf("ERROR: parseArrayLiteral - Expected closing bracket, got %s", p.curToken.Literal))
		if config.DebugMode {
			fmt.Printf("ERROR: parseArrayLiteral Error - Expected closing bracket, got %s\n", p.curToken.Literal)
		}
		return nil
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseArrayLiteral End. Array elements: %d\n", len(array.Elements))
		for i, elem := range array.Elements {
			fmt.Printf("DEBUG: parseArrayLiteral - Element %d: %T\n", i, elem)
		}
	}
	return array
}

func (p *Parser) parseSSLCommand() ast.Expression {
	command := &ast.SSLExpression{Token: p.curToken}
	var commandParts []string

	for !p.peekTokenIs(token.RBRACKET) && !p.peekTokenIs(token.EOF) {
		if config.DebugMode {
			fmt.Printf("DEBUG: parseArrayLiteral loop. Current token: %s\n", p.curToken.Literal)
		}
		commandParts = append(commandParts, p.curToken.Literal)
		p.nextToken()
	}

	if !p.curTokenIs(token.RBRACKET) {
		if config.DebugMode {
			fmt.Printf("DEBUG: parseArrayLiteral Error: Expected ], got %s\n", p.curToken.Literal)
		}
		p.errors = append(p.errors, "Expected closing bracket in array literal")
		return nil
	}

	command.Command = &ast.Identifier{Token: command.Token, Value: strings.Join(commandParts, " ")}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseSSLCommand Command: %s\n", command.Command.Value)
	}

	return command
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
				fmt.Printf("ERROR: parseVariableOrArrayAccess:  Expected RPAREN, got %s\n", p.curToken.Type)
			}
			return nil
		}
		return &ast.IndexExpression{Left: varExp, Index: index}
	}

	return varExp
}

func (p *Parser) ParseIRule() *ast.IRuleNode {
	if config.DebugMode {
		fmt.Printf("DEBUG: ParseIRule Start\n")
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
		fmt.Printf("DEBUG: ParseIRule End\n")
	}
	return irule
}

func (p *Parser) parseWhenNode() *ast.WhenNode {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseWhenNode Start\n")
	}
	when := &ast.WhenNode{}

	if !p.expectPeek(token.HTTP_REQUEST) || !p.peekTokenIs(token.LB_SELECTED) {
		if config.DebugMode {
			fmt.Printf("ERROR: parseWhenNode - Expected HTTP_REQUEST or LB_SELECTED, got %s\n", p.curToken.Type)
		}
		return nil
	}
	when.Event = p.curToken.Literal

	if !p.expectPeek(token.LBRACE) {
		if config.DebugMode {
			fmt.Printf("ERROR: parseWhenNode - Expected LBRACE, got %s\n", p.curToken.Type)
		}
		return nil
	}

	when.Statements = p.parseBlockStatements()

	if config.DebugMode {
		fmt.Printf("DEBUG: parseWhenNode End\n")
	}
	return when
}

func (p *Parser) parseBlockStatements() []ast.Statement {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseBlockStatementS (with an S) Start\n")
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

func (p *Parser) parseHttpCommand() ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseHttpCommand Start\n")
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
		// Parse the header name, which may contain multiple words
		var headerParts []string
		for p.peekTokenIs(token.IDENT) || p.peekTokenIs(token.MINUS) || p.peekTokenIs(token.STRING) {
			p.nextToken()
			headerParts = append(headerParts, p.curToken.Literal)
		}

		if len(headerParts) > 0 {
			headerName := strings.Join(headerParts, "")
			expr.Argument = &ast.StringLiteral{
				Token: token.Token{Type: token.STRING, Literal: headerName},
				Value: headerName,
			}
			if config.DebugMode {
				fmt.Printf("DEBUG: parseHttpCommand successfully parsed http::header arg %+v\n", expr.Argument)
			}
		}
	}

	// If we started with '[' or '[[', expect closing ']' or ']]'
	if p.curToken.Type == token.LBRACKET {
		if !p.expectPeek(token.RBRACKET) {
			p.errors = append(p.errors, fmt.Sprintf("ERROR: parseHttpCommand: Expected closing bracket after HTTP command, got %s", p.peekToken.Type))
			return nil
		}
		if p.peekTokenIs(token.RBRACKET) {
			p.nextToken() // consume second ']'
		}
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseHttpCommand End\n")
	}
	return expr
}

func (p *Parser) parseIfStatement() *ast.IfStatement {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseIfStatement Start\n")
	}
	stmt := &ast.IfStatement{Token: p.curToken}

	// Expect '{'
	if !p.expectPeek(token.LBRACE) {
		p.errors = append(p.errors, fmt.Sprintf("ERROR: parseIfStatement: Expected {, got %s", p.curToken.Literal))
		return nil
	}

	p.nextToken() // consume '{'

	// Optionally consume '('
	if p.peekTokenIs(token.LPAREN) {
		p.nextToken()
	}

	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RBRACE) {
		p.errors = append(p.errors, fmt.Sprintf("ERROR: parseIfStatement Condition Expected }, got %s", p.curToken.Literal))
		return nil
	}

	// Expect '{' for consequence block
	if !p.expectPeek(token.LBRACE) {
		p.errors = append(p.errors, fmt.Sprintf("ERROR: parseIfStatement Consequence Expected {, got %s", p.curToken.Literal))
		return nil
	}
	stmt.Consequence = p.parseBlockStatement()

	// Parse else clause if it exists
	if p.peekTokenIs(token.ELSE) {
		p.nextToken() // consume 'else'

		if !p.expectPeek(token.LBRACE) {
			p.errors = append(p.errors, fmt.Sprintf("ERROR: parseIfStatement ELSE Expected {, got %s", p.curToken.Literal))
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
		fmt.Printf("DEBUG: parseIfStatement End\n")
	}

	return stmt
}

func (p *Parser) parseWhenExpression() ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseWhenExpression Start\n")
	}
	expr := &ast.WhenExpression{Token: p.curToken}

	// Check if the next token is a valid expression token
	if p.isValidWhenEvent(token.TokenType(p.peekToken.Literal)) {
		// if p.peekTokenIs(token.HTTP_REQUEST) || p.peekTokenIs(token.LB_SELECTED) {
		p.nextToken() // Advance to the event token
	} else {
		p.errors = append(p.errors, "ERROR: parseWhenExpression - Expected HTTP_REQUEST or LB_SELECTED")
		return nil
	}

	expr.Event = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.LBRACE) {
		p.errors = append(p.errors, "ERROR: parseWhenExpression - expected LBRACE")
		return nil
	}

	expr.Block = p.parseBlockStatement()

	if config.DebugMode {
		fmt.Printf("DEBUG: parseWhenExpression End\n")
	}

	return expr
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
		p.errors = append(p.errors, "ERROR: parseSwitchStatement expected LBRACE")
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
		if config.DebugMode {
			fmt.Printf("ERROR: parseSwitchStatement expected RBRACE. Got=%s\n", p.curToken.Literal)
		}
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
		p.errors = append(p.errors, "ERROR: parseCaseStatement expected LBRACE")
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
		p.errors = append(p.errors, "ERROR: parseDefaultCase expected LBRACE")
		return nil
	}

	defaultCase.Consequence = p.parseBlockStatement()

	if config.DebugMode {
		fmt.Printf("DEBUG: End parseDefaultCase\n")
	}
	return defaultCase
}

func (p *Parser) parseIpExpression() ast.Expression {
	expression := &ast.IpExpression{Token: p.curToken}

	switch p.curToken.Type {
	case token.IP_CLIENT_ADDR:
		expression.Function = "client_addr"
	case token.IP_SERVER_ADDR:
		expression.Function = "server_addr"
	default:
		p.errors = append(p.errors, fmt.Sprintf("Unexpected IP token: %s", p.curToken.Literal))
		return nil
	}

	return expression
}

func (p *Parser) parseIpAddressLiteral() ast.Expression {
	return &ast.IpAddressLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseLoadBalancerCommand() ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: Start parseLoadBalancerCommand\n")
	}

	command := &ast.LoadBalancerExpression{Token: p.curToken}
	var commandParts []string

	for !p.curTokenIs(token.RBRACKET) && !p.curTokenIs(token.EOF) {
		if p.curTokenIs(token.LBRACKET) {
			// Handle nested command
			p.nextToken() // consume '['
			nestedCommand := p.parseLoadBalancerCommand()
			if nestedExpr, ok := nestedCommand.(*ast.LoadBalancerExpression); ok {
				commandParts = append(commandParts, "["+nestedExpr.Command.Value+"]")
			}
		} else {
			commandParts = append(commandParts, p.curToken.Literal)
		}

		if config.DebugMode {
			fmt.Printf("DEBUG: parseLoadBalancerCommand Adding to command %s\n", p.curToken.Literal)
		}

		// Stop parsing if we encounter an 'if' statement or other control structures
		if p.peekTokenIs(token.IF) || p.peekTokenIs(token.LBRACE) {
			break
		}

		p.nextToken()

		// Break if we've reached the end of this command
		if p.curTokenIs(token.RBRACKET) {
			break
		}
	}

	// Combine all parts into a single command string
	command.Command = &ast.Identifier{Token: command.Token, Value: strings.Join(commandParts, " ")}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseLoadBalancerCommand End. Command: %v\n", command.Command.Value)
	}

	return command
}

// Helper function to check if a token is an HTTP keyword
func (p *Parser) isHttpKeyword(tokenType token.TokenType) bool {
	for _, httpTokenType := range lexer.HttpKeywords {
		if tokenType == httpTokenType {
			return true
		}
	}
	return false
}

// Helper function to check if a token is an LB keyword
func (p *Parser) isLbKeyword(tokenType token.TokenType) bool {
	for _, lbTokenType := range lexer.LbKeywords {
		if tokenType == lbTokenType {
			return true
		}
	}
	return false
}

// Helper function to check if a token is an SSL keyword
func (p *Parser) isSSLKeyword(tokenType token.TokenType) bool {
	for _, sslTokenType := range lexer.SSLKeywords {
		if tokenType == sslTokenType {
			return true
		}
	}
	return false
}

func (p *Parser) isValidWhenEvent(t token.TokenType) bool {
	for _, validEvent := range validWhenEvents {
		if t == validEvent {
			return true
		}
	}
	return false
}

func (p *Parser) parseStringOperation() ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseStringOperation Start. Current token: %v\n", p.curToken.Literal)
	}

	stringOp := &ast.StringOperation{
		Token:    p.curToken,
		Function: p.curToken.Literal,
	}

	// Check for the operation (contains, starts_with, tolower, match, etc.)
	p.nextToken() // Move to the operation token

	// Accept any token type for the operation
	stringOp.Operation = p.curToken.Literal
	if config.DebugMode {
		fmt.Printf("DEBUG: parseStringOperation - Operation: %s\n", stringOp.Operation)
	}

	// Parse arguments
	var args []ast.Expression
	for p.peekTokenIs(token.LBRACKET) || p.peekTokenIs(token.STRING) || p.peekTokenIs(token.IDENT) || p.peekTokenIs(token.DOLLAR) {
		p.nextToken()
		arg := p.parseExpression(LOWEST)
		if arg == nil {
			if config.DebugMode {
				fmt.Printf("DEBUG: parseStringOperation Error: Failed to parse argument\n")
			}
			return nil
		}
		args = append(args, arg)
	}

	stringOp.Arguments = args

	// Validate number of arguments
	expectedArgs := map[string]int{
		"tolower":     1,
		"toupper":     1,
		"match":       2,
		"contains":    2,
		"starts_with": 2,
	}

	if expected, ok := expectedArgs[stringOp.Operation]; ok {
		if len(args) != expected {
			if config.DebugMode {
				fmt.Printf("DEBUG: parseStringOperation Error: %s operation expects %d argument(s), got %d\n", stringOp.Operation, expected, len(args))
			}
			return nil
		}
	} else {
		if config.DebugMode {
			fmt.Printf("DEBUG: parseStringOperation Error: Unknown string operation %s\n", stringOp.Operation)
		}
		return nil
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseStringOperation End. Result: %s %s %v\n", stringOp.Function, stringOp.Operation, stringOp.Arguments)
	}

	return stringOp
}

func (p *Parser) parseStartsWith(left ast.Expression) ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseStartsWith Start. Left expression: %v\n", left)
	}

	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	if config.DebugMode {
		fmt.Printf("DEBUG: parseStartsWith End. Result: %v\n", expression)
	}

	return expression
}

func (p *Parser) parsePoolStatement() ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: parsePoolStatement Start\n")
	}

	poolStmt := &ast.CallExpression{
		Token:    p.curToken,
		Function: &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal},
	}

	if !p.expectPeek(token.IDENT) {
		p.errors = append(p.errors, "ERROR: parsePoolStatement Expected IDENT")
		return nil
	}

	argument := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	poolStmt.Arguments = append(poolStmt.Arguments, argument)

	if config.DebugMode {
		fmt.Printf("DEBUG: parsePoolStatement End\n")
	}
	return poolStmt
}
