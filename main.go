package main

import (
	"fmt"
	"github.com/elkrammer/irule-validator/repl"
	"os"
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

	isValid := validateF5iRule(string(content))

	if isValid {
		fmt.Println("The F5 iRule is valid.")
	} else {
		fmt.Println("The F5 iRule is not valid.")
	}
}

func validateF5iRule(rule string) bool {
	return true
}
