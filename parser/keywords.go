package parser

var (
	reservedKeywords = map[string]bool{
		"when": true, "if": true, "else": true, "elseif": true, "foreach": true, "for": true,
		"switch": true, "case": true, "default": true, "return": true, "set": true,
		"unset": true, "puts": true, "log": true, "while": true, "break": true,
		"continue": true, "exit": true, "abort": true,
	}
	commonHeaders = []string{
		"Accept", "Accept-Charset", "Accept-Encoding", "Accept-Language", "Authorization",
		"Cache-Control", "Connection", "Cookie", "Content-Length", "Content-MD5", "Content-Type",
		"Date", "Expect", "From", "Host", "If-Match", "If-Modified-Since", "If-None-Match",
		"If-Range", "If-Unmodified-Since", "Max-Forwards", "Pragma", "Proxy-Authorization",
		"Range", "Referer", "TE", "Upgrade", "User-Agent", "Via", "Warning", "X-Requested-With",
		"X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto", "X-Csrf-Token",
		"Server", "X-Powered-By", "names", "Location", "user_agent", "exists",
	}
	commonIdentifiers = []string{
		"log", "puts", "exit", "reject", "insert", "remove", "set", "unset",
		"if", "else", "elseif", "switch", "case", "default", "foreach", "for", "while",
		"break", "continue", "return", "proc", "catch", "eval",
		"local0", "local1", "local2", "local3", "local4", "local5", "local6", "local7",
		"content_type", "uri_path", "value", "pool", "path", "domain", "expires",
		"content", "node", "virtual",
	}
)
