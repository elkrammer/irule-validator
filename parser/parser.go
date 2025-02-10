package parser

import (
	"fmt"
	"regexp"
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
	LOGICAL     // && or ||
	EQUALS      // ==
	LESSGREATER // > or <
	SUM         // +
	PRODUCT     // *
	PREFIX      // -X or !X
	CALL        // myFunction(X)
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

	braceCount          int
	declaredVariables   map[string]bool
	symbolTable         *SymbolTable
	currentLine         int
	lastKnownLine       int
	isParsingClassMatch bool
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:                 l,
		errors:            []string{},
		declaredVariables: make(map[string]bool),
		symbolTable:       NewSymbolTable(),
		currentLine:       1,
		lastKnownLine:     1,
	}

	// read two tokens so curToken and peekToken are both set
	p.nextToken()
	p.nextToken()

	// initialize prevToken to an "empty" token or a special "start of file" token
	p.prevToken = token.Token{Type: token.ILLEGAL, Literal: "", Line: p.l.CurrentLine()}

	// check for lexer errors
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
	p.registerPrefix(token.SLASH, p.parseSlashExpression)
	p.registerPrefix(token.REGEX, p.parseRegexLiteral)

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

	// IP Commands
	p.registerPrefix(token.IP_CLIENT_ADDR, p.parseIpExpression)
	p.registerPrefix(token.IP_SERVER_ADDR, p.parseIpExpression)
	p.registerPrefix(token.IP_REMOTE_ADDR, p.parseIpExpression)
	p.registerPrefix(token.IP_ADDRESS, p.parseIpAddressLiteral)

	p.registerPrefix(token.SWITCH, p.parseSwitchExpression)
	p.registerPrefix(token.DEFAULT, p.parseDefaultExpression)

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
	p.reportError("peekError: Expected next token to be %s, got %s instead", t, p.peekToken.Type)
}

func (p *Parser) nextToken() {
	p.prevToken = p.curToken
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
	p.currentLine = p.curToken.Line

	if p.peekToken.Line > 0 {
		p.lastKnownLine = p.peekToken.Line
	}

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
		errorMsg := fmt.Sprintf("Unbalanced braces: depth at end of parsing is %d", p.braceCount)
		p.reportError(errorMsg)
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
	case token.LTM:
		stmt = p.parseLtmRule()
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

	p.nextToken() // move past 'set'

	// parse the target (should be an identifier)
	if !p.curTokenIs(token.IDENT) && !p.curTokenIs(token.LBRACKET) {
		p.reportError("parseSetStatement: Expected an identifier or '[', got %s", p.curToken.Type)
		return nil
	}

	var variableName string

	if p.curTokenIs(token.LBRACKET) {
		// this is likely a command or expression in brackets
		expr := p.parseExpression(LOWEST)
		if expr == nil {
			return nil
		}
		stmt.Name = expr

		// try to extract variable name if it's an identifier
		if ident, ok := expr.(*ast.Identifier); ok {
			variableName = ident.Value
		}
	} else {
		// this is a simple identifier
		isValid, err := p.isValidIRuleIdentifier(p.curToken.Literal, "variable")
		if !isValid {
			p.reportError("parseSetStatement: Invalid identifier %s: %v", p.curToken.Literal, err)
			return nil
		}
		variableName = p.curToken.Literal
		stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	}

	// add the variable to the declared variables map
	if variableName != "" {
		p.declaredVariables[variableName] = true
		if config.DebugMode {
			fmt.Printf("DEBUG: parseSetStatement Added variable %s to declared variables\n", variableName)
		}
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseSetStatement Statement Name: %v.\n", stmt.Name)
	}

	p.nextToken() // move to the value

	// parse the value
	if p.curTokenIs(token.LBRACKET) {
		if p.peekTokenIs(token.IDENT) && p.peekToken.Literal == "class" {
			stmt.Value = p.parseClassCommand()
		} else {
			stmt.Value = p.parseArrayLiteral()
		}
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
		fmt.Printf("DEBUG: parseExpressionStatement Start, current token: %s, Line: %d\n", p.curToken.Type, p.currentLine)
	}
	stmt := &ast.ExpressionStatement{Token: p.curToken}

	if p.curTokenIs(token.IDENT) {
		switch p.curToken.Literal {
		case "pool":
			stmt.Expression = p.parsePoolStatement()
		case "node":
			stmt.Expression = p.parseNodeStatement()
		default:
			stmt.Expression = p.parseExpression(LOWEST)
		}
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

	// check for matches_regex as the current token
	if p.curTokenIs(token.IDENT) && p.curToken.Literal == "matches_regex" {
		if config.DebugMode {
			fmt.Printf("DEBUG: parseExpression encountered matches_regex as current token\n")
		}
		return p.parseMatchesRegexExpression(nil)
	}

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
				Token: token.Token{Type: token.IDENT, Literal: identifier, Line: p.l.CurrentLine()},
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
		// this is a variable
		return &ast.Identifier{Token: p.curToken, Value: value}
	}

	context := "standalone"
	if p.isParsingClassMatch {
		context = "class_match"
	}

	isValid, err := p.isValidIRuleIdentifier(value, context)
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
	token := p.curToken
	value := token.Literal[1 : len(token.Literal)-1] // remove quotes

	if strings.Contains(value, "\\") || strings.Contains(value, "${") {
		return p.parseInterpolatedString(token, value)
	}

	return &ast.StringLiteral{Token: token, Value: value}
}

func (p *Parser) parseInterpolatedString(token token.Token, value string) ast.Expression {
	parts := []ast.Expression{}
	currentPart := ""

	for i := 0; i < len(value); i++ {
		if value[i] == '\\' && i+1 < len(value) {
			currentPart += string(value[i : i+2])
			i++
		} else if value[i] == '$' && i+1 < len(value) && value[i+1] == '{' {
			if currentPart != "" {
				parts = append(parts, &ast.StringLiteral{Token: token, Value: currentPart})
				currentPart = ""
			}
			end := strings.Index(value[i:], "}")
			if end == -1 {
				p.reportError("parseInterpolatedString: Unterminated interpolation in string")
				return nil
			}
			expr := p.parseExpression(LOWEST)
			parts = append(parts, expr)
			i += end
		} else {
			currentPart += string(value[i])
		}
	}

	if currentPart != "" {
		parts = append(parts, &ast.StringLiteral{Token: token, Value: currentPart})
	}

	return &ast.InterpolatedString{Token: token, Parts: parts}
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseGroupedExpression Start. Token: %v\n", p.curToken.Literal)
	}

	if !p.curTokenIs(token.LPAREN) {
		p.reportError("parseGroupedExpression: Expected '(', got %s", p.curToken.Literal)
		return nil
	}

	p.nextToken()
	exp := p.parseExpression(LOWEST)

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
		fmt.Printf("DEBUG: parseBlockStatement Start - Current token: %s, Brace count: %d\n", p.curToken.Literal, p.braceCount)
	}
	block := &ast.BlockStatement{Token: p.curToken}
	block.Statements = []ast.Statement{}

	p.symbolTable.EnterScope()
	defer p.symbolTable.ExitScope()

	p.braceCount++
	p.nextToken() // consume opening brace

	if config.DebugMode {
		fmt.Printf("DEBUG: parseBlockStatement Entering block statement. Brace count: %d\n", p.braceCount)
	}

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		if config.DebugMode {
			fmt.Printf("DEBUG: parseBlockStatement loop - Current token: %s, Brace count: %d\n", p.curToken.Literal, p.braceCount)
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
		if config.DebugMode {
			fmt.Printf("parseBlockStatement: Unexpected EOF, expected '}'. Brace count: %d Line: %d", p.braceCount, p.lastKnownLine)
		}
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
		p.reportError("parseIndexExpression: expected LPAREN, got %v", p.curToken.Literal)
		return nil
	}

	p.nextToken() // move past '(' token
	exp.Index = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		p.reportError("parseIndexExpression: expected RPAREN, got %v", p.curToken.Literal)
		return nil
	}

	return exp
}

func (p *Parser) parseHashLiteral() ast.Expression {
	hash := &ast.HashLiteral{Token: p.curToken}
	hash.Pairs = make(map[ast.StringLiteral]ast.Expression)

	for !p.peekTokenIs(token.RBRACE) {
		p.nextToken()

		// parse key
		if !p.expectPeek(token.STRING) {
			p.reportError("parseHashLiteral: Expected STRING for key, got %v", p.curToken.Literal)
			return nil
		}
		key := &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}

		// parse value
		if !p.expectPeek(token.STRING) {
			p.reportError("parseHashLiteral: Expected STRING for value, got %v", p.curToken.Literal)
			return nil
		}
		value := &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}

		hash.Pairs[*key] = value

		if !p.peekTokenIs(token.RBRACE) && !p.expectPeek(token.COMMA) {
			p.reportError("parseHashLiteral: Expected COMMA, got %v", p.curToken.Literal)
			return nil
		}
	}

	if !p.expectPeek(token.RBRACE) {
		p.reportError("parseHashLiteral: Expected RBRACE, got %v", p.curToken.Literal)
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

	p.nextToken() // move past the opening bracket [

	if config.DebugMode {
		fmt.Printf("DEBUG: parseArrayLiteral after opening bracket. Current token: %s, Type: %s\n", p.curToken.Literal, p.curToken.Type)
	}

	// handle nested array or expression
	for !p.curTokenIs(token.RBRACKET) && !p.curTokenIs(token.EOF) {
		var expr ast.Expression

		if p.curTokenIs(token.IDENT) && p.curToken.Literal == "class" {
			expr = p.parseClassCommand()
			if expr != nil {
				array.Elements = append(array.Elements, expr)
				if config.DebugMode {
					fmt.Printf("DEBUG: parseArrayLiteral - isClass; Added element: %T, curTokenIs: %s\n", expr, p.curToken.Literal)
				}
				// after parsing a class command, we expect to be at the closing bracket
				if !p.curTokenIs(token.RBRACKET) {
					p.reportError("parseArrayLiteral - Expected closing bracket after class command, got %s", p.curToken.Literal)
					return nil
				}
				break // exit the loop as we've parsed the class command
			}
		} else if p.curTokenIs(token.IDENT) && p.curToken.Literal == "string" {
			expr = p.parseStringOperation()
		} else if p.curTokenIs(token.LBRACKET) {
			// handle nested command
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
				fmt.Printf("DEBUG: parseArrayLiteral - Added element: %T, curTokenIs: %s\n", expr, p.curToken.Literal)
			}
		} else {
			p.reportError("parseArrayLiteral: Failed to parse element %T, curTokenIs: %v\n", expr, p.curToken.Literal)
			return nil
		}

		// handle TCL-style command arguments
		for p.peekTokenIs(token.MINUS) {
			p.nextToken() // consume the '-'
			p.nextToken() // move to the argument
			arg := p.parseExpression(LOWEST)
			if arg != nil {
				array.Elements = append(array.Elements, arg)
			}
		}

		// break if we've reached the end of the array
		if p.peekTokenIs(token.RBRACKET) {
			break
		}

		// move to next token if it's not a '-' and not the closing bracket
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
			p.reportError("parseVariableOrArrayAccess:  Expected RPAREN, got %s", p.curToken.Type)
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
		p.reportError("parseWhenNode: Expected HTTP_REQUEST or LB_SELECTED, got %s", p.curToken.Type)
		return nil
	}
	when.Event = p.curToken.Literal

	if !p.expectPeek(token.LBRACE) {
		p.reportError("parseWhenNode: Expected LBRACE, got %s", p.curToken.Type)
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

	// check if the command is a valid HTTP keyword
	if _, isValidHttpCommand := lexer.HttpKeywords[fullCommand]; isValidHttpCommand {
		expr.Command = &ast.Identifier{Token: p.curToken, Value: fullCommand}
	} else {
		p.reportError("parseHttpCommand: Invalid HTTP command: %s", fullCommand)
		if config.DebugMode {
			fmt.Printf("   ERROR: parseHttpCommand - Invalid HTTP command detected: %s\n", fullCommand)
		}
		return nil
	}

	switch {
	case lexer.HttpKeywords[fullCommand] != token.ILLEGAL:
		expr.Command = &ast.Identifier{Token: p.curToken, Value: fullCommand}
	case fullCommand == "HTTP::header":
		expr.Command = &ast.Identifier{Token: p.curToken, Value: "HTTP::header"}
		if p.peekTokenIs(token.IDENT) {
			p.nextToken()
			switch p.curToken.Literal {
			case "names":
				expr.Argument = &ast.Identifier{Token: p.curToken, Value: "names"}
			case "exists":
				expr.Argument = &ast.Identifier{Token: p.curToken, Value: "exists"}
				if p.peekTokenIs(token.STRING) {
					p.nextToken()
					expr.Argument = &ast.ArrayLiteral{
						Token: p.curToken,
						Elements: []ast.Expression{
							&ast.Identifier{Token: p.curToken, Value: "exists"},
							&ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal},
						},
					}
				}
			default:
				expr.Argument = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
			}
		} else if p.peekTokenIs(token.STRING) {
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

	// check for additional arguments
	for p.peekTokenIs(token.STRING) {
		p.nextToken()
		if expr.Argument == nil {
			expr.Argument = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
		} else {
			// if there's already an argument, create a list of arguments
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

	// expect '{'
	if !p.expectPeek(token.LBRACE) {
		p.reportError("parseIfStatement: Expected {, got %s", p.curToken.Literal)
		return nil
	}

	p.nextToken() // consume '{'
	if config.DebugMode {
		fmt.Printf("DEBUG: parseIfStatement - Parsing condition, current token: %s\n", p.curToken.Literal)
	}

	// parse the condition
	stmt.Condition = p.parseComplexCondition()

	// check for matches_regex
	if p.peekTokenIs(token.IDENT) && p.peekToken.Literal == "matches_regex" {
		p.nextToken() // move to matches_regex
		regexExp := p.parseMatchesRegexExpression(stmt.Condition)
		stmt.Condition = regexExp
	}

	if !p.expectPeek(token.RBRACE) {
		p.reportError("parseIfStatement: Expected } after condition, got %s", p.peekToken.Literal)
		return nil
	}

	if p.peekTokenIs(token.LBRACE) {
		p.nextToken() // move to '{'
		stmt.Consequence = p.parseBlockStatement()
	} else {
		// handle empty body
		stmt.Consequence = &ast.BlockStatement{Token: p.curToken, Statements: []ast.Statement{}}
	}

	// parse else-if and else clauses
	currentStmt := stmt
	for p.peekTokenIs(token.ELSEIF) || p.peekTokenIs(token.ELSE) {
		p.nextToken() // consume 'elseif' or 'else'

		if p.curTokenIs(token.ELSEIF) {
			elseIfStmt := &ast.IfStatement{Token: p.curToken}

			if !p.expectPeek(token.LBRACE) {
				p.reportError("parseIfStatement: ELSEIF Expected {, got %s", p.curToken.Literal)
				return nil
			}

			p.nextToken() // consume '{'

			elseIfStmt.Condition = p.parseExpression(LOWEST)

			// check for matches_regex in else-if condition
			if p.peekTokenIs(token.IDENT) && p.peekToken.Literal == "matches_regex" {
				p.nextToken() // Move to matches_regex
				regexExp := p.parseMatchesRegexExpression(elseIfStmt.Condition)
				elseIfStmt.Condition = regexExp
				// expect '}' after matches_regex
				if !p.expectPeek(token.RBRACE) {
					p.reportError("parseIfStatement: ELSEIF Expected } after matches_regex, got %s", p.curToken.Literal)
					return nil
				}
			} else {
				// expect '}'
				if !p.expectPeek(token.RBRACE) {
					p.reportError("parseIfStatement: ELSEIF Expected }, got %s", p.curToken.Literal)
					return nil
				}
			}

			// expect '{' for else-if consequence block
			if !p.expectPeek(token.LBRACE) {
				p.reportError("parseIfStatement: ELSEIF Consequence Expected {, got %s", p.curToken.Literal)
				return nil
			}
			elseIfStmt.Consequence = p.parseBlockStatement()

			// add the else-if statement as an alternative to the current statement
			currentStmt.Alternative = &ast.BlockStatement{
				Statements: []ast.Statement{elseIfStmt},
			}
			currentStmt = elseIfStmt
		} else if p.curTokenIs(token.ELSE) {
			// parse the final else clause
			if !p.expectPeek(token.LBRACE) {
				p.reportError("parseIfStatement: ELSE Expected {, got %s", p.curToken.Literal)
				return nil
			}
			currentStmt.Alternative = p.parseBlockStatement()
			break // exit the loop after parsing the final else
		}
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseIfStatement End - Condition: %T, Current token: %s\n", stmt.Condition, p.curToken.Literal)
	}

	return stmt
}

func (p *Parser) parseWhenExpression() ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseWhenExpression Start\n")
	}
	expr := &ast.WhenExpression{Token: p.curToken}

	// check if the next token is a valid expression token
	if p.isValidWhenEvent(token.TokenType(p.peekToken.Literal)) {
		p.nextToken() // advance to the event token
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
		fmt.Printf("DEBUG: Start parseSwitchStatement at line %d\n", p.lastKnownLine)
	}
	switchStmt := &ast.SwitchStatement{Token: p.curToken}
	switchStmt.IsRegex = false
	switchStmt.IsGlob = false

	// parse switch options and value
	p.nextToken() // move past 'switch'

	// handle options like -glob
	for p.curTokenIs(token.MINUS) {
		option := p.curToken.Literal
		p.nextToken() // move past the option
		if p.curTokenIs(token.IDENT) {
			option += p.curToken.Literal
			switchStmt.Options = append(switchStmt.Options, option)
			if option == "-regex" {
				switchStmt.IsRegex = true
			} else if option == "-glob" {
				switchStmt.IsGlob = true
			}
			p.nextToken() // move past the option value
		}
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: Switch type - isRegex: %v, isGlob: %v\n", switchStmt.IsRegex, switchStmt.IsGlob)
	}

	// handle the -- separator if present
	if p.curTokenIs(token.MINUS) && p.peekTokenIs(token.MINUS) {
		p.nextToken() // move past first -
		p.nextToken() // move past second -
	}

	// parse the switch value (which might be a string operation)
	switchStmt.Value = p.parseExpression(LOWEST)

	if !p.expectPeek(token.LBRACE) {
		p.reportError("parseSwitchStatement: expected LBRACE")
		return nil
	}

	switchStmt.Cases = []*ast.CaseStatement{}

	p.nextToken() // Move past the opening brace

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		// line := p.lastKnownLine
		if config.DebugMode {
			fmt.Printf("DEBUG: parseSwitchStatement Switch loop - Current token: %s, Literal: %s, Line: %d\n", p.curToken.Type, p.curToken.Literal, p.lastKnownLine)
		}

		if p.curTokenIs(token.DEFAULT) {
			switchStmt.Default = p.parseDefaultCase()
		} else if p.curTokenIs(token.STRING) {
			// string-based syntax i.e. "/api*"
			if config.DebugMode {
				fmt.Printf("DEBUG: parseSwitchStatement Before calling parseStringCaseStatement - Token: %+v\n", p.curToken)
			}
			caseStmt := p.parseStringCaseStatement()
			if caseStmt != nil {
				switchStmt.Cases = append(switchStmt.Cases, caseStmt)
				caseStmt.Line = p.curToken.Line
				if config.DebugMode {
					fmt.Printf("DEBUG: parseSwitchStatement StringCase Adding case statement with pattern '%s' at line %d\n", caseStmt.Value, caseStmt.Line)
				}
			}
		} else {
			p.reportError(fmt.Sprintf("parseSwitchStatement: Invalid case statement starting with token: %s", p.curToken.Literal))
			return nil // error occurred in parsing case statement
		}

		// ensure we're moving forward after each case
		p.nextToken()
	}

	if config.DebugMode {
		fmt.Println("DEBUG: parseStringCaseStatement: Cases before validation:")
		for i, caseStmt := range switchStmt.Cases {
			fmt.Printf("  Case %d: Pattern '%s' at line %d\n", i, caseStmt.Value, caseStmt.Line)
		}
	}
	if err := p.validateSwitchPatterns(switchStmt); err != nil {
		p.reportError(err.Error())
		return nil
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
	case token.IP_REMOTE_ADDR:
		expression.Function = "remote_addr"
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
			// handle nested command
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

		// stop parsing if we encounter an 'if' statement or other control structures
		if p.peekTokenIs(token.IF) || p.peekTokenIs(token.LBRACE) {
			break
		}

		p.nextToken()

		// break if we've reached the end of this command
		if p.curTokenIs(token.RBRACKET) {
			break
		}
	}

	// combine all parts into a single command string
	command.Command = &ast.Identifier{Token: command.Token, Value: strings.Join(commandParts, " ")}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseLoadBalancerCommand End. Command: %v\n", command.Command.Value)
	}

	return command
}

func (p *Parser) isHttpKeyword(tokenType token.TokenType) bool {
	for _, httpTokenType := range lexer.HttpKeywords {
		if tokenType == httpTokenType {
			return true
		}
	}
	return false
}

func (p *Parser) isLbKeyword(tokenType token.TokenType) bool {
	for _, lbTokenType := range lexer.LbKeywords {
		if tokenType == lbTokenType {
			return true
		}
	}
	return false
}

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

	p.nextToken() // move past 'string'
	operation := p.curToken.Literal
	if config.DebugMode {
		fmt.Printf("DEBUG: parseStringOperation Operation: %v\n", stringOp.Operation)
	}

	// validate the operation
	if !validStringOperations[operation] {
		p.reportError("parseStringOperation: Invalid string operation: %s", operation)
		return nil
	}

	stringOp.Operation = operation

	var args []ast.Expression
	for p.peekToken.Type != token.RBRACKET && p.peekToken.Type != token.EOF {
		p.nextToken()
		if p.curTokenIs(token.MINUS) && p.peekTokenIs(token.IDENT) {
			args = append(args, &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal + p.peekToken.Literal})
			p.nextToken() // skip the identifier after '-'
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

	// perform checks based on the operation
	switch operation {
	case "match":
		if len(args) != 2 {
			p.errors = append(p.errors, fmt.Sprintf("line %d: 'string match' expects 2 arguments", p.curToken.Line))
		} else {
			p.checkVariableUsage(args[1], "second argument of 'string match'")
		}
	}

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
		p.nextToken() // move to the key
		key := p.parseExpression(LOWEST)

		if !p.expectPeek(token.STRING) {
			p.reportError("parseMapArgument: expected STRING, got %v", p.curToken.Literal)
			return nil
		}

		value := p.parseExpression(LOWEST)
		mapArg.Pairs[key] = value

		if !p.peekTokenIs(token.RBRACE) && !p.expectPeek(token.COMMA) {
			p.reportError("parseMapArgument: expected RBRACE or COMMA, got %v", p.curToken.Literal)
			return nil
		}
	}

	if !p.expectPeek(token.RBRACE) {
		p.reportError("parseMapArgument: expected RBRACE, got %v", p.curToken.Literal)
		return nil
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseMapArgument End\n")
	}

	return mapArg
}

func (p *Parser) parsePoolStatement() ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: parsePoolStatement Start - Current token: %s, Line: %d\n", p.curToken.Type, p.currentLine)
	}

	p.symbolTable.Declare(p, POOL)

	poolStmt := &ast.CallExpression{
		Token:    p.curToken,
		Function: &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal},
	}

	if !p.expectPeek(token.IDENT) {
		p.reportError("parsePoolStatement: Expected IDENT, got %v", p.curToken.Literal)
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
		fmt.Printf("DEBUG: parseClassCommand Start - curToken: %s (Type: %s), peekToken: %s (Type: %s)\n",
			p.curToken.Literal, p.curToken.Type, p.peekToken.Literal, p.peekToken.Type)
	}

	p.isParsingClassMatch = true
	defer func() { p.isParsingClassMatch = false }()

	cmd := &ast.ClassCommand{Token: p.curToken}

	// advance to the subcommand
	if !p.expectPeek(token.MATCH) {
		p.reportError("parseClassCommand: Expected 'match', got %s", p.curToken.Literal)
		return nil
	}

	cmd.Subcommand = p.curToken.Literal

	// parse the variable
	if !p.expectPeek(token.IDENT) {
		p.reportError("parseClassCommand: Expected variable, got %s", p.curToken.Literal)
		return nil
	}
	variable := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	cmd.Arguments = append(cmd.Arguments, variable)

	// parse the operator
	if !p.expectPeek(token.EQ) {
		p.reportError("parseClassCommand: Expected operator '==', got %s", p.curToken.Literal)
		return nil
	}
	operator := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	cmd.Arguments = append(cmd.Arguments, operator)

	// parse the value
	if !p.expectPeek(token.IDENT) {
		p.reportError("parseClassCommand: Expected value, got %s", p.curToken.Literal)
		return nil
	}
	value := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	cmd.Arguments = append(cmd.Arguments, value)

	if config.DebugMode {
		fmt.Printf("DEBUG: parseClassCommand End - Subcommand: %s, Arguments: %v\n", cmd.Subcommand, cmd.Arguments)
	}

	return cmd
}

func (p *Parser) parseStringLiteralContents(s *ast.StringLiteral) ast.Expression {
	if s == nil || s.Value == "" {
		return nil
	}
	if config.DebugMode {
		fmt.Printf("DEBUG: parseStringLiteralContents Start - Value: %s\n", s.Value)
	}
	return s
}

func (p *Parser) parseForEachStatement() ast.Statement {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseForEachStatement Start\n")
	}
	stmt := &ast.ForEachStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		p.reportError("parseForEachStatement: expected IDENT, got %v", p.curToken.Literal)
		return nil
	}

	stmt.Variable = p.curToken.Literal
	if config.DebugMode {
		fmt.Printf("DEBUG: parseForEachStatement Variable: %v\n", stmt.Variable)
	}
	p.declareVariable(stmt.Variable)

	p.nextToken() // move to the list expression

	// parse the list expression
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
		p.reportError("parseForEachStatement: Expected LBRACE, got %v", p.curToken.Literal)
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
		fmt.Printf("DEBUG: parseListLiteral Start. Current token: %s\n", p.curToken.Literal)
	}
	list := &ast.ListLiteral{Token: p.curToken}
	list.Elements = []ast.Expression{}

	p.nextToken() // move past '{'
	if config.DebugMode {
		fmt.Printf("DEBUG: parseListLiteral after opening brace. Current token: %s\n", p.curToken.Literal)
	}

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		if config.DebugMode {
			fmt.Printf("DEBUG: parseListLiteral parsing element. Current token: %s\n", p.curToken.Literal)
		}
		elem := p.parseExpression(LOWEST)
		if elem != nil {
			list.Elements = append(list.Elements, elem)
			if config.DebugMode {
				fmt.Printf("DEBUG: parseListLiteral added element: %T\n", elem)
			}
		} else {
			p.reportError("parseListLiteral: Failed to parse statement")
			if config.DebugMode {
				fmt.Printf("DEBUG: parseListLiteral failed to parse element\n")
			}
		}

		if p.peekTokenIs(token.RBRACE) || p.peekTokenIs(token.EOF) {
			if config.DebugMode {
				fmt.Printf("DEBUG: parseListLiteral breaking loop. Peek token: %v\n", p.peekToken.Literal)
			}
			break
		}

		// if the peek token is empty, move to the next token
		if p.peekToken.Literal == "" {
			if config.DebugMode {
				fmt.Printf("DEBUG: parseListLiteral encountered empty peek token, moving to next\n")
			}
			p.nextToken()
			continue
		}

		// if we're not at the end of the list, expect a comma or space
		if !p.peekTokenIs(token.COMMA) && !p.peekTokenIs(token.SPACE) {
			if config.DebugMode {
				fmt.Printf("DEBUG: parseListLiteral unexpected token. Peek token: %s\n", p.peekToken.Literal)
			}
			p.nextToken()
		} else {
			if config.DebugMode {
				fmt.Printf("DEBUG: parseListLiteral consuming separator. Peek token: %s\n", p.peekToken.Literal)
			}
			p.nextToken() // consume the comma or space
		}
	}

	if p.curTokenIs(token.EOF) {
		if config.DebugMode {
			fmt.Printf("WARNING: parseListLiteral reached EOF before finding closing brace\n")
		}
		p.reportError("parseListLiteral - Unexpected EOF, missing closing brace")
		return list
	}

	if !p.expectPeek(token.RBRACE) {
		p.reportError("parseListLiteral: Expected RBRACE brace, got %s", p.curToken.Literal)
		return list
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseListLiteral End. List elements: %d\n", len(list.Elements))
	}
	return list
}

func (p *Parser) isValidIRuleIdentifier(value string, identifierContext string) (bool, error) {
	if config.DebugMode {
		fmt.Printf("DEBUG: isValidIRuleIdentifier - Start. Value=%v, Context=%v\n", value, identifierContext)
	}

	// check for reserved keywords
	if reservedKeywords[strings.ToLower(value)] {
		if identifierContext == "variable" {
			return false, fmt.Errorf("ERROR: isValidIRuleIdentifier - '%s' is a reserved keyword and should not be used as a variable name", value)
		}
		if config.DebugMode {
			fmt.Printf("DEBUG: isValidIRuleIdentifier - Using reserved keyword '%s' in context '%s'\n", value, identifierContext)
		}
		return true, nil
	}

	// check if it's a variable (starts with $)
	if strings.HasPrefix(value, "$") {
		if config.DebugMode {
			fmt.Printf("DEBUG: isValidIRuleIdentifier - %s is a variable\n", value)
		}
		return true, nil
	}

	// check if it's a common iRule identifier or command
	if isCommonIRuleIdentifier(value) {
		if config.DebugMode {
			fmt.Printf("DEBUG: isValidIRuleIdentifier - %s is a common iRule identifier or command\n", value)
		}
		return true, nil
	}

	// check context-specific validations
	switch identifierContext {
	case "variable":
		// stricter check for variable names
		if regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(value) {
			if config.DebugMode {
				fmt.Printf("DEBUG: isValidIRuleIdentifier - %s is a valid variable identifier\n", value)
			}
			return true, nil
		}
		return false, fmt.Errorf("invalid variable identifier: %s", value)

	case "standalone", "class_match", "class_lookup", "pool_name", "event_name", "profile_name",
		"vs_name", "node_name", "monitor_name", "ssl_profile", "table_name", "proc_name":
		if regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(value) {
			if config.DebugMode {
				fmt.Printf("DEBUG: isValidIRuleIdentifier - %s is a valid identifier in context %s\n", value, identifierContext)
			}
			return true, nil
		}

		// additional checks for standalone context
		if identifierContext == "standalone" {
			// allow single-letter identifiers and check against common headers (case-insensitive)
			if len(value) == 1 && regexp.MustCompile(`^[a-zA-Z]$`).MatchString(value) {
				if config.DebugMode {
					fmt.Printf("DEBUG: isValidIRuleIdentifier - %s is a valid single-letter identifier\n", value)
				}
				return true, nil
			}
			// check against common headers (case-insensitive)
			for _, header := range commonHeaders {
				if strings.EqualFold(value, header) {
					if config.DebugMode {
						fmt.Printf("DEBUG: isValidIRuleIdentifier - %s is a valid common header\n", value)
					}
					return true, nil
				}
			}

			// check for command patterns (e.g., HTTP::*)
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
		}

	case "header":
		if isValidHeaderName(value) {
			if config.DebugMode {
				fmt.Printf("DEBUG: isValidIRuleIdentifier - %s is a valid HTTP header name\n", value)
			}
			return true, nil
		}
		return false, fmt.Errorf("invalid HTTP header name: %s", value)

	case "glob_pattern":
		return true, nil
	}

	// check if it's a valid command or keyword
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

	// check if it's a valid custom identifier (declared variable or function)
	if p.isValidCustomIdentifier(value) {
		return true, nil
	}

	// check if it's a valid logging facility
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
		return true // allow partial expressions during parsing
	}

	switch operator {
	case "contains", "starts_with", "ends_with", "equals":
		// these operators are valid for strings, HTTP expressions, array literals, IP address literals, and identifiers
		return (isStringType(left) || isHttpExpression(left) || isArrayLiteral(left) || isIpAddressLiteral(left) || isIdentifier(left)) &&
			(isStringType(right) || isHttpExpression(right) || isArrayLiteral(right) || isIpAddressLiteral(right) || isIdentifier(right))
	case "eq", "ne", "==", "!=":
		// equality operators are valid for most types
		return true
	case "<", ">", "<=", ">=":
		// comparison operators are valid for numbers and strings
		return (isNumberType(left) && isNumberType(right)) || (isStringType(left) && isStringType(right)) ||
			(isIdentifier(left) && isStringType(right)) || (isStringType(left) && isIdentifier(right))
	case "+", "-", "*", "/":
		// arithmetic operators are valid for numbers, infix expressions, array literals, and identifiers
		return (isNumberType(left) || isInfixExpression(left) || isArrayLiteral(left) || isIdentifier(left)) &&
			(isNumberType(right) || isInfixExpression(right) || isIdentifier(right))
	case "&&", "||":
		// logical operators are valid for boolean expressions, HTTP expressions, and identifiers
		return isBooleanType(left) || isHttpExpression(left) || isInfixExpression(left) || isIdentifier(left) ||
			isBooleanType(right) || isHttpExpression(right) || isInfixExpression(right) || isIdentifier(right)
	default:
		return true // allow unknown operators to be handled elsewhere
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
		return e.IsVariable // assume variables can be strings
	case *ast.HttpExpression, *ast.LoadBalancerExpression, *ast.SSLExpression:
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
	if p.declaredVariables[s] {
		return true
	}

	// check if it's a valid function name (assuming functions are declared with "proc")
	if strings.HasPrefix(s, "proc ") {
		return true
	}

	return false
}

func (p *Parser) reportError(format string, args ...interface{}) {
	var line int
	var msg string

	if len(args) > 0 {
		if lastArg, ok := args[len(args)-1].(int); ok {
			// if the last argument is an int, use it as the line number
			line = lastArg
			msg = fmt.Sprintf(format, args[:len(args)-1]...)
		} else {
			// if the last argument is not an int, use all args for the message
			line = p.lastKnownLine
			msg = fmt.Sprintf(format, args...)
		}
	} else {
		line = p.lastKnownLine
		msg = format
	}

	lineMsg := fmt.Sprintf("   %s, Line: %d", msg, line)
	p.errors = append(p.errors, lineMsg)
}

func (p *Parser) parseNodeStatement() ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseNodeStatement Start - Current token: %s, Line: %d\n", p.curToken.Type, p.l.CurrentLine())
	}

	p.symbolTable.Declare(p, NODE)

	nodeStmt := &ast.NodeStatement{
		Token: p.curToken,
	}

	// expect the next token to be an IP address
	if !p.expectPeek(token.IP_ADDRESS) {
		p.reportError("parseNodeStatement: expected IP_ADDRESS, got %v", p.curToken.Literal)
		return nil
	}
	nodeStmt.IPAddress = p.curToken.Literal

	// expect the next token to be a port number, but don't require it
	if p.peekTokenIs(token.NUMBER) {
		p.nextToken()
		nodeStmt.Port = p.curToken.Literal
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseNodeStatement End - IP: %s, Port: %s\n", nodeStmt.IPAddress, nodeStmt.Port)
	}

	return nodeStmt
}

func (p *Parser) parseLtmRule() ast.Statement {

	if config.DebugMode {
		fmt.Printf("DEBUG: parseLtmRule Start - Current token: %s, Line: %d\n", p.curToken.Type, p.l.CurrentLine())
	}
	stmt := &ast.LtmRule{Token: p.curToken}

	if !p.expectPeek(token.RULE) {
		p.reportError("parseLtmRule: expected RULE, got %v", p.curToken.Literal)
		return nil
	}

	if !p.expectPeek(token.IDENT) {
		p.reportError("parseLtmRule: expected IDENT, got %v", p.curToken.Literal)
		return nil
	}

	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.LBRACE) {
		p.reportError("parseLtmRule: expected LBRACE, got %v", p.curToken.Literal)
		return nil
	}

	stmt.Body = p.parseBlockStatement()
	if config.DebugMode {
		fmt.Printf("DEBUG: parseLtmRule End - Current token: %s, Line: %d\n", p.curToken.Type, p.l.CurrentLine())
	}

	return stmt
}

func (p *Parser) parseSlashExpression() ast.Expression {
	return &ast.SlashExpression{Token: p.curToken}
}

func isValidGlobPattern(pattern string) bool {
	result := len(pattern) > 0
	if config.DebugMode {
		fmt.Printf("DEBUG: isValidGlobPattern(%s) = %v\n", pattern, result)
	}
	return result
}

func isValidRegexPattern(pattern string) bool {
	_, err := regexp.Compile(pattern)
	result := err == nil
	if config.DebugMode {
		fmt.Printf("DEBUG: isValidRegexPattern(%s) = %v\n", pattern, result)
	}
	return result
}

func (p *Parser) parseStringCaseStatement() *ast.CaseStatement {
	if config.DebugMode {
		fmt.Printf("DEBUG: Start parseStringCaseStatement at line %d\n", p.currentLine)
	}

	caseStmt := &ast.CaseStatement{Token: p.curToken, Line: p.curToken.Line}

	pattern := p.parseExpression(LOWEST)
	startPattern, ok := pattern.(*ast.StringLiteral)
	if !ok {
		p.reportError(fmt.Sprintf("parseStringCaseStatement: Expected string literal for case pattern, got %T", pattern))
		return nil
	}

	// check for range case
	if p.peekTokenIs(token.MINUS) {
		p.nextToken() // consume the '-'
		p.nextToken() // move to the end range token
		endPattern := p.parseExpression(LOWEST)
		endStringLiteral, ok := endPattern.(*ast.StringLiteral)
		if !ok {
			p.reportError(fmt.Sprintf("parseStringCaseStatement: Expected string literal for range end, got %T", endPattern))
			return nil
		}

		// Create a MultiPattern for the range
		caseStmt.Value = &ast.MultiPattern{
			Patterns: []ast.Expression{
				startPattern,
				endStringLiteral,
			},
		}
	} else {
		caseStmt.Value = startPattern
	}

	if !p.expectPeek(token.LBRACE) {
		p.reportError("parseStringCaseStatement: Expected '{' after case pattern")
		return nil
	}

	caseStmt.Consequence = p.parseBlockStatement()

	if config.DebugMode {
		fmt.Printf("DEBUG: End parseStringCaseStatement, created case with pattern '%v' at line %d\n", caseStmt.Value, caseStmt.Line)
	}

	return caseStmt
}

func isGlobPattern(pattern string) bool {
	result := strings.ContainsAny(pattern, "*?") && !strings.ContainsAny(pattern, "(){}|^$+\\")
	if config.DebugMode {
		fmt.Printf("DEBUG: isGlobPattern(%s) = %v\n", pattern, result)
	}
	return result
}

func isRegexPattern(pattern string) bool {
	result := strings.ContainsAny(pattern, "^$+(){}|") || strings.Contains(pattern, ".*")
	if config.DebugMode {
		fmt.Printf("DEBUG: isRegexPattern(%s) = %v\n", pattern, result)
	}
	return result
}

func (p *Parser) validateSwitchPatterns(switchStmt *ast.SwitchStatement) error {
	if config.DebugMode {
		fmt.Printf("DEBUG: Start validateSwitchPatterns - isRegex: %v, isGlob: %v\n", switchStmt.IsRegex, switchStmt.IsGlob)
	}
	for i, caseStmt := range switchStmt.Cases {
		var pattern []string
		line := p.lastKnownLine

		if config.DebugMode {
			fmt.Printf("DEBUG: validateSwitchPatterns Case %d - Token: %+v, Line: %d\n", i, caseStmt.Token, line)
		}

		switch v := caseStmt.Value.(type) {
		case *ast.StringLiteral:
			pattern = []string{v.Value}
			if v.Token.Line > 0 {
				line = v.Token.Line
			}
			if config.DebugMode {
				fmt.Printf("DEBUG: StringLiteral pattern: %s, Token: %+v\n", pattern, v.Token)
			}
		case *ast.GlobPattern:
			pattern = []string{v.Value}
			if v.Token.Line > 0 {
				line = v.Token.Line
			}
			if config.DebugMode {
				fmt.Printf("DEBUG: GlobPattern pattern: %s, Token: %+v\n", pattern, v.Token)
			}
		case *ast.RegexPattern:
			pattern = []string{v.Value}
			if v.Token.Line > 0 {
				line = v.Token.Line
			}
			if config.DebugMode {
				fmt.Printf("DEBUG: GlobPattern pattern: %s, Token: %+v\n", pattern, v.Token)
			}
		case *ast.MultiPattern:
			for _, p := range v.Patterns {
				switch pv := p.(type) {
				case *ast.StringLiteral:
					pattern = append(pattern, pv.Value)
				case *ast.GlobPattern:
					pattern = append(pattern, pv.Value)
				default:
					return fmt.Errorf("unexpected pattern type: %T", p)
				}
			}
		default:
			return fmt.Errorf("unexpected case value type: %T", caseStmt.Value)
		}

		for _, pattern := range pattern {
			if config.DebugMode {
				fmt.Printf("DEBUG: Validating pattern: '%s' at line %d\n", pattern, line)
			}

			if switchStmt.IsRegex {
				if isGlobPattern(pattern) {
					p.reportError("Invalid regex pattern (looks like a glob pattern): %s", []interface{}{pattern, line}...)
				}
				if !isValidRegexPattern(pattern) {
					p.reportError("Invalid regex pattern: %s", []interface{}{pattern, line}...)
				}
			} else if switchStmt.IsGlob {
				if isRegexPattern(pattern) {
					p.reportError("Invalid glob pattern (looks like a regex pattern): %s", []interface{}{pattern, line}...)
				} else if !isValidGlobPattern(pattern) {
					p.reportError("Invalid glob pattern: %s Line: %d", pattern, line)
				}
			}
		}
	}

	if config.DebugMode {
		fmt.Println("DEBUG: End validateSwitchPatterns - All patterns valid")
	}
	return nil
}

func (p *Parser) parseMatchesRegexExpression(left ast.Expression) ast.Expression {
	if config.DebugMode {
		fmt.Printf("DEBUG: parseMatchesRegexExpression Start\n")
	}

	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Left:     left,
		Operator: "matches_regex",
	}

	if !p.expectPeek(token.REGEX) {
		p.reportError("parseMatchesRegexExpression: expected REGEX, got %v", p.curToken.Literal)
		return nil
	}

	regexPattern := p.curToken.Literal

	if !isValidRegexPattern(regexPattern) {
		p.reportError(fmt.Sprintf("Invalid regex pattern: %s", regexPattern))
		return nil
	}

	expression.Right = &ast.RegexPattern{
		Token: p.curToken,
		Value: p.curToken.Literal,
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: parseMatchesRegexExpression End, pattern: %s\n", expression.Right)
	}

	return expression
}

func (p *Parser) parseRegexLiteral() ast.Expression {
	return &ast.RegexPattern{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseComplexCondition() ast.Expression {
	expr := p.parseExpression(LOWEST)

	for p.peekTokenIs(token.OR) || p.peekTokenIs(token.AND) {
		operator := p.peekToken.Literal
		p.nextToken() // consume 'or' or 'and'
		p.nextToken() // move to the next token after 'or' or 'and'
		right := p.parseExpression(LOGICAL)

		expr = &ast.InfixExpression{
			Token:    p.curToken,
			Left:     expr,
			Operator: operator,
			Right:    right,
		}
	}

	return expr
}

func (p *Parser) checkVariableUsage(arg ast.Expression, context string) {
	switch expr := arg.(type) {
	case *ast.Identifier:
		if expr.Value[0] == '$' {
			// it's a variable reference, check if it's declared
			varName := expr.Value[1:] // Remove the $
			if !p.declaredVariables[varName] {
				p.reportError("checkVariableUsage - undeclared variable %s used in %s", expr.Value, context)
			}
		} else {
			// it's not a variable reference, but it should be
			if p.declaredVariables[expr.Value] {
				p.reportError("checkVariableUsage - %s should be referenced as $%s in %s", expr.Value, expr.Value, context)
			} else {
				p.reportError("checkVariableUsage - expected variable reference in %s, got %s", context, expr.Value)
			}
		}
	default:
		// it's not an identifier at all
		p.reportError("checkVariableUsage -expected variable reference in %s", context)
	}
}
