package lexer

import (
	// "strings"

	"github.com/elkrammer/irule-validator/token"
)

type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
}

func New(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

// read one forward character
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition += 1
}

func (l *Lexer) rewind() {
	if l.readPosition > 0 {
		l.position--
		l.readPosition--
		l.ch = l.input[l.readPosition-1]
	}
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
		} else {
			tok = newToken(token.ASSIGN, l.ch)
		}
	case '{':
		tok = newToken(token.LBRACE, l.ch)
	case '}':
		tok = newToken(token.RBRACE, l.ch)
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
	case 0:
		tok.Type = token.EOF
		tok.Literal = ""
	default:
		// Check for number
		if isDigit(l.ch) || (l.ch == '-' && isDigit(l.peekChar())) {
			tok.Type = token.NUMBER
			tok.Literal = l.readNumber()
			return tok
		}

		// Check for identifier
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			switch tok.Literal {
			case "IP::client_addr":
				tok.Type = token.IP_CLIENT_ADDR
			case "HTTP::host":
				tok.Type = token.HTTP_HOST
			case "HTTP::redirect":
				tok.Type = token.HTTP_REDIRECT
			case "HTTP_REQUEST":
				tok.Type = token.HTTP_REQUEST
			case "HTTP::uri":
				tok.Type = token.HTTP_URI
			case "eq":
				tok.Type = token.EQ
				tok.Literal = "=="
			default:
				tok.Type = token.LookupIdent(tok.Literal)
			}
			return tok
		}

		// Everything else is an illegal token
		tok = newToken(token.ILLEGAL, l.ch)
	}

	l.readChar()
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

// func (l *Lexer) readVariable() string {
// 	var str strings.Builder
// 	str.WriteRune('$') // Include the '$' character in the string builder
//
// 	for {
// 		l.readChar()
//
// 		if isLetter(l.ch) || isDigit(l.ch) {
// 			str.WriteByte(l.ch) // Append the character to the string builder
// 		} else {
// 			// l.rewind()
// 			l.readPosition-- // Move the position back to the non-letter/digit character
// 			break
// 		}
//
// 		// Check for the end of input
// 		if l.readPosition >= len(l.input) {
// 			break
// 		}
// 	}
// 	return str.String()
// }
