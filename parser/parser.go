package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

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
	prevToken token.Token
	peekToken token.Token

	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn

	braceCount        int
	declaredVariables map[string]bool
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:                 l,
		errors:            []string{},
		declaredVariables: make(map[string]bool),
	}
	// read two tokens so curToken and peekToken are both set
	p.nextToken()
	p.nextToken()

	// Initialize prevToken to an "empty" token or a special "start of file" token
	p.prevToken = token.Token{Type: token.ILLEGAL, Literal: ""}

	// Check for lexer errors
	if lexerErrors := l.Errors(); len(lexerErrors) > 0 {
		p.errors = append(p.errors, lexerErrors...)
	}

	p.prefixParseFns = make(map[token.TokenType]prefixParseFn)
	p.registerPrefix(token.ASTERISK, p.parsePrefixExpression)
	p.registerPrefix(token.BANG, p.parsePrefixExpression)
	p.registerPrefix(token.DOLLAR, p.parseVariableOrArrayAccess)
	p.registerPrefix(token.FALSE, p.parseBoolean)
	p.registerPrefix(token.IDENT, p.parseIdentifier)
	p.registerPrefix(token.LBRACE, p.parseHashLiteral)
	p.registerPrefix(token.LBRACKET, p.parseArrayLiteral)
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression)
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
	p.registerPrefix(token.HTTP_COOKIE, p.parseHttpCommand)

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
	p.registerPrefix(token.SSL_CERT, p.parseSSLCommand)
	p.registerPrefix(token.SSL_VERIFY_RESULT, p.parseSSLCommand)
	p.registerPrefix(token.SSL_SESSIONID, p.parseSSLCommand)
	p.registerPrefix(token.SSL_RENEGOTIATE, p.parseSSLCommand)
	p.registerPrefix(token.SSL_SESSIONVALID, p.parseSSLCommand)
	p.registerPrefix(token.SSL_SESSIONUPDATES, p.parseSSLCommand)

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
	p.reportError("peekError: Expected next token to be %s, got %s instead. Line: %d", t, p.peekToken.Type, p.peekToken.Line)
}

func (p *Parser) nextToken() {
	p.prevToken = p.curToken
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()

	if p.curToken.Type == token.LBRACE {
		p.braceCount++
	} else if p.curToken.Type == token.RBRACE {
		p.braceCount--
	}
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
			fmt.Printf("   ERROR: Failed to parse statement at token: %+v\n", p.curToken)
		}

		p.nextToken()
	}

	// Check for lexer errors after parsing
	lexerErrors := p.l.Errors()
	if len(lexerErrors) > 0 {
		p.errors = append(p.errors, lexerErrors...)
	}

	// Handle any remaining open blocks at EOF
	if p.braceCount != 0 {
		p.reportError("Unbalanced braces: depth at end of parsing is %d", p.braceCount)
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
	case token.FOREACH:
		return p.parseForEachStatement()
	case token.RETURN:
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
	case token.ELSEIF:
		stmt = p.parseIfStatement()
	case token.LBRACE:
		stmt = p.parseBlockStatement()
	case token.SWITCH:
		stmt = p.parseSwitchStatement()
	default:
		stmt = p.parseExpressionStatement()
	}

	if stmt == nil {
		p.reportError("parseStatement - Unexpected token: %s", p.curToken.Literal)
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

	// Check if the next token is a semicolon or a closing brace - if so, it's a bare return statement
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

	// Parse the target (should be an identifier)
	if !p.curTokenIs(token.IDENT) && !p.curTokenIs(token.LBRACKET) {
		p.reportError("parseSetStatement: Expected an identifier or '[', got %s", p.curToken.Type)
		return nil
	}

	if p.curTokenIs(token.LBRACKET) {
		// This is likely a command or expression in brackets
		expr := p.parseExpression(LOWEST)
		if expr == nil {
			return nil // Error already added in parseExpression
		}
		stmt.Name = expr
	} else {
		// This is a simple identifier
		isValid, err := p.isValidIRuleIdentifier(p.curToken.Literal, "variable")
		if !isValid {
			p.reportError("parseSetStatement: Invalid identifier %s: %v", p.curToken.Literal, err)
			return nil
		}
		stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseSetStatement Statement Name: %v.\n", stmt.Name)
	}

	p.nextToken() // Move to the value

	// Parse the value
	if p.curTokenIs(token.LBRACKET) {
		stmt.Value = p.parseArrayLiteral()
	} else {
		stmt.Value = p.parseExpression(LOWEST)
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseSetStatement name type: %T\n", stmt.Name)
		fmt.Printf("DEBUG: parseSetStatement statement name: %v\n", stmt.Name)
		fmt.Printf("DEBUG: parseSetStatement value type: %T\n", stmt.Value)
		fmt.Printf("DEBUG: parseSetStatement value: %v\n", stmt.Value)
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

	var leftExp ast.Expression

	switch {
	case p.curTokenIs(token.STRING):
		stringLit := &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
		return p.parseStringLiteralContents(stringLit)
	case p.curTokenIs(token.IDENT) && p.curToken.Literal == "string":
		return p.parseStringOperation()
	case p.curTokenIs(token.CLASS):
		return p.parseClassCommand()
	case p.curTokenIs(token.PERCENT):
		p.nextToken() // consume %
		return &ast.StringLiteral{Token: p.curToken, Value: "%" + p.curToken.Literal}
	case p.curTokenIs(token.IDENT) && strings.HasPrefix(p.curToken.Literal, "HTTP::"):
		return p.parseHttpCommand()
	case p.curTokenIs(token.LBRACE):
		return p.parseListLiteral()
	case p.curTokenIs(token.IDENT):
		leftExp = p.parseIdentifier()
	default:
		prefix := p.prefixParseFns[p.curToken.Type]
		if prefix == nil {
			if p.curTokenIs(token.RBRACE) || p.curTokenIs(token.RBRACKET) || p.curTokenIs(token.EOF) {
				return nil
			}
			p.noPrefixParseFnError(p.curToken.Type)
			return nil
		}
		leftExp = prefix()
	}

	// Check if leftExp is an InvalidIdentifier
	if invalidIdent, ok := leftExp.(*ast.InvalidIdentifier); ok {
		p.reportError("parseExpression: Got *ast.InvalidIdentifier: %s", invalidIdent.Value)
		return leftExp
	}

	// Handle multi-word identifiers with dashes
	if p.curTokenIs(token.IDENT) {
		identifier := p.curToken.Literal
		for p.peekTokenIs(token.MINUS) || (p.peekTokenIs(token.IDENT) && isValidHeaderName(identifier+"-"+p.peekToken.Literal)) {
			p.nextToken() // consume the '-' or move to the next part
			if p.curTokenIs(token.MINUS) {
				p.nextToken() // move to the next part after '-'
			}
			identifier += "-" + p.curToken.Literal
		}
		if identifier != p.curToken.Literal {
			leftExp = &ast.Identifier{
				Token: token.Token{Type: token.IDENT, Literal: identifier},
				Value: identifier,
			}
		}
	}

	// special handling for string literals
	if stringLit, ok := leftExp.(*ast.StringLiteral); ok {
		leftExp = p.parseStringLiteralContents(stringLit)
		if leftExp == nil {
			p.reportError("parseExpression - Error occurred parsing string contents")
			return nil
		}
	}

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
			fmt.Printf("DEBUG: parseExpression Parsing infix expression, operator: %s\n", p.curToken.Literal)
		}
		leftExp = infix(leftExp)
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseExpression End, result type: %T, value: %v\n", leftExp, leftExp)
	}

	return leftExp
}

func (p *Parser) parseIdentifier() ast.Expression {
	value := p.curToken.Literal

	if config.DebugMode {
		fmt.Printf("DEBUG: parseIdentifier called with value: %s\n", value)
	}

	if strings.HasPrefix(value, "$") {
		// This is a variable
		return &ast.Identifier{Token: p.curToken, Value: value}
	}

	isValid, err := p.isValidIRuleIdentifier(value, "standalone")
	if config.DebugMode {
		fmt.Printf("DEBUG: parseIdentifier: isValid: %v, %v, identifier: %s\n", isValid, err, value)
	}

	if !isValid || err != nil {
		p.reportError("parseIdentifier: Invalid identifier: %s", value)
		return &ast.InvalidIdentifier{Token: p.curToken, Value: value}
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseIdentifier: %s is a valid identifier\n", value)
	}
	return &ast.Identifier{Token: p.curToken, Value: value}
}

func isValidHeaderName(s string) bool {
	if config.DebugMode {
		fmt.Printf("DEBUG: isValidHeaderName called with value: %s\n", s)
	}

	// Check against a list of common headers
	for _, header := range commonHeaders {
		if strings.EqualFold(s, header) {
			if config.DebugMode {
				fmt.Printf("DEBUG: isValidHeaderName: %s is a valid common header name\n", s)
			}
			return true
		}
	}

	// Check if it's a valid custom header (starts with X- or has a hyphen)
	if strings.HasPrefix(strings.ToLower(s), "x-") || strings.Contains(s, "-") {
		return true
	}

	if config.DebugMode {
		fmt.Printf("   ERROR: isValidHeaderName: %s is not a valid header name\n", s)
	}
	return false
}

func (p *Parser) parseNumberLiteral() ast.Expression {
	lit := &ast.NumberLiteral{Token: p.curToken}

	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		p.reportError("parseNumberLiteral: could not parse %q as integer", p.curToken.Literal)
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
	if config.DebugMode {
		fmt.Printf("DEBUG: parseGroupedExpression Start. Token: %v\n", p.curToken.Literal)
	}

	// Check if we're actually starting with a left parenthesis
	if !p.curTokenIs(token.LPAREN) {
		p.reportError("parseGroupedExpression: Expected '(', got %s", p.curToken.Literal)
		return nil
	}

	p.nextToken()
	exp := p.parseExpression(LOWEST)

	// Ensure we have a matching closing parenthesis
	if !p.expectPeek(token.RPAREN) {
		p.reportError("parseGroupedExpression: Expected ')' to match '(', got %s", p.curToken.Literal)
		return nil
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseGroupedExpression End. Expr: %v\n", exp)
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
		p.prevToken = p.curToken
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
	p.reportError("No prefix parse function for %s found", t)
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
		fmt.Printf("DEBUG: parseBlockStatement Entering block statement. Brace count: %d\n", p.braceCount)
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
			fmt.Printf("   ERROR: parseBlockStatement Failed to parse statement at token: %+v\n", p.curToken)
		}

		p.nextToken()
	}

	if p.curTokenIs(token.RBRACE) {
		p.braceCount--
	} else if p.curTokenIs(token.EOF) {
		p.braceCount--
	}

	if p.curTokenIs(token.EOF) && p.braceCount > 0 {
		p.reportError("parseBlockStatement: Unexpected EOF, expected '}'. Brace count: %d", p.braceCount)
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseBlockStatement End, statements: %d\n", len(block.Statements))
	}

	return block
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	fmt.Printf("DEBUG: parseIndexExpression - Start\n")

	exp := &ast.IndexExpression{Token: p.curToken, Left: left}

	if !p.expectPeek(token.LPAREN) {
		if config.DebugMode {
			fmt.Printf("   ERROR: parseIndexExpression - Expected LPAREN, got %s\n", p.curToken.Type)
		}
		return nil
	}

	p.nextToken() // move past '(' token
	exp.Index = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		if config.DebugMode {
			fmt.Printf("   ERROR: parseIndexExpression - Expected RPAREN, got %s\n", p.curToken.Type)
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
				fmt.Printf("   ERROR: parseHashLiteral - Expected STRING, got %s\n", p.curToken.Type)
			}
			return nil
		}
		key := &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}

		// Parse value
		if !p.expectPeek(token.STRING) {
			if config.DebugMode {
				fmt.Printf("   ERROR: parseHashLiteral - Expected STRING, got %s\n", p.curToken.Type)
			}
			return nil
		}
		value := &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}

		hash.Pairs[*key] = value

		if !p.peekTokenIs(token.RBRACE) && !p.expectPeek(token.COMMA) {
			if config.DebugMode {
				fmt.Printf("   ERROR: parseHashLiteral - Expected COMMA for Peek token, got %s\n", p.curToken.Type)
			}
			return nil
		}
	}

	if !p.expectPeek(token.RBRACE) {
		if config.DebugMode {
			fmt.Printf("   ERROR: parseHashLiteral - Expected RBRACE, got %s\n", p.curToken.Type)
		}
		return nil
	}

	return hash
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

	if left == nil {
		if config.DebugMode {
			fmt.Printf("DEBUG: parseInfixExpression - Left expression is nil\n")
		}
		return nil
	}

	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()

	expression.Right = p.parseExpression(precedence)

	if expression.Right == nil {
		if config.DebugMode {
			fmt.Printf("DEBUG: parseInfixExpression - Right expression is nil\n")
		}
		p.reportError("parseInfixExpression: Invalid right-hand side of infix expression")
		return nil
	}

	if !isValidOperatorForTypes(expression.Operator, expression.Left, expression.Right) {
		if config.DebugMode {
			fmt.Printf("   ERROR: parseInfixExpression: isValidOperatorForTypes FALSE for '%v'\n", expression)
		}
		p.reportError("parseInfixExpression: Invalid operator %s for types %T and %T", expression.Operator, expression.Left, expression.Right)
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseInfixExpression End - Operator: %s, Left: %T, Right: %T\n", expression.Operator, expression.Left, expression.Right)
	}

	return expression
}

func (p *Parser) parseSetExpression() ast.Expression {
	stmt := &ast.SetStatement{Token: p.curToken}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseSetExpression - Starting\n")
	}

	p.nextToken() // move past 'set'

	variableName := p.curToken.Literal
	p.declareVariable(variableName)
	stmt.Name = &ast.Identifier{Token: p.curToken, Value: variableName}

	if stmt.Name == nil {
		return nil
	}

	// Declare the variable if it's a simple identifier
	if ident, ok := stmt.Name.(*ast.Identifier); ok {
		variableName := ident.Value
		if strings.HasPrefix(variableName, "$") {
			p.declareVariable(variableName)
			if config.DebugMode {
				fmt.Printf("DEBUG: parseSetExpression - Declared variable: %s\n", variableName)
			}
		}
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

	p.nextToken() // Move past the opening bracket [

	// Handle nested array or expression
	for !p.curTokenIs(token.RBRACKET) && !p.curTokenIs(token.EOF) {
		var expr ast.Expression

		if p.curTokenIs(token.IDENT) && p.curToken.Literal == "string" {
			expr = p.parseStringOperation()
		} else if p.curTokenIs(token.LBRACKET) {
			// Handle nested command
			nestedExpr := p.parseArrayLiteral()
			if nestedExpr == nil {
				return nil
			}
			expr = nestedExpr
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
				fmt.Printf("   ERROR: parseArrayLiteral - Failed to parse element %T\n", expr)
			}
			return nil
		}

		// Handle TCL-style command arguments
		for p.peekTokenIs(token.MINUS) {
			p.nextToken() // consume the '-'
			p.nextToken() // move to the argument
			arg := p.parseExpression(LOWEST)
			if arg != nil {
				array.Elements = append(array.Elements, arg)
			}
		}

		// Break if we've reached the end of the array
		if p.peekTokenIs(token.RBRACKET) {
			break
		}

		// Move to next token if it's not a '-' and not the closing bracket
		if !p.peekTokenIs(token.MINUS) && !p.peekTokenIs(token.RBRACKET) {
			p.nextToken()
		}
	}

	if !p.expectPeek(token.RBRACKET) {
		p.reportError("parseArrayLiteral - Expected closing bracket, got %s", p.curToken.Literal)
		if config.DebugMode {
			fmt.Printf("   ERROR: parseArrayLiteral Error - Expected closing bracket, got %s\n", p.curToken.Literal)
		}
		return nil
	}

	if config.DebugMode {
		for i, elem := range array.Elements {
			fmt.Printf("DEBUG: parseArrayLiteral - Element %d: %T\n", i, elem)
		}
		fmt.Printf("DEBUG: parseArrayLiteral End. Array elements: %d\n", len(array.Elements))
	}
	return array
}

func (p *Parser) parseSSLCommand() ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseSSLCommand Start. Current token: %s\n", p.curToken.Literal)
	}
	command := &ast.SSLExpression{Token: p.curToken}
	var commandParts []string

	for !p.peekTokenIs(token.RBRACKET) && !p.peekTokenIs(token.EOF) {
		if config.DebugMode {
			fmt.Printf("DEBUG: parseSSLCommand loop. Current token: %s\n", p.curToken.Literal)
		}
		commandParts = append(commandParts, p.curToken.Literal)
		p.nextToken()
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
		p.reportError("parseVariableOrArrayAccess: Expected identifier after $, got %s instead", p.curToken.Type)
		return nil
	}

	varExp := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal, IsVariable: true}

	if p.peekTokenIs(token.LPAREN) {
		p.nextToken() // consume '('
		p.nextToken() // move to index
		index := p.parseExpression(LOWEST)
		if !p.expectPeek(token.RPAREN) {
			if config.DebugMode {
				fmt.Printf("   ERROR: parseVariableOrArrayAccess:  Expected RPAREN, got %s\n", p.curToken.Type)
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
			fmt.Printf("   ERROR: parseWhenNode - Expected HTTP_REQUEST or LB_SELECTED, got %s\n", p.curToken.Type)
		}
		return nil
	}
	when.Event = p.curToken.Literal

	if !p.expectPeek(token.LBRACE) {
		if config.DebugMode {
			fmt.Printf("   ERROR: parseWhenNode - Expected LBRACE, got %s\n", p.curToken.Type)
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
		fmt.Printf("DEBUG: parseHttpCommand Start - Current Token: %s\n", p.curToken.Literal)
	}

	expr := &ast.HttpExpression{Token: p.curToken}
	fullCommand := p.curToken.Literal

	// Check if the command is a valid HTTP keyword
	if _, isValidHttpCommand := lexer.HttpKeywords[fullCommand]; isValidHttpCommand {
		expr.Command = &ast.Identifier{Token: p.curToken, Value: fullCommand}
	} else {
		p.reportError("parseHttpCommand - Invalid HTTP command: %s", fullCommand)
		if config.DebugMode {
			fmt.Printf("   ERROR: parseHttpCommand - Invalid HTTP command detected: %s\n", fullCommand)
		}
		return nil // Return nil for invalid commands
	}

	switch {
	case lexer.HttpKeywords[fullCommand] != token.ILLEGAL:
		expr.Command = &ast.Identifier{Token: p.curToken, Value: fullCommand}
	case fullCommand == "HTTP::header":
		expr.Command = &ast.Identifier{Token: p.curToken, Value: "HTTP::header"}
		if p.peekTokenIs(token.IDENT) && p.peekToken.Literal == "names" {
			p.nextToken()
			expr.Argument = &ast.Identifier{Token: p.curToken, Value: "names"}
		} else if p.peekTokenIs(token.STRING) || p.peekTokenIs(token.IDENT) {
			p.nextToken()
			expr.Argument = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
		}
	default:
		p.reportError("parseHttpCommand: Invalid HTTP command or header: %s", fullCommand)
		if config.DebugMode {
			fmt.Printf("   ERROR: parseHttpCommand - Invalid HTTP command or header detected: %s\n", fullCommand)
		}
		return nil
	}

	// Check for additional arguments
	for p.peekTokenIs(token.STRING) {
		p.nextToken()
		if expr.Argument == nil {
			expr.Argument = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
		} else {
			// If there's already an argument, create a list of arguments
			if argList, ok := expr.Argument.(*ast.ArrayLiteral); ok {
				argList.Elements = append(argList.Elements, &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal})
			} else {
				expr.Argument = &ast.ArrayLiteral{
					Token: p.curToken,
					Elements: []ast.Expression{
						expr.Argument,
						&ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal},
					},
				}
			}
		}
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseHttpCommand End - Command: %s, Argument: %v\n", expr.Command.Value, expr.Argument)
	}
	return expr
}

func (p *Parser) parseIfStatement() *ast.IfStatement {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseIfStatement Start - curToken: %s\n", p.curToken.Literal)
	}
	stmt := &ast.IfStatement{Token: p.curToken}

	// Expect '{'
	if !p.expectPeek(token.LBRACE) {
		p.reportError("parseIfStatement: Expected {, got %s", p.curToken.Literal)
		return nil
	}

	p.nextToken() // consume '{'

	// Parse the condition
	stmt.Condition = p.parseExpression(LOWEST)

	// Expect '}'
	if !p.expectPeek(token.RBRACE) {
		p.reportError("parseIfStatement: Expected }, got %s", p.curToken.Literal)
		return nil
	}

	// Expect '{' for consequence block
	if !p.expectPeek(token.LBRACE) {
		p.reportError("parseIfStatement: Consequence Expected '{', got %s", p.curToken.Literal)
		return nil
	}
	stmt.Consequence = p.parseBlockStatement()

	// Parse else-if and else clauses
	currentStmt := stmt
	for p.peekTokenIs(token.ELSEIF) || p.peekTokenIs(token.ELSE) {
		p.nextToken() // consume 'elseif' or 'else'

		if p.curTokenIs(token.ELSEIF) {
			elseIfStmt := &ast.IfStatement{Token: p.curToken}

			// Expect '{'
			if !p.expectPeek(token.LBRACE) {
				p.reportError("parseIfStatement: ELSEIF Expected {, got %s", p.curToken.Literal)
				return nil
			}

			p.nextToken() // consume '{'

			// Parse the else-if condition
			elseIfStmt.Condition = p.parseExpression(LOWEST)

			// Expect '}'
			if !p.expectPeek(token.RBRACE) {
				p.reportError("parseIfStatement: ELSEIF Expected }, got %s", p.curToken.Literal)
				return nil
			}

			// Expect '{' for else-if consequence block
			if !p.expectPeek(token.LBRACE) {
				p.reportError("parseIfStatement: ELSEIF Consequence Expected {, got %s", p.curToken.Literal)
				return nil
			}
			elseIfStmt.Consequence = p.parseBlockStatement()

			// Add the else-if statement as an alternative to the current statement
			currentStmt.Alternative = &ast.BlockStatement{
				Statements: []ast.Statement{elseIfStmt},
			}
			currentStmt = elseIfStmt
		} else if p.curTokenIs(token.ELSE) {
			// Parse the final else clause
			if !p.expectPeek(token.LBRACE) {
				p.reportError("parseIfStatement: ELSE Expected {, got %s", p.curToken.Literal)
				return nil
			}
			currentStmt.Alternative = p.parseBlockStatement()
			break // Exit the loop after parsing the final else
		}
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseIfStatement End - Condition: %T\n", stmt.Condition)
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
		p.nextToken() // Advance to the event token
	} else {
		p.reportError("parseWhenExpression - Expected HTTP_REQUEST or LB_SELECTED")
		return nil
	}

	expr.Event = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.LBRACE) {
		p.reportError("parseWhenExpression - expected LBRACE")
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

	// Handle options like -glob
	for p.curTokenIs(token.MINUS) {
		option := p.curToken.Literal
		p.nextToken() // move past the option
		if p.curTokenIs(token.IDENT) {
			option += " " + p.curToken.Literal
			switchStmt.Options = append(switchStmt.Options, option)
			p.nextToken() // move past the option value
		}
	}

	// Handle the -- separator if present
	if p.curTokenIs(token.MINUS) && p.peekTokenIs(token.MINUS) {
		p.nextToken() // move past first -
		p.nextToken() // move past second -
	}

	// Parse the switch value (which might be a string operation)
	switchStmt.Value = p.parseExpression(LOWEST)

	if !p.expectPeek(token.LBRACE) {
		p.reportError("parseSwitchStatement expected LBRACE")
		return nil
	}

	switchStmt.Cases = []*ast.CaseStatement{}

	p.nextToken() // Move past the opening brace

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		if config.DebugMode {
			fmt.Printf("DEBUG: parseSwitchStatement Switch loop - Current token: %s, Literal: %s\n", p.curToken.Type, p.curToken.Literal)
		}

		if p.curTokenIs(token.DEFAULT) {
			switchStmt.Default = p.parseDefaultCase()
		} else if p.curTokenIs(token.STRING) {
			caseStmt := p.parseCaseStatement()
			if caseStmt != nil {
				switchStmt.Cases = append(switchStmt.Cases, caseStmt)
				if config.DebugMode {
					fmt.Printf("DEBUG: parseSwitchStatement Added case, total cases: %d\n", len(switchStmt.Cases))
				}
			}
		} else {
			p.reportError(fmt.Sprintf("parseSwitchStatement: Invalid case statement starting with token: %s", p.curToken.Literal))
			return nil // Error occurred in parsing case statement
		}

		// Ensure we're moving forward after each case
		if p.peekTokenIs(token.RBRACE) {
			p.nextToken()
			break
		}
	}

	if !p.curTokenIs(token.RBRACE) {
		if config.DebugMode {
			fmt.Printf("   ERROR: parseSwitchStatement expected RBRACE. Got=%s\n", p.curToken.Literal)
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
		p.reportError("parseCaseStatement expected LBRACE")
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
		p.reportError("parseDefaultCase: Expected LBRACE")
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
		p.reportError("parseIpExpression: Unexpected IP token: %s", p.curToken.Literal)
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
		fmt.Printf("DEBUG: parseStringOperation Start\n")
	}
	stringOp := &ast.StringOperation{Token: p.curToken}

	p.nextToken() // Move past 'string'
	stringOp.Operation = p.curToken.Literal
	if config.DebugMode {
		fmt.Printf("DEBUG: parseStringOperation Operation: %v\n", stringOp.Operation)
	}

	var args []ast.Expression
	for p.peekToken.Type != token.RBRACKET && p.peekToken.Type != token.EOF {
		p.nextToken()
		if p.curTokenIs(token.MINUS) && p.peekTokenIs(token.IDENT) {
			args = append(args, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal + p.peekToken.Literal})
			p.nextToken() // Skip the identifier after '-'
		} else if p.curTokenIs(token.LBRACE) {
			mapArg := p.parseMapArgument()
			if mapArg != nil {
				args = append(args, mapArg)
			}
		} else {
			arg := p.parseExpression(LOWEST)
			if arg != nil {
				args = append(args, arg)
			}
		}
	}

	stringOp.Arguments = args
	if config.DebugMode {
		fmt.Printf("DEBUG: parseStringOperation Arguments: %v\n", stringOp.Arguments)
		fmt.Printf("DEBUG: parseStringOperation End\n")
	}
	return stringOp
}

func (p *Parser) parseMapArgument() ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseMapArgument Start\n")
	}
	mapArg := &ast.MapLiteral{Token: p.curToken}
	mapArg.Pairs = make(map[ast.Expression]ast.Expression)

	for !p.peekTokenIs(token.RBRACE) {
		p.nextToken() // Move to the key
		key := p.parseExpression(LOWEST)

		if !p.expectPeek(token.STRING) {
			if config.DebugMode {
				fmt.Printf("   ERROR: parseMapArgument expected STRING, got %v\n", p.curToken.Literal)
			}
			return nil
		}

		value := p.parseExpression(LOWEST)
		mapArg.Pairs[key] = value

		if !p.peekTokenIs(token.RBRACE) && !p.expectPeek(token.COMMA) {
			if config.DebugMode {
				fmt.Printf("   ERROR: parseMapArgument expected RBACE OR COMMA, got %v\n", p.curToken.Literal)
			}
			return nil
		}
	}

	if !p.expectPeek(token.RBRACE) {
		return nil
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseMapArgument End\n")
	}

	return mapArg
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
		p.reportError("parsePoolStatement: Expected IDENT")
		return nil
	}

	argument := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	poolStmt.Arguments = append(poolStmt.Arguments, argument)

	if config.DebugMode {
		fmt.Printf("DEBUG: parsePoolStatement End\n")
	}
	return poolStmt
}

func (p *Parser) parseClassCommand() ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseClassCommand Start - curToken: %v, peekToken: %v\n", p.curToken.Literal, p.peekToken.Literal)
	}

	expression := &ast.ClassCommand{
		Token: p.curToken, // This should be the 'class' token
	}

	// Advance to the subcommand
	if !p.expectPeek(token.MATCH) {
		if config.DebugMode {
			fmt.Printf("   ERROR: parseClassCommand Expected 'match', got %v\n", p.curToken.Literal)
		}
		return nil
	}
	expression.Subcommand = p.curToken.Literal

	if config.DebugMode {
		fmt.Printf("DEBUG: parseClassCommand Subcommand: %v\n", expression.Subcommand)
	}

	return p.parseClassMatchOrSearch(expression)
}

func (p *Parser) parseClassMatchOrSearch(cmd *ast.ClassCommand) ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseClassMatchOrSearch Start - curToken: %v (type: %v), peekToken: %v (type: %v)\n",
			p.curToken.Literal, p.curToken.Type, p.peekToken.Literal, p.peekToken.Type)
	}

	// Parse item (should be a variable)
	p.nextToken()
	item := p.parseExpression(LOWEST)
	cmd.Arguments = append(cmd.Arguments, item)

	// Parse operator
	p.nextToken()
	operator := p.parseExpression(LOWEST)
	cmd.Arguments = append(cmd.Arguments, operator)

	// Parse class name
	p.nextToken()
	className := p.parseExpression(LOWEST)
	cmd.Arguments = append(cmd.Arguments, className)

	if config.DebugMode {
		fmt.Printf("DEBUG: parseClassMatchOrSearch End - Arguments: %v\n", cmd.Arguments)
	}

	return cmd
}

func (p *Parser) parseStringLiteralContents(s *ast.StringLiteral) ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseStringLiteralContents Start - Value: %s\n", s.Value)
	}

	parts := []ast.Expression{}
	currentPart := ""
	inCommand := false
	value := s.Value
	startPosition := 0

	for len(value) > 0 {
		if strings.HasPrefix(value, "[") && !inCommand {
			if currentPart != "" {
				parts = append(parts, &ast.StringLiteral{Token: s.Token, Value: currentPart})
				currentPart = ""
			}
			inCommand = true
			commandStart := 1
			end := strings.Index(value[1:], "]")
			if end != -1 {
				end++ // Adjust for the starting '['
				command := value[commandStart:end]
				parts = append(parts, &ast.HttpExpression{Token: s.Token, Command: &ast.Identifier{Token: s.Token, Value: command}})
				value = value[end+1:]
			} else {
				// Unclosed command, treat as literal
				currentPart += "["
				value = value[1:]
			}
			inCommand = false
		} else if strings.HasPrefix(value, "${") && !inCommand {
			if currentPart != "" {
				parts = append(parts, &ast.StringLiteral{Token: s.Token, Value: currentPart})
				currentPart = ""
			}
			end := strings.Index(value, "}")
			if end != -1 {
				varName := value[2:end]
				parts = append(parts, &ast.Identifier{Token: token.Token{Type: token.IDENT, Literal: varName}, Value: varName})
				value = value[end+1:]
			} else {
				// Unclosed variable, treat as literal
				currentPart += "${"
				value = value[2:]
			}
		} else if strings.HasPrefix(value, "$") && !inCommand {
			if currentPart != "" {
				parts = append(parts, &ast.StringLiteral{Token: s.Token, Value: currentPart})
				currentPart = ""
			}
			identStart := 1
			end := strings.IndexFunc(value[1:], func(r rune) bool {
				return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_'
			})
			if end == -1 {
				end = len(value) - 1
			} else {
				end++ // Adjust for the starting '$'
			}
			identifier := value[identStart:end]
			parts = append(parts, &ast.Identifier{Token: s.Token, Value: identifier})
			value = value[end:]
		} else {
			if inCommand {
				currentPart += string(value[0])
				value = value[1:]
			} else {
				// Handle regular text
				end := strings.IndexAny(value, "[${$")
				if end == -1 {
					currentPart += value
					break
				}
				currentPart += value[:end]
				value = value[end:]
			}

			// Break condition to prevent infinite loop
			if startPosition == len(value) {
				if config.DebugMode {
					fmt.Printf("DEBUG: parseStringLiteralContents - Breaking loop due to no progress\n")
				}
				break
			}
		}
		startPosition = len(value)
	}

	if currentPart != "" {
		parts = append(parts, &ast.StringLiteral{Token: s.Token, Value: currentPart})
	}

	if len(parts) == 0 {
		return s
	}

	if len(parts) == 1 {
		return parts[0]
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseStringLiteralContents End - Parts: %d\n", len(parts))
	}

	return &ast.InterpolatedString{Token: s.Token, Parts: parts}
}

func (p *Parser) parseForEachStatement() ast.Statement {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseForEachStatement Start\n")
	}
	stmt := &ast.ForEachStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		if config.DebugMode {
			fmt.Printf("   ERROR: parseForEachStatement Expected IDENT, got: %v\n", p.curToken.Literal)
		}
		return nil
	}

	stmt.Variable = p.curToken.Literal
	if config.DebugMode {
		fmt.Printf("DEBUG: parseForEachStatement Variable: %v\n", stmt.Variable)
	}
	p.declareVariable(stmt.Variable)

	p.nextToken() // Move to the list expression

	// Parse the list expression
	if p.curTokenIs(token.LBRACE) {
		listExpr := p.parseListLiteral()
		if listLiteral, ok := listExpr.(*ast.ListLiteral); ok {
			for _, elem := range listLiteral.Elements {
				if ident, ok := elem.(*ast.Identifier); ok {
					if isValid, _ := p.isValidIRuleIdentifier(ident.Value, "header"); !isValid {
						p.reportError("parseForEachStatement: Invalid header name in foreach loop: %s", ident.Value)
					}
				}
			}
		}
		stmt.List = listExpr
	} else {
		stmt.List = p.parseExpression(LOWEST)
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseForEachStatement List: %+v\n", stmt.List)
	}

	if !p.expectPeek(token.LBRACE) {
		if config.DebugMode {
			fmt.Printf("   ERROR: parseForEachStatement Expected LBRACE, got: %v\n", p.curToken.Literal)
		}
		return nil
	}

	stmt.Body = p.parseBlockStatement()
	if config.DebugMode {
		fmt.Printf("DEBUG: parseForEachStatement Body: %+v\n", stmt.Body)
		fmt.Printf("DEBUG: parseForEachStatement End, Final Statement: %+v\n", stmt)
	}

	return stmt
}

func (p *Parser) parseListLiteral() ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseListLiteral Start\n")
	}
	list := &ast.ListLiteral{Token: p.curToken}
	list.Elements = []ast.Expression{}

	p.nextToken() // Move past '{'

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		elem := p.parseExpression(LOWEST)
		if elem != nil {
			list.Elements = append(list.Elements, elem)
		}

		if p.peekTokenIs(token.RBRACE) {
			break
		}

		p.nextToken()
	}

	if !p.expectPeek(token.RBRACE) {
		if config.DebugMode {
			fmt.Printf("   ERROR: parseListLiteral Expected RBRACE but got %s\n", p.curToken.Type)
		}
		return nil
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseListLiteral End\n")
	}
	return list
}

func (p *Parser) isValidIRuleIdentifier(value string, identifierContext string) (bool, error) {
	if config.DebugMode {
		fmt.Printf("DEBUG: isValidIRuleIdentifier - Start. Value=%v, Context=%v\n", value, identifierContext)
	}

	// Check for reserved keywords
	if reservedKeywords[strings.ToLower(value)] {
		if identifierContext == "variable" {
			return false, fmt.Errorf("ERROR: isValidIRuleIdentifier - '%s' is a reserved keyword and should not be used as a variable name", value)
		}
		if config.DebugMode {
			fmt.Printf("DEBUG: isValidIRuleIdentifier - Using reserved keyword '%s' in context '%s'\n", value, identifierContext)
		}
		return true, nil
	}

	// Check if it's a variable (starts with $)
	if strings.HasPrefix(value, "$") {
		if config.DebugMode {
			fmt.Printf("DEBUG: isValidIRuleIdentifier - %s is a variable\n", value)
		}
		return true, nil
	}

	// Check if it's a common iRule identifier or command
	if isCommonIRuleIdentifier(value) {
		if config.DebugMode {
			fmt.Printf("DEBUG: isValidIRuleIdentifier - %s is a common iRule identifier or command\n", value)
		}
		return true, nil
	}

	// Check context-specific validations
	switch identifierContext {
	case "variable":
		// Stricter check for variable names
		if regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(value) {
			if config.DebugMode {
				fmt.Printf("DEBUG: isValidIRuleIdentifier - %s is a valid variable identifier\n", value)
			}
			return true, nil
		}
		return false, fmt.Errorf("invalid variable identifier: %s", value)

	case "standalone":
		// Allow single-letter identifiers and check against common headers (case-insensitive)
		if len(value) == 1 && regexp.MustCompile(`^[a-zA-Z]$`).MatchString(value) {
			if config.DebugMode {
				fmt.Printf("DEBUG: isValidIRuleIdentifier - %s is a valid single-letter identifier\n", value)
			}
			return true, nil
		}
		// Check against common headers (case-insensitive)
		for _, header := range commonHeaders {
			if strings.EqualFold(value, header) {
				if config.DebugMode {
					fmt.Printf("DEBUG: isValidIRuleIdentifier - %s is a valid common header\n", value)
				}
				return true, nil
			}
		}

		// Check for command patterns (e.g., HTTP::*)
		if strings.Contains(value, "::") {
			parts := strings.Split(value, "::")
			if len(parts) == 2 {
				validPrefixes := []string{"HTTP", "TCP", "SSL", "LB"}
				for _, prefix := range validPrefixes {
					if strings.EqualFold(parts[0], prefix) {
						if config.DebugMode {
							fmt.Printf("DEBUG: isValidIRuleIdentifier - %s is a valid command pattern\n", value)
						}
						return true, nil
					}
				}
			}
		}

	case "header":
		if isValidHeaderName(value) {
			if config.DebugMode {
				fmt.Printf("DEBUG: isValidIRuleIdentifier - %s is a valid HTTP header name\n", value)
			}
			return true, nil
		}
		return false, fmt.Errorf("invalid HTTP header name: %s", value)
	}

	// Check if it's a valid command or keyword
	if _, ok := lexer.HttpKeywords[value]; ok {
		if config.DebugMode {
			fmt.Printf("DEBUG: isValidIRuleIdentifier - %s is a valid HTTP keyword\n", value)
		}
		return true, nil
	}
	if _, ok := lexer.LbKeywords[value]; ok {
		if config.DebugMode {
			fmt.Printf("DEBUG: isValidIRuleIdentifier - %s is a valid LB keyword\n", value)
		}
		return true, nil
	}
	if _, ok := lexer.SSLKeywords[value]; ok {
		if config.DebugMode {
			fmt.Printf("DEBUG: isValidIRuleIdentifier - %s is a valid SSL keyword\n", value)
		}
		return true, nil
	}

	// Check if it's a valid custom identifier (declared variable or function)
	if p.isValidCustomIdentifier(value) {
		return true, nil
	}

	// Check if it's a valid logging facility
	if isValidLoggingFacility(value) {
		if config.DebugMode {
			fmt.Printf("DEBUG: isValidIRuleIdentifier - %s is a valid logging facility\n", value)
		}
		return true, nil
	}

	return false, fmt.Errorf("ERROR: isValidIRuleIdentifier - invalid identifier: %s", value)
}

func isValidOperatorForTypes(operator string, left, right ast.Expression) bool {
	if left == nil || right == nil {
		return true // Allow partial expressions during parsing
	}

	switch operator {
	case "contains", "starts_with", "ends_with", "equals":
		// These operators are valid for strings, HTTP expressions, array literals, IP address literals, and identifiers
		return (isStringType(left) || isHttpExpression(left) || isArrayLiteral(left) || isIpAddressLiteral(left) || isIdentifier(left)) &&
			(isStringType(right) || isHttpExpression(right) || isArrayLiteral(right) || isIpAddressLiteral(right) || isIdentifier(right))
	case "eq", "==", "!=":
		// Equality operators are valid for most types
		return true
	case "<", ">", "<=", ">=":
		// Comparison operators are valid for numbers and strings
		return (isNumberType(left) && isNumberType(right)) || (isStringType(left) && isStringType(right)) ||
			(isIdentifier(left) && isStringType(right)) || (isStringType(left) && isIdentifier(right))
	case "+", "-", "*", "/":
		// Arithmetic operators are valid for numbers, infix expressions, array literals, and identifiers
		return (isNumberType(left) || isInfixExpression(left) || isArrayLiteral(left) || isIdentifier(left)) &&
			(isNumberType(right) || isInfixExpression(right) || isIdentifier(right))
	case "&&", "||":
		// Logical operators are valid for boolean expressions, HTTP expressions, and identifiers
		return isBooleanType(left) || isHttpExpression(left) || isInfixExpression(left) || isIdentifier(left) ||
			isBooleanType(right) || isHttpExpression(right) || isInfixExpression(right) || isIdentifier(right)
	default:
		return true // Allow unknown operators to be handled elsewhere
	}
}

func isIpAddressLiteral(expr ast.Expression) bool {
	_, ok := expr.(*ast.IpAddressLiteral)
	return ok
}

func isIdentifier(expr ast.Expression) bool {
	_, ok := expr.(*ast.Identifier)
	return ok
}

func isInfixExpression(expr ast.Expression) bool {
	_, ok := expr.(*ast.InfixExpression)
	return ok
}

func isHttpExpression(expr ast.Expression) bool {
	_, ok := expr.(*ast.HttpExpression)
	return ok
}

func isArrayLiteral(expr ast.Expression) bool {
	_, ok := expr.(*ast.ArrayLiteral)
	return ok
}

func isStringType(expr ast.Expression) bool {
	switch e := expr.(type) {
	case *ast.StringLiteral:
		return true
	case *ast.Identifier:
		return e.IsVariable // Assume variables can be strings
	case *ast.HttpExpression, *ast.LoadBalancerExpression, *ast.SSLExpression:
		// Assuming these expressions return string values
		return true
	default:
		return false
	}
}

func isNumberType(expr ast.Expression) bool {
	switch expr.(type) {
	case *ast.NumberLiteral:
		return true
	case *ast.Identifier:
		return true
	default:
		return false
	}
}

func isBooleanType(expr ast.Expression) bool {
	switch expr.(type) {
	case *ast.Boolean:
		return true
	case *ast.InfixExpression:
		return true
	case *ast.Identifier:
		return true
	default:
		return false
	}
}

func isCommonIRuleIdentifier(s string) bool {
	for _, identifier := range commonIdentifiers {
		if strings.EqualFold(s, identifier) {
			return true
		}
	}
	return false
}

func isValidLoggingFacility(s string) bool {
	validFacilities := []string{"local0.", "local1.", "local2.", "local3.", "local4.", "local5.", "local6.", "local7."}
	for _, facility := range validFacilities {
		if s == facility {
			return true
		}
	}
	return false
}

func (p *Parser) declareVariable(name string) {
	p.declaredVariables[name] = true
}

func (p *Parser) isValidCustomIdentifier(s string) bool {
	// Check if it's a declared variable
	if p.declaredVariables[s] {
		return true
	}

	// Check if it's a valid function name (assuming functions are declared with "proc")
	if strings.HasPrefix(s, "proc ") {
		return true
	}

	return false
}

func (p *Parser) reportError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	formattedMsg := "   " + msg
	p.errors = append(p.errors, formattedMsg)
}
