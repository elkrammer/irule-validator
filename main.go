package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/elkrammer/irule-validator/config"
	"github.com/elkrammer/irule-validator/lexer"
	"github.com/elkrammer/irule-validator/parser"
	"github.com/elkrammer/irule-validator/repl"
)

func main() {
	debug := flag.Bool("debug", false, "Run in debug mode")
	flag.Parse()
	config.DebugMode = *debug

	args := flag.Args()
	if len(args) == 0 {
		repl.Start(os.Stdin, os.Stdout)
		return
	}

	filename := os.Args[0]

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

		p.ParseProgram()
		if len(p.Errors()) != 0 {
			printParserErrors(out, p.Errors())
			os.Exit(1)
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
