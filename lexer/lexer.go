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

	// fmt.Printf("[Lexer] NextToken: Current char = '%c'\n", l.ch)

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
		if peekedWord == "HTTP_REQUEST" {
			l.readIdentifier() // consume the word
			return token.Token{Type: token.HTTP_REQUEST, Literal: "HTTP_REQUEST"}
		}
		if peekedWord == "HTTP::uri" {
			l.readIdentifier() // consume the word
			return token.Token{Type: token.HTTP_URI, Literal: "HTTP::uri"}
		}
		if peekedWord == "HTTP::host" {
			l.readIdentifier() // consume the word
			return token.Token{Type: token.HTTP_HOST, Literal: "HTTP::host"}
		}
		if peekedWord == "HTTP::redirect" {
			l.readIdentifier() // consume the word
			return token.Token{Type: token.HTTP_REDIRECT, Literal: "HTTP::redirect"}
		}
		if peekedWord == "HTTP::header" {
			l.readIdentifier() // consume the word
			return token.Token{Type: token.HTTP_HEADER, Literal: "HTTP::header"}
		}
		if peekedWord == "HTTP::respond" {
			l.readIdentifier() // consume the word
			return token.Token{Type: token.HTTP_RESPOND, Literal: "HTTP::respond"}
		}
		if peekedWord == "HTTP::method" {
			l.readIdentifier() // consume the word
			return token.Token{Type: token.HTTP_METHOD, Literal: "HTTP::method"}
		}
		if peekedWord == "HTTP::path" {
			l.readIdentifier() // consume the word
			return token.Token{Type: token.HTTP_PATH, Literal: "HTTP::path"}
		}
		if peekedWord == "HTTP::query" {
			l.readIdentifier() // consume the word
			return token.Token{Type: token.HTTP_QUERY, Literal: "HTTP::query"}
		}
		if peekedWord == "HTTP::redirect" {
			l.readIdentifier() // consume the word
			return token.Token{Type: token.HTTP_REDIRECT, Literal: "HTTP::redirect"}
		}

		identifier := l.readIdentifier()
		// fmt.Printf("DEBUG: Read identifier: %s\n", identifier)
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
			tok.Type = token.NUMBER
			tok.Literal = l.readNumber()
			// fmt.Printf("NextToken: Identified number = '%s'\n", tok.Literal)
			return tok
		}

		// Check for identifier
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			switch tok.Literal {
			case "IP::client_addr":
				tok.Type = token.IP_CLIENT_ADDR
			case "eq":
				tok.Type = token.EQ
				tok.Literal = "=="
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

func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
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
