package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/elkrammer/irule-validator/config"
	"github.com/elkrammer/irule-validator/lexer"
	"github.com/elkrammer/irule-validator/parser"
	"github.com/elkrammer/irule-validator/repl"
	"github.com/spf13/pflag"
)

func main() {
	config.SetupFlags()
	args := pflag.Args()

	if len(args) == 0 {
		repl.Start(os.Stdin, os.Stdout)
		return
	}

	filename := args[0]

	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file :%v\n", err)
		os.Exit(1)
	}

	if config.DebugMode {
		fmt.Printf("DEBUG: Input content:\n%s\n", string(content))
	}

	l := lexer.New(string(content))
	p := parser.New(l)

	p.ParseProgram()
	if len(p.Errors()) != 0 {
		if config.PrintErrors {
			printParserErrors(os.Stdout, p.Errors())
		}
		fmt.Printf("❌ Errors parsing irule %v\n", filename)
		os.Exit(1)
	}

	// You can add further processing of the parsed program here if needed
	fmt.Printf("✅ Successfully parsed irule %v\n", filename)
}

func printParserErrors(out io.Writer, errors []string) {
	for _, msg := range errors {
		io.WriteString(out, strings.TrimSpace(msg))
		io.WriteString(out, "\n")
	}
}
