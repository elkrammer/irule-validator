# 📏 iRule-Validator

![Static Badge](https://img.shields.io/badge/build-passing-elk)
![GitHub Release](https://img.shields.io/github/v/release/elkrammer/irule-validator)
![Static Badge](https://img.shields.io/badge/license-MIT-blue?)

Ever written an F5 iRule and thought "this should work!"—only to get hit with
an "Invalid expression on line 42"?

Deploying an iRule, waiting for automation to kick in, and then realizing it’s
broken is frustrating. Wouldn't it be great to catch those errors before they
waste your time?

You're in luck! This project lets you parse F5 iRules, catch syntax errors early,
and debug with confidence.

![irule-validator](https://github.com/user-attachments/assets/6fdf255e-aa6e-4d73-972e-18ad3e700502)

## 🚀 Usage

```bash
Usage of ./irule-validator:
  -d, --debug          Debugging Mode
  -h, --help           Show help message
  -p, --print-errors   Print Errors

If no parameter is specified it will run in quiet mode returning only
the result.
If a file name is specified, it will parse the provided file.
If no file name is specified, it will go into REPL mode.

Examples:
./irule-validator http.irule      # Parse http.irule and show only the result
./irule-validator -p http.irule   # Parse http.irule and print errors
./irule-validator                 # Start REPL
```

When using this in a CI/CD pipeline, be sure to call it with `-p` to get
those sweet error printouts you so desperately crave. 🤤

## 🛠️ Features

- Parse and validate various iRule-specific constructs
- Static Syntax Analysis
  - Glob and regex pattern validation
  - Symbol table to prevent incompatible command combinations
- Detailed error reporting with line numbers
- Debug mode for detailed parsing information

## 🦄 Disclaimer

Does it validate every possible command with perfect accuracy? Not quite.
Full syntax validation is *hard*, and F5 iRules are a bottomless pit
of edge cases. 🕳️

Building a complete F5 iRule parser is like trying to solve a puzzle where
the pieces keep changing shape. But hey, I already have a parser that
covers most of the use cases I need and that's good enough for me! 🎉

## 🧑 Contributing

PRs are welcome! 💥
