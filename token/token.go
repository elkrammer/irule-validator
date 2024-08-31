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
	ASSIGN       = "="
	PLUS         = "+"
	MINUS        = "-"
	BANG         = "!"
	ASTERISK     = "*"
	SLASH        = "/"
	LT           = "<"
	GT           = ">"
	EQ           = "=="
	NOT_EQ       = "!="
	DOLLAR       = "$"
	COLON        = ":"
	DOUBLE_COLON = "::"

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
	ELSEIF = "ELSEIF"
	RETURN = "RETURN"
	TRUE   = "TRUE"
	FALSE  = "FALSE"
	ARRAY  = "ARRAY"
	SET    = "SET"

	// HTTP TOKENS
	HTTP_REQUEST  = "HTTP_REQUEST"
	HTTP_RESPONSE = "HTTP_RESPONSE"
	HTTP_URI      = "HTTP::uri"
	HTTP_METHOD   = "HTTP::method"
	HTTP_HOST     = "HTTP::host"
	HTTP_PATH     = "HTTP::path"
	HTTP_QUERY    = "HTTP::query"
	HTTP_HEADER   = "HTTP::header"
	HTTP_REDIRECT = "HTTP::redirect"
	HTTP_RESPOND  = "HTTP::respond"
	HTTP_COLLECT  = "HTTP::collect"
	HTTP_RELEASE  = "HTTP::release"
	HTTP_PAYLOAD  = "HTTP::payload"
	HTTP_COOKIE   = "HTTP::cookie"
	HTTP_VERSION  = "HTTP::version"
	HTTP_STATUS   = "HTTP::status"
	HTTP_USERNAME = "HTTP::username"
	HTTP_PASSWORD = "HTTP::password"
	HTTP_PROXY    = "HTTP::proxy"
	HTTP_CLASS    = "HTTP::class"
	HTTP_COMPRESS = "HTTP::compress"
	HTTP_FILTER   = "HTTP::filter"

	SSL_CIPHER      = "SSL::cipher"
	SSL_CIPHER_BITS = "SSL::cipher_bits"

	IP_ADDRESS     = "IP_ADDRESS"
	IP_CLIENT_ADDR = "IP::client_addr"
	IP_SERVER_ADDR = "IP::server_addr"

	// LOAD BALANCING
	// Events
	LB_SELECTED  = "LB_SELECTED"
	LB_FAILED    = "LB_FAILED"
	LB_QUEUED    = "LB_QUEUED"
	LB_COMPLETED = "LB_COMPLETED"
	LB_METHOD    = "LB::method"

	// Commands
	LB_MODE     = "LB::mode"
	LB_SELECT   = "LB::select"
	LB_RESELECT = "LB::reselect"
	LB_DETACH   = "LB::detach"

	// Server-related
	LB_SERVER      = "LB::server"
	LB_SERVER_ADDR = "LB::server addr"
	LB_SERVER_PORT = "LB::server port"

	// Pool-related
	LB_POOL         = "LB::pool"
	LB_POOL_NAME    = "LB::pool name"
	LB_POOL_MEMBER  = "LB::pool member"
	LB_POOL_MEMBERS = "LB::pool members"

	// Status and health
	LB_STATUS = "LB::status"
	LB_ALIVE  = "LB::alive"

	// Session persistence
	LB_PERSIST = "LB::persist"

	// Priority and scoring
	LB_SCORE    = "LB::score"
	LB_PRIORITY = "LB::priority"

	// Connection-related
	LB_CONNECT = "LB::connect"

	// Miscellaneous
	LB_BIAS  = "LB::bias"
	LB_SNAT  = "LB::snat"
	LB_LIMIT = "LB::limit"
	LB_CLASS = "LB::class"

	SESSION_DATA    = "SESSION::data"
	SESSION_PERSIST = "SESSION::persist"

	// F5 Event Contexts KEYWORDS - when <EVENT_CONTEXT> {}
	CLIENTSSL_HANDSHAKE = "CLIENTSSL_HANDSHAKE"
	SERVERSSL_HANDSHAKE = "SERVERSSL_HANDSHAKE"
	TCP_REQUEST         = "TCP_REQUEST"
	CLIENT_ACCEPTED     = "CLIENT_ACCEPTED"
	SERVER_CONNECTED    = "SERVER_CONNECTED"

	// iRule-specific keywords
	STARTS_WITH = "starts_with"
	ENDS_WITH   = "ends_with"
	WHEN        = "when"
	THEN        = "then"
	CONTAINS    = "contains"
	MATCH       = "match"
	MATCHES     = "matches"

	// Additional control structures
	SWITCH  = "switch"
	CASE    = "case"
	DEFAULT = "default"

	// Additional operators
	AND = "&&"
	OR  = "||"

	// iRule-specific commands
	LOG  = "log"
	POOL = "pool"
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
	"set":         SET,
	"contains":    CONTAINS,
	"match":       MATCH,
	"elseif":      ELSEIF,
	"switch":      SWITCH,
	"case":        CASE,
	"default":     DEFAULT,
	"array":       ARRAY,
	"matches":     MATCHES,
	"ends_with":   ENDS_WITH,
	"equals":      EQ,
	"eq":          EQ,

	// F5 Event Contexts
	"HTTP_REQUEST":        HTTP_REQUEST,
	"HTTP_RESPOND":        HTTP_RESPOND,
	"CLIENTSSL_HANDSHAKE": CLIENTSSL_HANDSHAKE,
	"SERVERSSL_HANDSHAKE": SERVERSSL_HANDSHAKE,
	"LB_SELECTED":         LB_SELECTED,
	"LB_FAILED":           LB_FAILED,
	"TCP_REQUEST":         TCP_REQUEST,
	"IP_CLIENT_ADDR":      IP_CLIENT_ADDR,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
