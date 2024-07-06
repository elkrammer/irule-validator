package evaluator

import (
	"fmt"

	"github.com/elkrammer/irule-validator/object"
)

var builtins = map[string]*object.Builtin{
	"puts": &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			for _, arg := range args {
				fmt.Println(arg.Inspect())
			}

			return NULL
		},
	},
}
