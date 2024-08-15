package token

type TokenType string

type Token struct {
	Type    TokenType
	Literal string
}

// predefined token types
const (
	// Things
	SEMICOLON = ";"
	EOF       = "EOF"
	NEWLINE   = "\n"

	// types
	BLOCK   = "BLOCK"
	IDENT   = "IDENT"
	ILLEGAL = "ILLEGAL"
	NUMBER  = "NUMBER"
	STRING  = "STRING"

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
	DOLLAR   = "$"

	// delimiters
	COMMA    = ","
	LPAREN   = "("
	RPAREN   = ")"
	LBRACE   = "{"
	RBRACE   = "}"
	LBRACKET = "["
	RBRACKET = "]"

	// KEYWORDS
	IF     = "IF"
	ELSE   = "ELSE"
	RETURN = "RETURN"
	TRUE   = "TRUE"
	FALSE  = "FALSE"
	ARRAY  = "ARRAY"

	// F5 iRules SPECIFIC TOKENS
	HTTP_URI      = "HTTP::uri"
	HTTP_METHOD   = "HTTP::method"
	HTTP_HOST     = "HTTP::host"
	HTTP_PATH     = "HTTP::path"
	HTTP_QUERY    = "HTTP::query"
	HTTP_HEADER   = "HTTP::header"
	HTTP_REDIRECT = "HTTP::redirect"

	SSL_CIPHER      = "SSL::cipher"
	SSL_CIPHER_BITS = "SSL::cipher_bits"

	IP_CLIENT_ADDR = "IP::client_addr"
	IP_SERVER_ADDR = "IP::server_addr"

	LB_SERVER = "LB::server"
	LB_METHOD = "LB::method"

	SESSION_DATA    = "SESSION::data"
	SESSION_PERSIST = "SESSION::persist"

	// F5 Event Contexts KEYWORDS - when <EVENT_CONTEXT> {}
	HTTP_REQUEST        = "HTTP_REQUEST"
	HTTP_RESPONSE       = "HTTP_RESPONSE"
	CLIENTSSL_HANDSHAKE = "CLIENTSSL_HANDSHAKE"
	SERVERSSL_HANDSHAKE = "SERVERSSL_HANDSHAKE"
	LB_SELECTED         = "LB_SELECTED"
	LB_FAILED           = "LB_FAILED"
	TCP_REQUEST         = "TCP_REQUEST"

	// F5 COMMANDS
	STARTS_WITH = "starts_with"
	WHEN        = "when"
	THEN        = "then"
)

var keywords = map[string]TokenType{
	// F5 iRules Keywords
	"when":        WHEN,
	"then":        THEN,
	"if":          IF,
	"else":        ELSE,
	"return":      RETURN,
	"true":        TRUE,
	"false":       FALSE,
	"starts_with": STARTS_WITH,

	// F5 Event Contexts
	"HTTP_REQUEST":        HTTP_REQUEST,
	"HTTP_RESPONSE":       HTTP_RESPONSE,
	"CLIENTSSL_HANDSHAKE": CLIENTSSL_HANDSHAKE,
	"SERVERSSL_HANDSHAKE": SERVERSSL_HANDSHAKE,
	"LB_SELECTED":         LB_SELECTED,
	"LB_FAILED":           LB_FAILED,
	"TCP_REQUEST":         TCP_REQUEST,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
