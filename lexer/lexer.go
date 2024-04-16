package lexer

import (
	"strings"

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

	// Was this a simple token-type?
	// if val, ok := l.lookup[l.ch]; ok {
	// 	// Skip the character itself and return the found value
	// 	l.readChar()
	// 	return val
	// }

	switch l.ch {
	case ']':
		tok.Type = token.ILLEGAL
		tok.Literal = "Closing ']' without opening one"
	case '}':
		tok.Type = token.ILLEGAL
		tok.Literal = "Closing '}' without opening one"
	case '$':
		val := l.readVariable()
		tok.Type = token.VARIABLE
		tok.Literal = val
	case '"':
		tok.Type = token.STRING
		tok.Literal = l.readString()
	case '+':
		// tok = newToken(token.PLUS, l.ch)
		tok = newToken(token.IDENT, l.ch)
	case 0:
		tok.Type = token.EOF
		tok.Literal = ""
	// case '[':
	// 	str, err := l.readEval()
	// 	if err == nil {
	// 		tok.Type = token.EVAL
	// 		tok.Literal = "[" + str + "]"
	// 	} else {
	// 		tok.Type = token.ILLEGAL
	// 		tok.Literal = err.Error()
	// 	}
	// case '{':
	// 	str, err := l.readBlock()
	// 	if err == nil {
	// 		tok.Type = token.BLOCK
	// 		tok.Literal = str
	// 	} else {
	// 		tok.Type = token.ILLEGAL
	// 		tok.Literal = err.Error()
	// 	}
	default:
		// Check for number
		if (l.ch == '-' && isDigit(l.peekChar())) || isDigit(l.ch) {
			tok.Type = token.NUMBER
			tok.Literal = l.readNumber()
			return tok
		}
		// Check for identifier
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = token.LookupIdent(tok.Literal)
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
	for isLetter(l.ch) || l.ch == ':' {
		l.readChar()
	}
	return l.input[position:l.position]
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

// skip comment (until the end of the line).
func (l *Lexer) skipComment() {
	// Read until the end of the line or the end of the input
	for l.ch != '\x00' && l.ch != '\n' {
		l.readChar()
	}

	// if it's a newline
	if l.ch == '\n' {
		return
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
	position := l.position + 1
	for {
		l.readChar()
		if l.ch == '"' || l.ch == 0 {
			break
		}
	}
	return l.input[position:l.position]
}

func (l *Lexer) readVariable() string {
	var str strings.Builder
	str.WriteRune('$') // Include the '$' character in the string builder
	// str := string(l.ch)

	for {
		l.readChar()

		// // Check for the end of input
		// if l.readPosition >= len(l.input) {
		// 	// return str
		// 	break
		// }
		//
		// if l.ch == '$' || isLetter(l.ch) {
		if isLetter(l.ch) || isDigit(l.ch) {
			// str += string(l.ch)
			str.WriteByte(l.ch) // Append the character to the string builder
		} else {
			l.rewind()
			break
			// return str
		}

		// Check for the end of input
		if l.readPosition >= len(l.input) {
			break
		}
	}
	return str.String()
}
