package ast

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/elkrammer/irule-validator/token"
)

func precedence(op string) int {
	precedences := map[string]int{
		token.EQ:          1,
		token.NOT_EQ:      1,
		token.LT:          2,
		token.GT:          2,
		token.PLUS:        3,
		token.MINUS:       3,
		token.SLASH:       4,
		token.ASTERISK:    4,
		token.STARTS_WITH: 5,
	}
	if p, ok := precedences[op]; ok {
		return p
	}
	return 0
}

// interface that all AST nodes must implement
type Node interface {
	TokenLiteral() string
	String() string
}

// interface for statement nodes
type Statement interface {
	Node
	statementNode()
}

// interface for expression nodes
type Expression interface {
	Node
	expressionNode()
}

// represents the entire program
type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	} else {
		return ""
	}
}

func (p *Program) String() string {
	var out bytes.Buffer

	for _, s := range p.Statements {
		out.WriteString(s.String())
	}
	return out.String()
}

// represents an identifier expression
type Identifier struct {
	Token      token.Token
	Value      string
	IsVariable bool
	IsKeyword  bool
}

func (i *Identifier) expressionNode() {}
func (i *Identifier) String() string  { return i.Value }
func (i *Identifier) TokenLiteral() string {
	return i.Value
}

type InvalidIdentifier struct {
	Token token.Token
	Value string
}

func (ii *InvalidIdentifier) expressionNode()      {}
func (ii *InvalidIdentifier) TokenLiteral() string { return ii.Token.Literal }
func (ii *InvalidIdentifier) String() string       { return ii.Value }

type SetStatement struct {
	Token token.Token
	Name  Expression
	Value Expression
}

func (ls *SetStatement) statementNode()       {}
func (ls *SetStatement) expressionNode()      {}
func (ls *SetStatement) TokenLiteral() string { return ls.Token.Literal }
func (ls *SetStatement) String() string {
	var out bytes.Buffer

	out.WriteString(ls.TokenLiteral() + " ")
	out.WriteString(ls.Name.String())

	if ls.Value != nil {
		out.WriteString(" ")
		out.WriteString(ls.Value.String())
	}

	return out.String()
}

// RETURN Statement
type ReturnStatement struct {
	Token       token.Token
	ReturnValue Expression
}

func (rs *ReturnStatement) statementNode()       {}
func (rs *ReturnStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *ReturnStatement) String() string {
	var out bytes.Buffer

	out.WriteString(rs.TokenLiteral() + " ")

	if rs.ReturnValue != nil {
		out.WriteString(rs.ReturnValue.String())
	}

	return out.String()
}

// EXPRESSION
type ExpressionStatement struct {
	Token      token.Token
	Expression Expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

// Numbers
type NumberLiteral struct {
	Token token.Token
	Value int64
}

func (il *NumberLiteral) expressionNode()      {}
func (nl *NumberLiteral) TokenLiteral() string { return fmt.Sprintf("%d", nl.Value) }
func (il *NumberLiteral) String() string       { return il.Token.Literal }

// PREFIXES
type PrefixExpression struct {
	Token    token.Token
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PrefixExpression) String() string {
	var out bytes.Buffer

	out.WriteString(pe.Operator)
	out.WriteString(pe.Right.String())

	return out.String()
}

// INFIX EXPRESSIONS
type InfixExpression struct {
	Token    token.Token
	Left     Expression
	Operator string
	Right    Expression
}

func (ie *InfixExpression) expressionNode()      {}
func (ie *InfixExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *InfixExpression) String() string {
	var out bytes.Buffer

	if ie.Left != nil {
		out.WriteString(ie.Left.String())
	}

	out.WriteString(" " + ie.Operator + " ")

	if ie.Right != nil {
		switch right := ie.Right.(type) {
		case *ParenthesizedExpression:
			out.WriteString(right.String())
		case *InfixExpression:
			if precedence(right.Operator) < precedence(ie.Operator) {
				out.WriteString("(")
				out.WriteString(right.String())
				out.WriteString(")")
			} else {
				out.WriteString(right.String())
			}
		default:
			out.WriteString(ie.Right.String())
		}
	}

	return out.String()
}

// BOOLEAN LTERALS
type Boolean struct {
	Token token.Token
	Value bool
}

func (b *Boolean) expressionNode()      {}
func (b *Boolean) TokenLiteral() string { return b.Token.Literal }
func (b *Boolean) String() string       { return b.Token.Literal }

// STRING LITERAL
type StringLiteral struct {
	Token token.Token
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLiteral) String() string {
	return `"` + sl.Value + `"`
}

// BLOCKS
type BlockStatement struct {
	Token      token.Token // { token
	Statements []Statement
}

func (bs *BlockStatement) statementNode()       {}
func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BlockStatement) String() string {
	var out bytes.Buffer

	for _, s := range bs.Statements {
		out.WriteString(s.String())
	}

	return out.String()
}

// IF EXPRESSION
type IfExpression struct {
	Token       token.Token // `if` token
	Condition   Expression
	Consequence *BlockStatement
	Alternative *BlockStatement
}

func (ie *IfExpression) expressionNode()      {}
func (ie *IfExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IfExpression) String() string {
	var out bytes.Buffer

	out.WriteString("if")
	out.WriteString(ie.Condition.String())
	out.WriteString(" ")
	out.WriteString(ie.Consequence.String())

	if ie.Alternative != nil {
		out.WriteString("else ")
		out.WriteString(ie.Alternative.String())
	}

	return out.String()
}

type IfStatement struct {
	Token       token.Token
	Condition   Expression
	Consequence *BlockStatement
	Alternative *BlockStatement
}

func (is *IfStatement) statementNode()       {}
func (is *IfStatement) TokenLiteral() string { return is.Token.Literal }
func (is *IfStatement) String() string {
	var out bytes.Buffer

	out.WriteString("if ")
	out.WriteString(is.Condition.String())
	out.WriteString(" ")
	out.WriteString(is.Consequence.String())

	if is.Alternative != nil {
		out.WriteString(" else ")
		out.WriteString(is.Alternative.String())
	}

	return out.String()
}

// HASH LITERALS
type HashLiteral struct {
	Token token.Token // '{' token
	Pairs map[StringLiteral]Expression
}

func (hl *HashLiteral) expressionNode()      {}
func (hl *HashLiteral) TokenLiteral() string { return hl.Token.Literal }
func (hl *HashLiteral) String() string {
	var out bytes.Buffer

	pairs := []string{}
	for key, value := range hl.Pairs {
		pairs = append(pairs, key.String()+":"+value.String())
	}

	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")

	return out.String()
}

type IndexExpression struct {
	Token token.Token // [ token
	Left  Expression
	Index Expression
}

func (ie *IndexExpression) expressionNode()      {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IndexExpression) String() string {
	var out bytes.Buffer

	out.WriteString(ie.Left.String())
	out.WriteString("[")
	out.WriteString(ie.Index.String())
	out.WriteString("]")

	return out.String()
}

type ListLiteral struct {
	Token    token.Token // '{' token
	Elements []Expression
}

func (al *ListLiteral) expressionNode()      {}
func (al *ListLiteral) TokenLiteral() string { return al.Token.Literal }
func (al *ListLiteral) String() string {
	var out bytes.Buffer

	elements := []string{}
	for _, el := range al.Elements {
		elements = append(elements, el.String())
	}

	out.WriteString("{")
	out.WriteString(strings.Join(elements, " "))
	out.WriteString("}")

	return out.String()
}

type CallExpression struct {
	Token     token.Token // '(' token
	Function  Expression  // identifier or FunctionLiteral
	Arguments []Expression
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }
func (ce *CallExpression) String() string {
	var out bytes.Buffer

	args := []string{}
	for _, a := range ce.Arguments {
		args = append(args, a.String())
	}

	out.WriteString(ce.Function.String())
	out.WriteString("(")
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")

	return out.String()
}

type ParenthesizedExpression struct {
	Expression Expression
}

func (pe *ParenthesizedExpression) expressionNode()      {}
func (pe *ParenthesizedExpression) TokenLiteral() string { return "(" }
func (pe *ParenthesizedExpression) String() string {
	return "(" + pe.Expression.String() + ")"
}

// represents a TCL list (equivalent to an array)
type ArrayLiteral struct {
	Token    token.Token // '[' token
	Elements []Expression
}

func (al *ArrayLiteral) expressionNode()      {}
func (al *ArrayLiteral) TokenLiteral() string { return al.Token.Literal }
func (al *ArrayLiteral) String() string {
	var out bytes.Buffer

	elements := []string{}
	for _, el := range al.Elements {
		elements = append(elements, el.String())
	}

	out.WriteString("[")
	out.WriteString(strings.Join(elements, " ")) // TCL uses space as separator
	out.WriteString("]")

	return out.String()
}

// CommandSubstitution represents a command substitution in TCL, enclosed in square brackets
type CommandSubstitution struct {
	Token   token.Token // '[' token
	Command Expression
}

func (cs *CommandSubstitution) expressionNode()      {}
func (cs *CommandSubstitution) TokenLiteral() string { return cs.Token.Literal }
func (cs *CommandSubstitution) String() string {
	var out bytes.Buffer
	out.WriteString("[")
	out.WriteString(cs.Command.String())
	out.WriteString("]")
	return out.String()
}

// WHEN EXPRESSION
type WhenExpression struct {
	Token token.Token // when token
	Event Expression  // identifier like HTTP_REQUEST
	Block *BlockStatement
}

func (we *WhenExpression) expressionNode()      {}
func (we *WhenExpression) TokenLiteral() string { return we.Token.Literal }
func (we *WhenExpression) String() string {
	var out bytes.Buffer
	out.WriteString("when ")
	out.WriteString(we.Event.String())
	out.WriteString(" ")
	out.WriteString(we.Block.String())
	return out.String()
}

// HTTP URI EXPRESSION
type HttpUriExpression struct {
	Token  token.Token // HTTP_URI token
	Method *Identifier
}

func (hue *HttpUriExpression) expressionNode()      {}
func (hue *HttpUriExpression) TokenLiteral() string { return hue.Token.Literal }
func (hue *HttpUriExpression) String() string {
	var out bytes.Buffer
	out.WriteString("HTTP::uri")
	if hue.Method != nil {
		out.WriteString(" ")
		out.WriteString(hue.Method.String())
	}
	return out.String()
}

type IRuleNode struct {
	When       *WhenNode
	Statements []Statement
}

type WhenNode struct {
	Event      string
	Statements []Statement
}

type HttpExpression struct {
	Token    token.Token // http token
	Command  *Identifier // HTTP command (e.g., HTTP::uri)
	Method   *Identifier // optional method (e.g., path, host)
	Argument Expression
}

func (he *HttpExpression) expressionNode()      {}
func (he *HttpExpression) TokenLiteral() string { return he.Token.Literal }
func (he *HttpExpression) String() string {
	var out bytes.Buffer
	out.WriteString("[")
	out.WriteString(he.Command.String())
	if he.Method != nil {
		out.WriteString(" ")
		out.WriteString(he.Method.String())
	}
	out.WriteString("]")
	return out.String()
}

type BracketExpression struct {
	Token      token.Token
	Expression Expression
}

func (be *BracketExpression) expressionNode()      {}
func (be *BracketExpression) TokenLiteral() string { return be.Token.Literal }
func (be *BracketExpression) String() string {
	var out bytes.Buffer
	out.WriteString("[")
	out.WriteString(be.Expression.String())
	out.WriteString("]")
	return out.String()
}

type SwitchStatement struct {
	Token   token.Token // switch token
	Options []string
	Value   Expression
	Cases   []*CaseStatement
	Default *CaseStatement
	IsRegex bool
	IsGlob  bool
}

func (ss *SwitchStatement) expressionNode()      {}
func (ls *SwitchStatement) statementNode()       {}
func (ss *SwitchStatement) TokenLiteral() string { return ss.Token.Literal }
func (ss *SwitchStatement) String() string {
	var out bytes.Buffer
	out.WriteString("switch ")
	out.WriteString(ss.Value.String())
	out.WriteString(" {\n")
	for _, c := range ss.Cases {
		out.WriteString(c.String())
	}
	if ss.Default != nil {
		out.WriteString("default ")
		out.WriteString(ss.Default.String())
	}
	out.WriteString("}\n")

	return out.String()
}

type CaseStatement struct {
	Token       token.Token // case token
	Value       Expression
	Consequence *BlockStatement
	Line        int
}

func (cs *CaseStatement) expressionNode()      {}
func (cs *CaseStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *CaseStatement) String() string {
	var out bytes.Buffer
	out.WriteString(cs.Value.String())
	out.WriteString(" ")
	out.WriteString(cs.Consequence.String())
	out.WriteString("\n")
	return out.String()
}

type IpExpression struct {
	Token    token.Token // token associated with this expression
	Function string      // specific IP function (e.g., "client_addr" or "server_addr")
}

func (ie *IpExpression) expressionNode()      {}
func (ie *IpExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IpExpression) String() string       { return "IP::" + ie.Function }

type IpAddressLiteral struct {
	Token token.Token
	Value string
}

func (ip *IpAddressLiteral) expressionNode()      {}
func (ip *IpAddressLiteral) TokenLiteral() string { return ip.Token.Literal }
func (ip *IpAddressLiteral) String() string       { return ip.Value }

type LoadBalancerExpression struct {
	Token    token.Token // LB token
	Command  *Identifier // Load Balancer command (e.g., LB::select)
	Method   *Identifier // optional method or subcommand
	Argument Expression
}

func (lbe *LoadBalancerExpression) expressionNode()      {}
func (lbe *LoadBalancerExpression) TokenLiteral() string { return lbe.Token.Literal }
func (lbe *LoadBalancerExpression) String() string {
	var out bytes.Buffer
	out.WriteString("[")
	out.WriteString(lbe.Command.String())
	if lbe.Method != nil {
		out.WriteString(" ")
		out.WriteString(lbe.Method.String())
	}
	out.WriteString("]")
	return out.String()
}

type SSLExpression struct {
	Token    token.Token // SSL token
	Command  *Identifier // SSL command (e.g., SSL::cert)
	Method   *Identifier // optional method or subcommand
	Argument Expression
}

func (se *SSLExpression) expressionNode()      {}
func (se *SSLExpression) TokenLiteral() string { return se.Token.Literal }
func (se *SSLExpression) String() string {
	var out bytes.Buffer
	out.WriteString("SSL::")
	out.WriteString(se.Command.String())
	if se.Method != nil {
		out.WriteString(" ")
		out.WriteString(se.Method.String())
	}
	if se.Argument != nil {
		out.WriteString(" ")
		out.WriteString(se.Argument.String())
	}
	return out.String()
}

type StringOperation struct {
	Token     token.Token  // 'string' token
	Function  string       // string function (e.g., "tolower")
	Operation string       // operation (e.g., "tolower")
	Arguments []Expression // argument to the string operation
}

func (so *StringOperation) expressionNode()      {}
func (so *StringOperation) TokenLiteral() string { return so.Token.Literal }
func (so *StringOperation) String() string {
	var out bytes.Buffer
	out.WriteString(so.Function)
	out.WriteString(" ")
	out.WriteString(so.Operation)

	for _, arg := range so.Arguments {
		out.WriteString(" ")
		out.WriteString(arg.String())
	}

	return out.String()
}

// represents a map literal in the AST
type MapLiteral struct {
	Token token.Token // the token.LBRACE token
	Pairs map[Expression]Expression
}

func (ml *MapLiteral) expressionNode()      {}
func (ml *MapLiteral) TokenLiteral() string { return ml.Token.Literal }
func (ml *MapLiteral) String() string {
	var out bytes.Buffer

	pairs := []string{}
	for key, value := range ml.Pairs {
		pairs = append(pairs, key.String()+" "+value.String())
	}

	out.WriteString("{")
	out.WriteString(strings.Join(pairs, ", "))
	out.WriteString("}")

	return out.String()
}

type ClassCommand struct {
	Token      token.Token
	Subcommand string
	Options    []Expression
	Arguments  []Expression
}

func (cc *ClassCommand) expressionNode()      {}
func (cc *ClassCommand) TokenLiteral() string { return cc.Token.Literal }
func (cc *ClassCommand) String() string {
	var out bytes.Buffer
	out.WriteString("class ")
	out.WriteString(cc.Subcommand)
	for _, opt := range cc.Options {
		out.WriteString(" [")
		out.WriteString(opt.String())
		out.WriteString("]")
	}
	for _, arg := range cc.Arguments {
		out.WriteString(" ")
		out.WriteString(arg.String())
	}
	return out.String()
}

// represents a string that may contain embedded expressions
type InterpolatedString struct {
	Token token.Token //  token containing the string literal
	Parts []Expression
}

func (is *InterpolatedString) expressionNode()      {}
func (is *InterpolatedString) TokenLiteral() string { return is.Token.Literal }
func (is *InterpolatedString) String() string {
	var out bytes.Buffer

	for _, part := range is.Parts {
		out.WriteString(part.String())
	}

	return out.String()
}

type ForEachStatement struct {
	Token    token.Token // 'foreach' token
	Variable string
	List     Expression
	Body     *BlockStatement
}

func (fs *ForEachStatement) statementNode()       {}
func (fs *ForEachStatement) TokenLiteral() string { return fs.Token.Literal }
func (fs *ForEachStatement) String() string {
	var out bytes.Buffer

	out.WriteString("foreach ")
	out.WriteString(fs.Variable)
	out.WriteString(" in ")
	if fs.List != nil {
		out.WriteString(fs.List.String())
	} else {
		out.WriteString("<nil>")
	}
	out.WriteString(" ")
	if fs.Body != nil {
		out.WriteString(fs.Body.String())
	} else {
		out.WriteString("<nil>")
	}

	return out.String()
}

type NodeStatement struct {
	Token     token.Token
	IPAddress string
	Port      string
}

func (ns *NodeStatement) expressionNode()      {}
func (ns *NodeStatement) TokenLiteral() string { return ns.Token.Literal }
func (ns *NodeStatement) String() string {
	return fmt.Sprintf("node %s %s", ns.IPAddress, ns.Port)
}

type LtmRule struct {
	Token token.Token
	Name  *Identifier
	Body  *BlockStatement
}

func (lr *LtmRule) statementNode()       {}
func (lr *LtmRule) TokenLiteral() string { return lr.Token.Literal }
func (lr *LtmRule) String() string {
	var out bytes.Buffer
	out.WriteString("ltm rule ")
	out.WriteString(lr.Name.String())
	out.WriteString(" ")
	out.WriteString(lr.Body.String())
	return out.String()
}

type SlashExpression struct {
	Token token.Token
}

func (se *SlashExpression) expressionNode()      {}
func (se *SlashExpression) TokenLiteral() string { return se.Token.Literal }
func (se *SlashExpression) String() string       { return "/" }

type GlobPattern struct {
	Token token.Token
	Value string
}

func (gp *GlobPattern) expressionNode()      {}
func (gp *GlobPattern) TokenLiteral() string { return gp.Token.Literal }
func (gp *GlobPattern) String() string       { return "{" + gp.Value + "}" }

type MultiPattern struct {
	Patterns []Expression
}

func (mp *MultiPattern) expressionNode()      {}
func (mp *MultiPattern) TokenLiteral() string { return "MultiPattern" }
func (mp *MultiPattern) String() string {
	var out bytes.Buffer
	patterns := []string{}
	for _, p := range mp.Patterns {
		patterns = append(patterns, p.String())
	}
	out.WriteString(strings.Join(patterns, " - "))
	return out.String()
}

type RegexPattern struct {
	Token token.Token
	Value string
}

func (rp *RegexPattern) expressionNode()      {}
func (rp *RegexPattern) TokenLiteral() string { return rp.Token.Literal }
func (rp *RegexPattern) String() string       { return rp.Value }

type RegsubExpression struct {
	Token       token.Token
	Flags       []string
	Pattern     Expression
	InputString Expression
	Replacement Expression
	ResultVar   *Identifier
}

func (re *RegsubExpression) expressionNode()      {}
func (re *RegsubExpression) TokenLiteral() string { return re.Token.Literal }
func (re *RegsubExpression) String() string {
	var out bytes.Buffer
	out.WriteString("regsub")
	for _, flag := range re.Flags {
		out.WriteString(" " + flag)
	}
	out.WriteString(" " + re.Pattern.String())
	out.WriteString(" " + re.InputString.String())
	out.WriteString(" " + re.Replacement.String())
	out.WriteString(" " + re.ResultVar.String())
	return out.String()
}

type CommandInvocation struct {
	Token     token.Token
	Command   string
	Arguments []Expression
}

func (ci *CommandInvocation) expressionNode()      {}
func (ci *CommandInvocation) TokenLiteral() string { return ci.Token.Literal }
func (ci *CommandInvocation) String() string {
	var out bytes.Buffer
	out.WriteString("[")
	out.WriteString(ci.Command)
	for _, arg := range ci.Arguments {
		out.WriteString(" ")
		if arg != nil {
			out.WriteString(arg.String())
		} else {
			out.WriteString("<nil>")
		}
	}
	out.WriteString("]")
	return out.String()
}
