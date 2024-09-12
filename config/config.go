package config

import "github.com/spf13/pflag"

// App Config
var DebugMode bool
var PrintErrors bool

// Setup program flags
func SetupFlags() {
	// Define flags
	pflag.BoolVarP(&DebugMode, "debug", "d", false, "Run in debug mode")
	pflag.BoolVarP(&PrintErrors, "print-errors", "p", false, "Print Errors")

	pflag.Parse()
}
