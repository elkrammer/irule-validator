package config

import (
	"fmt"
	"os"
	"runtime/debug"
	"time"

	"github.com/spf13/pflag"
)

// App Config
var DebugMode bool
var PrintErrors bool
var PrintVersion bool

// Setup program flags
func SetupFlags() {
	pflag.BoolVarP(&DebugMode, "debug", "d", false, "Debugging Mode")
	pflag.BoolVarP(&PrintErrors, "print-errors", "p", false, "Print Errors")
	pflag.BoolVarP(&PrintVersion, "version", "v", false, "Print App Version")

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

	if *&PrintVersion {
		version := printVersion()
		fmt.Println(version)
		os.Exit(0)
	}
}

func printVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "Version: unknown"
	}

	var revision, buildTime string
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.time":
			buildTime = setting.Value
		}
	}

	shortRevision := "unknown"
	if revision != "" {
		shortRevision = revision
		if len(shortRevision) > 8 {
			shortRevision = shortRevision[:8]
		}
	}

	shortBuildTime := "unknown"
	if buildTime != "" {
		if t, err := time.Parse(time.RFC3339, buildTime); err == nil {
			shortBuildTime = t.Format("2006-01-02")
		}
	}

	return fmt.Sprintf("irule-validator Revision: %s, Build Time: %s", shortRevision, shortBuildTime)
}
