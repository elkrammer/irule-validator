package lexer

import (
	"fmt"

	"github.com/elkrammer/irule-validator/config"
	"github.com/elkrammer/irule-validator/token"
)

type Lexer struct {
	input         string
	position      int      // current position in input (points to current char)
	readPosition  int      // current reading position in input (after current char)
	ch            byte     // current char under examination
	braceDepth    int      // current depth in block statements
	line          int      // current line number
	errors        []string // catch lexing errors
	inSwitchBlock bool
}

var HttpKeywords = map[string]token.TokenType{
	"HTTP_REQUEST":   token.HTTP_REQUEST,
	"HTTP::uri":      token.HTTP_URI,
	"HTTP::host":     token.HTTP_HOST,
	"HTTP::cookie":   token.HTTP_COOKIE,
	"HTTP::redirect": token.HTTP_REDIRECT,
	"HTTP::header":   token.HTTP_HEADER,
	"HTTP::respond":  token.HTTP_RESPOND,
	"HTTP::method":   token.HTTP_METHOD,
	"HTTP::path":     token.HTTP_PATH,
	"HTTP::query":    token.HTTP_QUERY,
}

var LbKeywords = map[string]token.TokenType{
	"LB_SELECTED":  token.LB_SELECTED,
	"LB_FAILED":    token.LB_FAILED,
	"LB_QUEUED":    token.LB_QUEUED,
	"LB_COMPLETED": token.LB_COMPLETED,
	"LB::mode":     token.LB_MODE,
	"LB::select":   token.LB_SELECT,
	"LB::reselect": token.LB_RESELECT,
	"LB::detach":   token.LB_DETACH,
	"LB::server":   token.LB_SERVER,
	"LB::pool":     token.LB_POOL,
	"LB::status":   token.LB_STATUS,
	"LB::alive":    token.LB_ALIVE,
	"LB::persist":  token.LB_PERSIST,
	"LB::method":   token.LB_METHOD,
	"LB::score":    token.LB_SCORE,
	"LB::priority": token.LB_PRIORITY,
	"LB::connect":  token.LB_CONNECT,
	"LB::bias":     token.LB_BIAS,
	"LB::snat":     token.LB_SNAT,
	"LB::limit":    token.LB_LIMIT,
	"LB::class":    token.LB_CLASS,
}

var SSLKeywords = map[string]token.TokenType{
	"SSL::cipher":         token.SSL_CIPHER,
	"SSL::cipher_bits":    token.SSL_CIPHER_BITS,
	"SSL::clienthello":    token.SSL_CLIENTHELLO,
	"SSL::serverhello":    token.SSL_SERVERHELLO,
	"SSL::cert":           token.SSL_CERT,
	"SSL::verify_result":  token.SSL_VERIFY_RESULT,
	"SSL::sessionid":      token.SSL_SESSIONID,
	"SSL::renegotiate":    token.SSL_RENEGOTIATE,
	"SSL::sessionvalid":   token.SSL_SESSIONVALID,
	"SSL::sessionupdates": token.SSL_SESSIONUPDATES,
}

func New(input string) *Lexer {
	l := &Lexer{input: input, line: 1}
	l.readChar()
	if config.DebugMode {
		fmt.Printf("DEBUG: Lexer initialized with input length: %d\n", len(input))
	}
	return l
}

// read one forward character
func (l *Lexer) readChar() {
	// if config.DebugMode {
	// 	fmt.Printf(">>> readChar: BEFORE - l.ch: %q(%d), l.position: %d, l.readPosition: %d\n", l.ch, l.ch, l.position, l.readPosition)
	// }
	if l.readPosition >= len(l.input) {
		l.ch = 0
		if config.DebugMode {
			fmt.Printf("DEBUG: Reached EOF in lexer at position %d. Line: %d\n", l.position, l.line)
		}
	} else {
		l.ch = l.input[l.readPosition]
		// if config.DebugMode {
		// 	fmt.Printf(">>> readChar: Reading l.input[%d] = %q (%d)\n", l.readPosition, l.ch, l.ch)
		// }
	}
	l.position = l.readPosition
	l.readPosition += 1

	// update line number
	if l.ch == '\n' {
		l.line++
	}
	// if config.DebugMode {
	// 	fmt.Printf(">>> readChar: AFTER  - l.ch: %q(%d), l.position: %d, l.readPosition: %d\n", l.ch, l.ch, l.position, l.readPosition)
	// }
}

func newToken(tokenType token.TokenType, ch byte, line int) token.Token {
	return token.Token{Type: tokenType, Literal: string(ch), Line: line}
}

func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	// if config.DebugMode {
	// 	fmt.Printf("DEBUG LEXER: NextToken() Entry - l.ch: %q, l.position: %d, l.readPosition: %d\n", l.ch, l.position, l.readPosition)
	// }

	l.skipWhitespace()

	// check for comments
	if l.ch == '#' || (l.ch == '/' && l.peekChar() == '/') {
		if l.inSwitchBlock {
			l.reportError("Comments are not allowed in switch statement")
			l.skipComment()
			return token.Token{
				Type:    token.SKIP_TO_NEXT_CASE,
				Literal: "SKIP_TO_NEXT_CASE",
				Line:    l.line,
			}
		}
		l.skipComment()
		return l.NextToken()
	}

	switch l.ch {
	case '\n':
		l.line++
		return l.NextToken()
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = token.Token{Type: token.EQ, Literal: literal, Line: l.line}
			if config.DebugMode {
				fmt.Printf("DEBUG: Lexer produced EQ token in case '=': %v\n", tok)
			}

		} else {
			tok = newToken(token.ASSIGN, l.ch, l.line)
		}
	case '{':
		if l.peekChar() == '^' {
			// this is likely the start of a regex pattern
			pattern := l.readRegexPattern()
			tok = token.Token{Type: token.REGEX, Literal: pattern}
		} else {
			tok = newToken(token.LBRACE, l.ch, l.line)
			l.braceDepth++
		}
		if config.DebugMode {
			fmt.Printf("DEBUG: Lexer identified opening brace '{', depth now %d\n", l.braceDepth)
		}
	case '}':
		tok = newToken(token.RBRACE, l.ch, l.line)
		l.braceDepth--
		if config.DebugMode {
			fmt.Printf("DEBUG: Lexer identified closing brace '}', depth now %d\n", l.braceDepth)
		}
	case '(':
		tok = newToken(token.LPAREN, l.ch, l.line)
	case ')':
		tok = newToken(token.RPAREN, l.ch, l.line)
	case '[':
		tok = newToken(token.LBRACKET, l.ch, l.line)
	case ']':
		tok = newToken(token.RBRACKET, l.ch, l.line)
	case ',':
		tok = newToken(token.COMMA, l.ch, l.line)
	case '%':
		tok = newToken(token.PERCENT, l.ch, l.line)
	case '^':
		tok = newToken(token.CARET, l.ch, l.line)
	case '$':
		tok.Type = token.IDENT
		tok.Literal = l.readVariable()
		return tok
	case '"', '\'':
		tok.Type = token.STRING
		tok.Literal = l.readString()
	case '+':
		tok = newToken(token.PLUS, l.ch, l.line)
	case ';':
		tok = newToken(token.SEMICOLON, l.ch, l.line)
	case '<':
		tok = newToken(token.LT, l.ch, l.line)
	case '>':
		tok = newToken(token.GT, l.ch, l.line)
	case '*':
		tok = newToken(token.ASTERISK, l.ch, l.line)
	case '/':
		tok = newToken(token.SLASH, l.ch, l.line)
	case '-':
		if l.isPartOfHeaderName() {
			return l.readHeaderName()
		}
		tok = newToken(token.MINUS, l.ch, l.line)
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = token.Token{Type: token.AND, Literal: literal, Line: l.line}
		} else {
			tok = newToken(token.AND, l.ch, l.line)
		}
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = token.Token{Type: token.OR, Literal: literal, Line: l.line}
		} else {
			tok = newToken(token.ILLEGAL, l.ch, l.line)
		}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = token.Token{Type: token.NOT_EQ, Literal: literal, Line: l.line}
		} else {
			tok = newToken(token.BANG, l.ch, l.line)
		}
	case ':':
		if l.peekChar() == ':' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = token.Token{Type: token.DOUBLE_COLON, Literal: literal, Line: l.line}
		} else {
			tok = newToken(token.COLON, l.ch, l.line)
		}
	case 'H':
		peekedWord := l.peekWord()
		if tokenType, isHTTPKeyword := HttpKeywords[peekedWord]; isHTTPKeyword {
			identifier, line := l.readIdentifier()
			return token.Token{Type: tokenType, Literal: identifier, Line: line}
		}
		fallthrough
	case 'L':
		peekedWord := l.peekWord()
		if tokenType, isLBKeyword := LbKeywords[peekedWord]; isLBKeyword {
			l.readIdentifier()
			return token.Token{Type: tokenType, Literal: peekedWord, Line: l.line}
		}
		fallthrough
	case 'S':
		peekedWord := l.peekWord()
		if tokenType, isSSLKeyword := SSLKeywords[peekedWord]; isSSLKeyword {
			l.readIdentifier()
			return token.Token{Type: tokenType, Literal: peekedWord, Line: l.line}
		}

		identifier, line := l.readIdentifier()
		return token.Token{Type: token.IDENT, Literal: identifier, Line: line}
	case 0:
		if l.braceDepth > 0 {
			if config.DebugMode {
				fmt.Printf("Unexpected EOF: unclosed brace, depth: %d", l.braceDepth)
			}
		}
		tok.Type = token.EOF
		tok.Literal = ""
		if config.DebugMode {
			fmt.Printf("DEBUG: Lexer reached EOF at position %d\n", l.position)
		}
	default:
		// check for number
		if IsDigit(l.ch) || (l.ch == '-' && IsDigit(l.peekChar())) {
			return l.readNumberOrIpAddress()
		}

		// check for identifier
		if IsLetter(l.ch) {
			tok.Literal, tok.Line = l.readIdentifier()
			switch tok.Literal {
			case "IP::client_addr":
				tok.Type = token.IP_CLIENT_ADDR
			case "IP::server_addr":
				tok.Type = token.IP_SERVER_ADDR
			case "IP::remote_addr":
				tok.Type = token.IP_REMOTE_ADDR
			case "eq":
				tok.Type = token.EQ
				tok.Literal = "eq"
			case "ne":
				tok.Type = token.NOT_EQ
				tok.Literal = "ne"
			case "equals":
				tok.Type = token.EQ
				tok.Literal = "equals"
			case "starts_with":
				tok.Type = token.STARTS_WITH
			case "contains":
				tok.Type = token.CONTAINS
			case "foreach":
				tok.Type = token.FOREACH
			case "default":
				tok.Type = token.DEFAULT
				tok.Literal = "default"
			case "or":
				tok.Type = token.OR
			case "and":
				tok.Type = token.AND
			default:
				tok.Type = token.LookupIdent(tok.Literal)
			}
			return tok
		}

		// everything else is an illegal token
		l.reportError("NextToken: Illegal token found = '%c'", l.ch)
		tok = newToken(token.ILLEGAL, l.ch, l.line)
	}

	l.readChar()

	if config.DebugMode {
		fmt.Printf("DEBUG: Lexer produced token: %v. State AFTER readChar() - l.ch: %q, l.position: %d, l.readPosition: %d\n", tok, l.ch, l.position, l.readPosition)
	}

	return tok
}

func (l *Lexer) readIdentifier() (string, int) {
	position := l.position
	startLine := l.line
	for IsLetter(l.ch) || IsDigit(l.ch) || l.ch == '_' || l.ch == ':' || l.ch == '.' || l.ch == '-' {
		if l.ch == '\n' {
			l.line++
		}
		l.readChar()
	}
	return l.input[position:l.position], startLine
}

func IsLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || ch == ':' || ch == '.'
}

func IsDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

// skips over single-line and block comments.
func (l *Lexer) skipComment() {
	// handle single-line comments starting with # or //
	if l.ch == '#' || (l.ch == '/' && l.peekChar() == '/') {
		for l.ch != '\x00' && l.ch != '\n' {
			l.readChar()
		}
		if l.ch == '\n' {
			l.readChar() // move past the newline character
		}
		return
	}

	// handle block comments starting with /*
	if l.ch == '/' && l.peekChar() == '*' {
		l.readChar() // Move past the /
		l.readChar() // Move past the *

		// read until the end of the block comment (*/)
		for {
			if l.ch == '*' && l.peekChar() == '/' {
				l.readChar() // move past the *
				l.readChar() // move past the /
				break
			}
			// if end of input is reached without finding */, break to avoid infinite loop
			if l.ch == '\x00' {
				break
			}
			l.readChar()
		}
	}

	// skip any whitespace after the comment
	l.skipWhitespace()
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	} else {
		return l.input[l.readPosition]
	}
}

func (l *Lexer) readString() string {
	startingQuote := l.ch // capture the type of quote used to start the string
	position := l.position + 1
	for {
		l.readChar()

		// break if we encounter the same quote used to start the string or the end of input
		if l.ch == startingQuote || l.ch == 0 {
			break
		}

		// handle escape sequences
		if l.ch == '\\' && l.peekChar() == startingQuote {
			l.readChar() // skip the escaped quote
		}
	}

	return l.input[position:l.position]
}

func (l *Lexer) readVariable() string {
	position := l.position
	l.readChar() // consume $
	for IsLetter(l.ch) || IsDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) peekWord() string {
	peekPos := l.position

	if peekPos >= len(l.input) {
		return ""
	}

	startPeekPos := peekPos

	for peekPos < len(l.input) {
		ch := l.input[peekPos]
		if !(IsLetter(ch) || IsDigit(ch) || ch == ':' || ch == '_') {
			break
		}
		peekPos++
	}

	if startPeekPos == peekPos {
		return ""
	}

	word := l.input[startPeekPos:peekPos]
	return word
}

func (l *Lexer) readNumberOrIpAddress() token.Token {
	startPosition := l.position
	isNegative := l.ch == '-'
	if isNegative {
		l.readChar()
	}

	for IsDigit(l.ch) {
		l.readChar()
	}

	if l.ch == '.' {
		return l.readIpAddress(startPosition)
	}

	return token.Token{
		Type:    token.NUMBER,
		Literal: l.input[startPosition:l.position],
		Line:    l.line,
	}
}

func (l *Lexer) readIpAddress(startPosition int) token.Token {
	dotCount := 0
	for IsDigit(l.ch) || l.ch == '.' {
		if l.ch == '.' {
			dotCount++
			if dotCount > 3 {
				break
			}
		}
		l.readChar()
	}

	if dotCount == 3 {
		return token.Token{
			Type:    token.IP_ADDRESS,
			Literal: l.input[startPosition:l.position],
			Line:    l.line,
		}
	}

	// if it's not a valid IP address, treat it as a number
	return token.Token{
		Type:    token.NUMBER,
		Literal: l.input[startPosition:l.position],
		Line:    l.line,
	}
}

func (l *Lexer) isPartOfHeaderName() bool {
	// check if the previous token was an identifier or part of a header name
	return l.position > 0 && (IsLetter(l.input[l.position-1]) || l.input[l.position-1] == '-')
}

func (l *Lexer) readHeaderName() token.Token {
	position := l.position
	for l.position < len(l.input) && (IsLetter(l.ch) || IsDigit(l.ch) || l.ch == '-') {
		l.readChar()
	}
	return token.Token{Type: token.IDENT, Literal: l.input[position:l.position], Line: l.line}
}

func (l *Lexer) reportError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	formattedMsg := "   [Lexer] " + msg + fmt.Sprintf(", Line: %d", l.line)
	l.errors = append(l.errors, formattedMsg)
}

func (l *Lexer) Errors() []string {
	return l.errors
}

func (l *Lexer) CurrentLine() int {
	return l.line
}

func (l *Lexer) readRegexPattern() string {
	position := l.position + 1
	for {
		l.readChar()
		if l.ch == '}' && l.peekChar() != '}' {
			break
		}
		if l.ch == 0 {
			l.reportError("Unterminated regex pattern")
			return ""
		}
	}
	return l.input[position:l.position]
}

func (l *Lexer) EnterSwitchBlock() {
	l.inSwitchBlock = true
}

func (l *Lexer) ExitSwitchBlock() {
	l.inSwitchBlock = false
}
