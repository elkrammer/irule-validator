package object

import (
	"fmt"
)

type ObjectType string

const (
	NUMBER_OBJ  = "NUMBER"
	BOOLEAN_OBJ = "BOOLEAN"
	NULL_OBJ    = "NULL"
)

type Object interface {
	Type() ObjectType
	Inspect() string
}

type Number struct {
	Value float64
}

func (i *Number) Type() ObjectType { return NUMBER_OBJ }
func (i *Number) Inspect() string  { return fmt.Sprintf("%v", i.Value) }

type Boolean struct {
	Value bool
}

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string  { return fmt.Sprintf("%v", b.Value) }

type Null struct{}

func (n *Null) Type() ObjectType { return NULL_OBJ }
func (n *Null) Inspect() string  { return "" }
