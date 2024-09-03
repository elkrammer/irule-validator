package lexer

import (
	"fmt"

	"github.com/elkrammer/irule-validator/config"
	"github.com/elkrammer/irule-validator/token"
)

type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
	braceDepth   int
}

var HttpKeywords = map[string]token.TokenType{
	"HTTP_REQUEST":   token.HTTP_REQUEST,
	"HTTP::uri":      token.HTTP_URI,
	"HTTP::host":     token.HTTP_HOST,
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
	"SSL::cipher":      token.SSL_CIPHER,
	"SSL::cipher_bits": token.SSL_CIPHER_BITS,
	"SSL::clienthello": token.SSL_CLIENTHELLO,
	"SSL::serverhello": token.SSL_SERVERHELLO,
}

func New(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	if config.DebugMode {
		fmt.Printf("DEBUG: Lexer initialized with input length: %d\n", len(input))
	}
	return l
}

// read one forward character
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
		if config.DebugMode {
			fmt.Printf("DEBUG: Reached EOF in lexer at position %d\n", l.position)
		}
	} else {
		l.ch = l.input[l.readPosition]
		// if config.DebugMode {
		// 	fmt.Printf("DEBUG: Read char '%c' at position %d\n", l.ch, l.readPosition)
		// }
	}
	l.position = l.readPosition
	l.readPosition += 1
}

func newToken(tokenType token.TokenType, ch byte) token.Token {
	return token.Token{Type: tokenType, Literal: string(ch)}
}

func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	l.skipWhitespace()

	// skip single line comments
	if l.ch == '#' || (l.ch == '/' && l.peekChar() == '/') {
		l.skipComment()
		return l.NextToken()
	}

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = token.Token{Type: token.EQ, Literal: literal}
			if config.DebugMode {
				fmt.Printf("DEBUG: Lexer produced EQ token in case '=': %v\n", tok)
			}

		} else {
			tok = newToken(token.ASSIGN, l.ch)
		}
	case '{':
		tok = newToken(token.LBRACE, l.ch)
		l.braceDepth++
		if config.DebugMode {
			fmt.Printf("DEBUG: Lexer identified opening brace '{', depth now %d\n", l.braceDepth)
		}
	case '}':
		tok = newToken(token.RBRACE, l.ch)
		l.braceDepth--
		if config.DebugMode {
			fmt.Printf("DEBUG: Lexer identified closing brace '}', depth now %d\n", l.braceDepth)
		}
	case '(':
		tok = newToken(token.LPAREN, l.ch)
	case ')':
		tok = newToken(token.RPAREN, l.ch)
	case '[':
		tok = newToken(token.LBRACKET, l.ch)
	case ']':
		tok = newToken(token.RBRACKET, l.ch)
	case ',':
		tok = newToken(token.COMMA, l.ch)
	case '$':
		tok.Type = token.IDENT
		tok.Literal = l.readVariable()
		return tok
	case '"', '\'':
		tok.Type = token.STRING
		tok.Literal = l.readString()
	case '+':
		tok = newToken(token.PLUS, l.ch)
	case ';':
		tok = newToken(token.SEMICOLON, l.ch)
	case '<':
		tok = newToken(token.LT, l.ch)
	case '>':
		tok = newToken(token.GT, l.ch)
	case '*':
		tok = newToken(token.ASTERISK, l.ch)
	case '/':
		tok = newToken(token.SLASH, l.ch)
	case '-':
		tok = newToken(token.MINUS, l.ch)
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = token.Token{Type: token.AND, Literal: literal}
		} else {
			tok = newToken(token.AND, l.ch)
		}
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = token.Token{Type: token.OR, Literal: literal}
		} else {
			tok = newToken(token.ILLEGAL, l.ch)
		}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = token.Token{Type: token.NOT_EQ, Literal: literal}
		} else {
			tok = newToken(token.BANG, l.ch)
		}
	case ':':
		if l.peekChar() == ':' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = token.Token{Type: token.DOUBLE_COLON, Literal: literal}
		} else {
			tok = newToken(token.COLON, l.ch)
		}
	case 'H':
		peekedWord := l.peekWord()
		if tokenType, isHTTPKeyword := HttpKeywords[peekedWord]; isHTTPKeyword {
			l.readIdentifier() // consume the word
			return token.Token{Type: tokenType, Literal: peekedWord}
		}

		identifier := l.readIdentifier()
		return token.Token{Type: token.IDENT, Literal: identifier}
	case 'L':
		peekedWord := l.peekWord()
		if tokenType, isLBKeyword := LbKeywords[peekedWord]; isLBKeyword {
			l.readIdentifier() // consume the word
			return token.Token{Type: tokenType, Literal: peekedWord}
		}

		identifier := l.readIdentifier()
		return token.Token{Type: token.IDENT, Literal: identifier}
	case 'S':
		peekedWord := l.peekWord()
		if tokenType, isSSLKeyword := SSLKeywords[peekedWord]; isSSLKeyword {
			l.readIdentifier() // consume the word
			return token.Token{Type: tokenType, Literal: peekedWord}
		}

		identifier := l.readIdentifier()
		return token.Token{Type: token.IDENT, Literal: identifier}
	case 0:
		if l.braceDepth > 0 {
			fmt.Printf("Unexpected EOF: unclosed brace, depth: %d\n", l.braceDepth)
		}
		tok.Type = token.EOF
		tok.Literal = ""
		if config.DebugMode {
			fmt.Printf("DEBUG: Lexer reached EOF at position %d\n", l.position)
		}
	default:
		// Check for number
		if isDigit(l.ch) || (l.ch == '-' && isDigit(l.peekChar())) {
			return l.readNumberOrIpAddress()
		}

		// Check for identifier
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			switch tok.Literal {
			case "IP::client_addr":
				tok.Type = token.IP_CLIENT_ADDR
			case "eq":
				tok.Type = token.EQ
				tok.Literal = "eq"
			case "starts_with":
				tok.Type = token.STARTS_WITH
				tok.Literal = "starts_with"
			case "default":
				tok.Type = token.DEFAULT
				tok.Literal = "default"
			default:
				tok.Type = token.LookupIdent(tok.Literal)
			}
			// fmt.Printf("NextToken: Identified token = '%s'\n", tok.Literal)
			return tok
		}

		// Everything else is an illegal token
		fmt.Printf("NextToken: Illegal token found = '%c'\n", l.ch)
		tok = newToken(token.ILLEGAL, l.ch)
	}

	l.readChar()

	if config.DebugMode {
		fmt.Printf("DEBUG: Lexer produced token: Type=%s, Literal='%s', Position=%d\n", tok.Type, tok.Literal, l.position)
	}
	return tok
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' || l.ch == ':' || l.ch == '.' {
		l.readChar()
	}
	return l.input[position:l.position]
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || ch == ':' || ch == '.'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

// skips over single-line and block comments.
func (l *Lexer) skipComment() {
	// Handle single-line comments starting with # or //
	if l.ch == '#' || (l.ch == '/' && l.peekChar() == '/') {
		for l.ch != '\x00' && l.ch != '\n' {
			l.readChar()
		}
		if l.ch == '\n' {
			l.readChar() // move past the newline character
		}
		return
	}

	// Handle block comments starting with /*
	if l.ch == '/' && l.peekChar() == '*' {
		l.readChar() // Move past the /
		l.readChar() // Move past the *

		// Read until the end of the block comment (*/)
		for {
			if l.ch == '*' && l.peekChar() == '/' {
				l.readChar() // Move past the *
				l.readChar() // Move past the /
				break
			}
			// If end of input is reached without finding */, break to avoid infinite loop
			if l.ch == '\x00' {
				break
			}
			l.readChar()
		}
	}

	// Skip any whitespace after the comment
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
	startingQuote := l.ch // Capture the type of quote used to start the string
	position := l.position + 1
	for {
		l.readChar()

		// Break if we encounter the same quote used to start the string or the end of input
		if l.ch == startingQuote || l.ch == 0 {
			break
		}

		// Handle escape sequences
		if l.ch == '\\' && l.peekChar() == startingQuote {
			l.readChar() // Skip the escaped quote
		}
	}

	return l.input[position:l.position]
}

func (l *Lexer) readVariable() string {
	position := l.position
	l.readChar() // consume $
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) peekWord() string {
	position := l.position
	for isLetter(l.ch) || l.ch == ':' || l.ch == '_' {
		l.readChar()
	}
	word := l.input[position:l.position]
	l.position = position
	l.readPosition = position + 1
	l.ch = l.input[position]
	// fmt.Printf("DEBUG: peekWord result: %s\n", word)
	return word
}

func (l *Lexer) readNumberOrIpAddress() token.Token {
	startPosition := l.position
	isNegative := l.ch == '-'
	if isNegative {
		l.readChar()
	}

	for isDigit(l.ch) {
		l.readChar()
	}

	if l.ch == '.' {
		return l.readIpAddress(startPosition)
	}

	return token.Token{
		Type:    token.NUMBER,
		Literal: l.input[startPosition:l.position],
	}
}

func (l *Lexer) readIpAddress(startPosition int) token.Token {
	dotCount := 0
	for isDigit(l.ch) || l.ch == '.' {
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
		}
	}

	// If it's not a valid IP address, treat it as a number
	return token.Token{
		Type:    token.NUMBER,
		Literal: l.input[startPosition:l.position],
	}
}
