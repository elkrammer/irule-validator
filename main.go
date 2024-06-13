package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/elkrammer/irule-validator/evaluator"
	"github.com/elkrammer/irule-validator/lexer"
	"github.com/elkrammer/irule-validator/parser"
	"github.com/elkrammer/irule-validator/repl"
)

func main() {
	if len(os.Args) < 2 {
		repl.Start(os.Stdin, os.Stdout)
	}

	filename := os.Args[1]

	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file :%v\n", err)
		os.Exit(1)
	}

	scanner := bufio.NewScanner(bufio.NewReader(bytes.NewReader(content)))
	var out io.Writer = os.Stdout

	for scanner.Scan() {
		line := scanner.Text()
		l := lexer.New(line)
		p := parser.New(l)

		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			printParserErrors(out, p.Errors())
			os.Exit(1)
		}

		evaluated := evaluator.Eval(program)
		if evaluated != nil {
			io.WriteString(out, evaluated.Inspect())
			io.WriteString(out, "\n")
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error scanning file: %v\n", err)
		os.Exit(1)
	}
}

func printParserErrors(out io.Writer, errors []string) {
	for _, msg := range errors {
		io.WriteString(out, strings.TrimSpace(msg))
		io.WriteString(out, "\n")
	}
}
