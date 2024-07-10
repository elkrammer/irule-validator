package evaluator

import (
	"fmt"
	"strings"

	"github.com/elkrammer/irule-validator/ast"
	"github.com/elkrammer/irule-validator/config"
	"github.com/elkrammer/irule-validator/object"
)

var (
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
	NULL  = &object.Null{}
)

func Eval(node ast.Node, env *object.Environment) object.Object {
	if config.DebugMode {
		fmt.Printf("DEBUG: Eval - Node type: %T\n", node)
	}
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
		val := Eval(node.Value, env)
		// Unwrap single-element arrays resulting from expr evaluations
		if arr, ok := val.(*object.Array); ok && len(arr.Elements) == 1 {
			val = arr.Elements[0]
		}
		env.Set(strings.TrimPrefix(node.Name.Value, "$"), val)
		return val
	case *ast.Identifier:
		return evalIdentifier(node, env)
	case *ast.ListLiteral:
		elements := evalExpressions(node.Elements, env)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}
		return &object.Array{Elements: elements}
	case *ast.ExprExpression:
		return Eval(node.Expression, env)
	case *ast.FunctionLiteral:
		params := node.Parameters
		body := node.Body
		function := &object.Function{Parameters: params, Env: env, Body: body}
		if node.Name != nil {
			env.Set(node.Name.Value, function)
		}
		return function
	case *ast.CallExpression:
		if config.DebugMode {
			fmt.Printf("DEBUG: Evaluating CallExpression: %v\n", node)
			fmt.Printf("DEBUG: CallExpression - Function: %T, Arguments: %d\n", node.Function, len(node.Arguments))
		}

		function := Eval(node.Function, env)
		if isError(function) {
			return function
		}

		args := evalExpressions(node.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}

		return applyFunction(function, args)
	case *ast.StringLiteral:
		return &object.String{Value: node.Value}

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
	case left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ:
		return evalStringInfixExpression(operator, left, right)
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

	if config.DebugMode {
		fmt.Printf("DEBUG: Evaluating block statement: %v\n", block)
	}

	for _, statement := range block.Statements {
		result = Eval(statement, env)

		if config.DebugMode {
			fmt.Printf("DEBUG: Evaluating statement in block: %T\n", statement)
			fmt.Printf("DEBUG: Statement result: %v\n", result)
		}

		if result != nil {
			rt := result.Type()
			if rt == object.RETURN_VALUE_OBJ || rt == object.ERROR_OBJ {
				return result
			}
		}
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: Block statement result: %v\n", result)
	}
	return result
}

func evalProgram(program *ast.Program, env *object.Environment) object.Object {
	var result object.Object

	if config.DebugMode {
		fmt.Printf("DEBUG: Starting to eval program\n")
	}
	for _, statement := range program.Statements {
		if config.DebugMode {
			fmt.Printf("Evaluating statement: %T\n", statement)
		}
		result = Eval(statement, env)
		if config.DebugMode {
			fmt.Printf("DEBUG: Statement result: %v\n", result)
		}

		switch result := result.(type) {
		case *object.ReturnValue:
			return result.Value
		case *object.Error:
			return result
		}
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: Finished evaluating program\n")
		fmt.Printf("Final result: %+v\n", result)
	}
	return result
}

func evalIdentifier(node *ast.Identifier, env *object.Environment) object.Object {
	if config.DebugMode {
		fmt.Printf("DEBUG: Evaluating identifier: %s\n", node.Value)
	}
	if node.IsVariable {
		// Remove the leading $ for lookup
		val, ok := env.Get(strings.TrimPrefix(node.Value, "$"))
		if !ok {
			if config.DebugMode {
				fmt.Printf("DEBUG: Variable not found: %s\n", node.Value)
			}
			return newError("identifier not found: %s", node.Value)
		}
		if config.DebugMode {
			fmt.Printf("DEBUG: Identifier value: %v\n", val)
		}
		return val
	}

	if builtin, ok := builtins[node.Value]; ok {
		return builtin
	}

	val, ok := env.Get(node.Value)
	if !ok {
		if config.DebugMode {
			fmt.Printf("DEBUG: Function not found: %s\n", node.Value)
		}
		return newError("identifier not found: %s", node.Value)
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: Function found: %s = %v\n", node.Value, val)
	}
	return val
}

func evalListLiteral(node *ast.ListLiteral, env *object.Environment) object.Object {
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
	if config.DebugMode {
		fmt.Printf("DEBUG: Evaluating expr command with args: %v\n", args)
	}

	result := Eval(args[0], env)
	if config.DebugMode {
		fmt.Printf("DEBUG: expr command result: %v\n", result)
	}
	if number, ok := result.(*object.Number); ok {
		return number
	}

	return newError("expr command expects a number expression, got=%T", result)
}

func applyFunction(fn object.Object, args []object.Object) object.Object {
	if config.DebugMode {
		fmt.Printf("DEBUG: Applying function: %T with args: %+v\n", fn, args)
	}

	switch fn := fn.(type) {
	case *object.Function:
		extendedEnv := extendFunctionEnv(fn, args)
		evaluated := Eval(fn.Body, extendedEnv)
		if config.DebugMode {
			fmt.Printf("DEBUG: Function body: %v\n", fn.Body)
			fmt.Printf("DEBUG: Extended environment: %v\n", extendedEnv)
			fmt.Printf("DEBUG: Function body evaluated to: %v\n", evaluated)
		}
		return unwrapReturnValue(evaluated)

	case *object.Builtin:
		return fn.Fn(args...)

	default:
		return newError("not a function: %s", fn.Type())
	}
}

func extendFunctionEnv(
	fn *object.Function,
	args []object.Object,
) *object.Environment {
	env := object.NewEnclosedEnvironment(fn.Env)

	for paramIdx, param := range fn.Parameters {
		env.Set(param.Value, args[paramIdx])
	}

	return env
}

func unwrapReturnValue(obj object.Object) object.Object {
	if returnValue, ok := obj.(*object.ReturnValue); ok {
		return returnValue.Value
	}

	return obj
}

func evalStringInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	if operator != "+" {
		return newError("unknown operator: %s %s %s",
			left.Type(), operator, right.Type())
	}

	leftVal := left.(*object.String).Value
	rightVal := right.(*object.String).Value
	return &object.String{Value: leftVal + rightVal}
}
