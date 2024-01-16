package token

type TokenType string

type Token struct {
	Type    TokenType
	Literal string
}

const (
	// SPECIAL TOKENS
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"
	IDENT   = "IDENT"

	// Identifiers / Literals
	HTTP_URI    = "HTTP::uri"
	HTTP_METHOD = "HTTP::method"
	HTTP_HOST   = "HTTP::host"
	HTTP_PATH   = "HTTP::path"
	HTTP_QUERY  = "HTTP::query"
	HTTP_HEADER = "HTTP::header"

	SSL_CIPHER      = "SSL::cipher"
	SSL_CIPHER_BITS = "SSL::cipher_bits"

	IP_CLIENT_ADDR = "IP::client_addr"
	IP_SERVER_ADDR = "IP::server_addr"

	LB_SERVER = "LB::server"
	LB_METHOD = "LB::method"

	SESSION_DATA    = "SESSION::data"
	SESSION_PERSIST = "SESSION::persist"

	// F5 Event Contexts
	// when <EVENT_CONTEXT> {}
	HTTP_REQUEST        = "HTTP_REQUEST"
	HTTP_RESPONSE       = "HTTP_RESPONSE"
	CLIENTSSL_HANDSHAKE = "CLIENTSSL_HANDSHAKE"
	SERVERSSL_HANDSHAKE = "SERVERSSL_HANDSHAKE"
	LB_SELECTED         = "LB_SELECTED"
	LB_FAILED           = "LB_FAILED"
	TCP_REQUEST         = "TCP_REQUEST"

	// F5 COMMANDS
	STARTS_WITH = "starts_with"

	//operators
	ASSIGN   = "="
	PLUS     = "+"
	MINUS    = "-"
	BANG     = "!"
	ASTERISK = "*"
	SLASH    = "/"
	LT       = "<"
	GT       = ">"
	EQ       = "=="
	NOT_EQ   = "!="

	// delimiters
	COMMA     = ","
	SEMICOLON = ";"
	LPAREN    = "("
	RPAREN    = ")"
	LBRACE    = "{"
	RBRACE    = "}"
	LBRACKET  = "["
	RBRACKET  = "]"

	// keywords
	IF     = "IF"
	ELSE   = "ELSE"
	RETURN = "RETURN"
	WHEN   = "WHEN"
)

var keywords = map[string]TokenType{
	"if":     IF,
	"else":   ELSE,
	"return": RETURN,
	"WHEN":   WHEN,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}