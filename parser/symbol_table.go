package parser

type SymbolType int

const (
	NODE SymbolType = iota
	POOL
)

type SymbolTable struct {
	scopes []map[SymbolType]SymbolInfo
}

type SymbolInfo struct {
	declared bool
	// line     int
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		scopes: []map[SymbolType]SymbolInfo{make(map[SymbolType]SymbolInfo)},
	}
}

func (st *SymbolTable) EnterScope() {
	st.scopes = append(st.scopes, make(map[SymbolType]SymbolInfo))
}

func (st *SymbolTable) ExitScope() {
	if len(st.scopes) > 1 {
		st.scopes = st.scopes[:len(st.scopes)-1]
	}
}

func (st *SymbolTable) Declare(p *Parser, symType SymbolType) {
	currentScope := st.scopes[len(st.scopes)-1]

	if symType == NODE && currentScope[POOL].declared {
		p.reportError("Invalid combination: 'node' and 'pool' in the same block.")
		return
	}
	if symType == POOL && currentScope[NODE].declared {
		p.reportError("Invalid combination: 'pool' and 'node' in the same block.")
		return
	}

	currentScope[symType] = SymbolInfo{declared: true}
}
