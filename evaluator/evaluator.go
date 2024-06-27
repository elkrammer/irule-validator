package evaluator

import (
	"fmt"

	"github.com/elkrammer/irule-validator/ast"
	"github.com/elkrammer/irule-validator/object"
)

var (
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
	NULL  = &object.Null{}
)

func Eval(node ast.Node, env *object.Environment) object.Object {
	fmt.Printf("Evaluating node: %T\n", node) // Debug print
	switch node := node.(type) {

	// statements
	case *ast.Program:
		return evalProgram(node, env)
	case *ast.ExpressionStatement:
		return Eval(node.Expression, env)

		// Expressions
	case *ast.NumberLiteral:
		return &object.Number{Value: node.Value}
	case *ast.Boolean:
		return nativeBoolToBooleanObject(node.Value)
	case *ast.PrefixExpression:
		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}
		return evalPrefixExpression(node.Operator, right)
	case *ast.InfixExpression:
		left := Eval(node.Left, env)
		if isError(left) {
			return left
		}

		right := Eval(node.Right, env)
		if isError(right) {
			return right
		}

		return evalInfixExpression(node.Operator, left, right)
	case *ast.BlockStatement:
		return evalBlockStatement(node, env)
	case *ast.IfExpression:
		return evalIfExpression(node, env)
	case *ast.ReturnStatement:
		val := Eval(node.ReturnValue, env)
		if isError(val) {
			return val
		}
		return &object.ReturnValue{Value: val}
	case *ast.SetStatement:
		// if node == nil {
		// 	fmt.Printf("SetStatement is nil\n") // Debug print
		// 	return nil
		// }
		// fmt.Printf("Evaluating SetStatement: %s\n", node.String()) // Add this line
		val := Eval(node.Value, env)
		if isError(val) {
			return val
		}
		env.Set(node.Name.Value, val)
		fmt.Printf("Set %s to %v\n", node.Name.Value, val) // Add this line
		return val                                         // Return the value that was set
	case *ast.Identifier:
		return evalIdentifier(node, env)
	case *ast.ArrayLiteral:
		elements := evalExpressions(node.Elements, env)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}
		return &object.Array{Elements: elements}
	case *ast.ExprExpression:
		return Eval(node.Expression, env)

	}

	return nil
}

func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}

func isError(obj object.Object) bool {
	if obj != nil {
		return obj.Type() == object.ERROR_OBJ
	}
	return false
}

func evalStatements(stmts []ast.Statement, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range stmts {
		result = Eval(statement, env)

		if returnValue, ok := result.(*object.ReturnValue); ok {
			return returnValue.Value
		}
	}

	return result
}

func nativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return TRUE
	}
	return FALSE
}

func evalPrefixExpression(operator string, right object.Object) object.Object {
	switch operator {
	case "!":
		return evalBangOperatorExpression(right)
	case "-":
		if boolean, ok := right.(*object.Boolean); ok {
			return newError("invalid command name '-%s'", boolean.Inspect())
		}
		return evalMinusPrefixOperatorExpression(right)
	default:
		return newError("unknown operator: %s%s", operator, right.Type())
	}
}

func evalBangOperatorExpression(right object.Object) object.Object {
	switch obj := right.(type) {
	case *object.Boolean:
		return nativeBoolToBooleanObject(!obj.Value)
	case *object.Null:
		return TRUE
	case *object.Number:
		if obj.Value == 0 {
			return TRUE
		}
		return FALSE
	default:
		return FALSE
	}
}

func evalMinusPrefixOperatorExpression(right object.Object) object.Object {
	if right.Type() != object.NUMBER_OBJ {
		return newError("unknown operator: -%s", right.Type())
	}

	value := right.(*object.Number).Value
	return &object.Number{Value: -value}
}

func evalInfixExpression(operator string, left, right object.Object) object.Object {
	switch {
	case left.Type() == object.NUMBER_OBJ && right.Type() == object.NUMBER_OBJ:
		return evalNumberInfixExpression(operator, left, right)
	case operator == "!=":
		return nativeBoolToBooleanObject(left != right)
	case left.Type() == object.BOOLEAN_OBJ && right.Type() == object.BOOLEAN_OBJ:
		return nativeBoolToBooleanObject(left != right)
	case left.Type() != right.Type():
		return newError("type mismatch: %s %s %s", left.Type(), operator, right.Type())
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalNumberInfixExpression(operator string, left, right object.Object) object.Object {
	leftVal := left.(*object.Number).Value
	rightVal := right.(*object.Number).Value

	switch operator {
	case "+":
		return &object.Number{Value: leftVal + rightVal}
	case "-":
		return &object.Number{Value: leftVal - rightVal}
	case "*":
		return &object.Number{Value: leftVal * rightVal}
	case "/":
		return &object.Number{Value: leftVal / rightVal}
	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)
	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalIfExpression(ie *ast.IfExpression, env *object.Environment) object.Object {
	condition := Eval(ie.Condition, env)

	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return Eval(ie.Consequence, env)
	} else if ie.Alternative != nil {
		return Eval(ie.Alternative, env)
	} else {
		return NULL
	}
}

// isTruthy determines the truthiness of an object
func isTruthy(obj object.Object) bool {
	switch obj.Type() {
	case object.NULL_OBJ:
		return false
	case object.BOOLEAN_OBJ:
		return obj.(*object.Boolean).Value
	case object.NUMBER_OBJ:
		return obj.(*object.Number).Value != 0
	default:
		return true
	}
}

func evalBlockStatement(
	block *ast.BlockStatement,
	env *object.Environment,
) object.Object {
	var result object.Object

	for _, statement := range block.Statements {
		result = Eval(statement, env)

		if result != nil {
			rt := result.Type()
			if rt == object.RETURN_VALUE_OBJ || rt == object.ERROR_OBJ {
				return result
			}
		}
	}

	return result
}

func evalProgram(program *ast.Program, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range program.Statements {
		fmt.Printf("Evaluating statement: %T\n", statement) // Add this line
		result = Eval(statement, env)

		switch result := result.(type) {
		case *object.ReturnValue:
			return result.Value
		case *object.Error:
			return result
		}
	}

	fmt.Printf("Final result: %+v\n", result) // Add this line
	return result
}

func evalIdentifier(node *ast.Identifier, env *object.Environment) object.Object {
	if node.IsVariable {
		if val, ok := env.Get(node.Value); ok {
			return val
		}
		return newError("identifier not found: $" + node.Value)
	} else {
		if val, ok := env.Get(node.Value); ok {
			return val
		}
		return newError("identifier not found: " + node.Value)
	}
}

// func evalArrayLiteral(node *ast.ArrayLiteral, env *object.Environment) object.Object {
// 	elements := evalExpressions(node.Elements, env)
// 	if len(elements) == 1 && isError(elements[0]) {
// 		return elements[0]
// 	}
//
// 	// If there's only one element and it's a function call to 'expr'
// 	if len(elements) == 1 {
// 		if callExp, ok := node.Elements[0].(*ast.CallExpression); ok {
// 			if ident, ok := callExp.Function.(*ast.Identifier); ok && ident.Value == "expr" {
// 				return evalExpressionCommand(callExp.Arguments, env)
// 			}
// 		}
// 	}
//
// 	return &object.Array{Elements: elements}
// }

// func evalArrayLiteral(node *ast.ArrayLiteral, env *object.Environment) object.Object {
// 	elements := evalExpressions(node.Elements, env)
// 	if len(elements) == 1 && isError(elements[0]) {
// 		return elements[0]
// 	}
//
// 	// If there's only one element and it's an ExprExpression
// 	if len(elements) == 1 {
// 		if exprObj, ok := node.Elements[0].(*ast.ExprExpression); ok {
// 			return Eval(exprObj.Expression, env)
// 		}
// 	}
//
// 	return &object.Array{Elements: elements}
// }

func evalArrayLiteral(node *ast.ArrayLiteral, env *object.Environment) object.Object {
	elements := evalExpressions(node.Elements, env)
	if len(elements) == 1 && isError(elements[0]) {
		return elements[0]
	}

	// If there's only one element and it's from an ExprExpression, return it directly
	if len(elements) == 1 {
		if _, ok := node.Elements[0].(*ast.ExprExpression); ok {
			return elements[0]
		}
	}

	return &object.Array{Elements: elements}
}

func evalExpressions(
	exps []ast.Expression,
	env *object.Environment,
) []object.Object {
	var result []object.Object

	for _, e := range exps {
		evaluated := Eval(e, env)
		if isError(evaluated) {
			return []object.Object{evaluated}
		}
		result = append(result, evaluated)
	}

	return result
}

func evalExpressionCommand(args []ast.Expression, env *object.Environment) object.Object {
	if len(args) != 1 {
		return newError("wrong number of arguments for expr. got=%d, want=1", len(args))
	}

	result := Eval(args[0], env)
	if number, ok := result.(*object.Number); ok {
		return number
	}

	return newError("expr command expects a number expression, got=%T", result)
}
