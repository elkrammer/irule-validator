package repl

import (
	"bufio"
	"fmt"
	"io"

	"github.com/elkrammer/irule-validator/lexer"
	"github.com/elkrammer/irule-validator/parser"
)

const PROMPT = ">> "

func Start(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)

	for {
		fmt.Fprint(out, PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}

		line := scanner.Text()
		l := lexer.New(line)
		p := parser.New(l)

		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			printParserErrors(out, p.Errors())
			continue
		}
		io.WriteString(out, program.String())
		io.WriteString(out, "\n")
	}
}

func printParserErrors(out io.Writer, errors []string) {
	io.WriteString(out, "Woops! We ran into some funky business here!\n")
	io.WriteString(out, "Parser Errors:\n")
	for _, msg := range errors {
		io.WriteString(out, "  "+msg+"\n")
	}
}
