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
	BLOCK    = "BLOCK"
	EVAL     = "EVAL"
	IDENT    = "IDENT"
	ILLEGAL  = "ILLEGAL"
	NUMBER   = "NUMBER"
	STRING   = "STRING"
	VARIABLE = "VARIABLE"

	// // SPECIAL TOKENS
	// ILLEGAL  = "ILLEGAL"
	// EOF      = "EOF"
	// IDENT    = "IDENT"
	// STRING   = "STRING"
	// VARIABLE = "VARIABLE"
	//
	// //operators
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
	//
	// // delimiters
	// COMMA        = ","
	// SEMICOLON    = ";"
	LPAREN   = "("
	RPAREN   = ")"
	LBRACE   = "{"
	RBRACE   = "}"
	LBRACKET = "["
	RBRACKET = "]"
	// DOUBLE_COLON = "::"
	//
	// // KEYWORDS
	// // --------
	IF     = "IF"
	ELSE   = "ELSE"
	RETURN = "RETURN"
	TRUE   = "TRUE"
	FALSE  = "FALSE"

	// HTTP_URI      = "HTTP::uri"
	// HTTP_METHOD   = "HTTP::method"
	// HTTP_HOST     = "HTTP::host"
	// HTTP_PATH     = "HTTP::path"
	// HTTP_QUERY    = "HTTP::query"
	// HTTP_HEADER   = "HTTP::header"
	// HTTP_REDIRECT = "HTTP::redirect"
	//
	// SSL_CIPHER      = "SSL::cipher"
	// SSL_CIPHER_BITS = "SSL::cipher_bits"
	//
	// IP_CLIENT_ADDR = "IP::client_addr"
	// IP_SERVER_ADDR = "IP::server_addr"
	//
	// LB_SERVER = "LB::server"
	// LB_METHOD = "LB::method"
	//
	// SESSION_DATA    = "SESSION::data"
	// SESSION_PERSIST = "SESSION::persist"
	//
	// // F5 Event Contexts KEYWORDS
	// // when <EVENT_CONTEXT> {}
	// HTTP_REQUEST        = "HTTP_REQUEST"
	// HTTP_RESPONSE       = "HTTP_RESPONSE"
	// CLIENTSSL_HANDSHAKE = "CLIENTSSL_HANDSHAKE"
	// SERVERSSL_HANDSHAKE = "SERVERSSL_HANDSHAKE"
	// LB_SELECTED         = "LB_SELECTED"
	// LB_FAILED           = "LB_FAILED"
	// TCP_REQUEST         = "TCP_REQUEST"
	//
	// // F5 COMMANDS
	// STARTS_WITH = "starts_with"
	// WHEN        = "WHEN"
	// THEN        = "THEN"
)

var keywords = map[string]TokenType{
	"if":     IF,
	"else":   ELSE,
	"return": RETURN,
	"true":   TRUE,
	"false":  FALSE,
	// "HTTP_REQUEST":   HTTP_REQUEST,
	// "HTTP_RESPONSE":  HTTP_RESPONSE,
	// "HTTP::uri":      HTTP_URI,
	// "HTTP::method":   HTTP_METHOD,
	// "HTTP::host":     HTTP_HOST,
	// "HTTP::path":     HTTP_PATH,
	// "HTTP::query":    HTTP_QUERY,
	// "HTTP::header":   HTTP_HEADER,
	// "HTTP::redirect": HTTP_REDIRECT,
	//
	// "SSL::cipher":      SSL_CIPHER,
	// "SSL::cipher_bits": SSL_CIPHER_BITS,
	//
	// "IP::client_addr": IP_CLIENT_ADDR,
	// "IP::server_addr": IP_SERVER_ADDR,
	//
	// "LB::server": LB_SERVER,
	// "LB::method": LB_METHOD,
	//
	// "SESSION::data":    SESSION_DATA,
	// "SESSION::persist": SESSION_PERSIST,
	//
	// // F5 Event Contexts KEYWORDS
	// // when <EVENT_CONTEXT> {}
	// "CLIENTSSL_HANDSHAKE": CLIENTSSL_HANDSHAKE,
	// "SERVERSSL_HANDSHAKE": SERVERSSL_HANDSHAKE,
	// "LB_SELECTED":         LB_SELECTED,
	// "LB_FAILED":           LB_FAILED,
	// "TCP_REQUEST":         TCP_REQUEST,
	//
	// // F5 COMMANDS
	// "starts_with": STARTS_WITH,
	// "WHEN":        WHEN,
	// "then":        THEN,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
