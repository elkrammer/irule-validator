package evaluator

import (
	"github.com/elkrammer/irule-validator/ast"
	"github.com/elkrammer/irule-validator/object"
)

func Eval(node ast.Node) object.Object {
	switch node := node.(type) {

	// statements
	case *ast.Program:
		return evalStatements(node.Statements)
	case *ast.ExpressionStatement:
		return Eval(node.Expression)

		// Expressions
	case *ast.NumberLiteral:
		return &object.Number{Value: node.Value}

	}

	return nil
}

func evalStatements(stmts []ast.Statement) object.Object {
	var result object.Object

	for _, statement := range stmts {
		result = Eval(statement)
	}

	return result
}
