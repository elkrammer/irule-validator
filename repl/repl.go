package repl

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/peterh/liner"

	"github.com/elkrammer/irule-validator/lexer"
	"github.com/elkrammer/irule-validator/token"
)

const PROMPT = ">> "

func Start(in io.Reader, out io.Writer) {
	history := filepath.Join(os.TempDir(), ".irule-validator")
	l := liner.NewLiner()
	defer l.Close()

	l.SetCtrlCAborts(true)

	if f, err := os.Open(history); err == nil {
		l.ReadHistory(f)
		f.Close()
	}

	for {
		if line, err := l.Prompt(PROMPT); err == nil {
			if line == "exit" {
				if f, err := os.Create(history); err == nil {
					l.WriteHistory(f)
					f.Close()
				}
				os.Exit(0)
				break
			}

			l.AppendHistory(line)
			line := lexer.New(line)

			for tok := line.NextToken(); tok.Type != token.EOF; tok = line.NextToken() {
				fmt.Fprintf(out, "%v\n", tok)
			}
		} else if err == liner.ErrPromptAborted {
			fmt.Fprintln(out, "Aborted")
			os.Exit(0)
			break
		}
	}
}

func printParserErrors(out io.Writer, errors []string) {
	for _, msg := range errors {
		io.WriteString(out, "\t"+msg+"\n")
	}
}
