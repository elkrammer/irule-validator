package config

import (
	"fmt"
	"github.com/spf13/pflag"
	"os"
)

// App Config
var DebugMode bool
var PrintErrors bool

// Setup program flags
func SetupFlags() {
	pflag.BoolVarP(&DebugMode, "debug", "d", false, "Debugging Mode")
	pflag.BoolVarP(&PrintErrors, "print-errors", "p", false, "Print Errors")

	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		pflag.PrintDefaults() // prints the default flag options

		fmt.Fprintf(os.Stderr, `
If no parameter is specified it will run in quiet mode returning only
the result.
If a file name is specified, it will parse the provided file.
If no file name is specified, it will go into REPL mode.

Examples:
./irule-validator http.irule      # Parse http.irule and show only the result
./irule-validator -p http.irule   # Parse http.irule and print errors
./irule-validator                 # Start REPL
`)
	}

	// Manually check for the help flag before parsing
	help := pflag.BoolP("help", "h", false, "Show help message")

	pflag.Parse()

	if *help {
		pflag.Usage()
		os.Exit(0)
	}
}
