# 📏 iRule-Validator

![Static Badge](https://img.shields.io/badge/build-passing-elk)
![GitHub Release](https://img.shields.io/github/v/release/elkrammer/irule-validator)
![Static Badge](https://img.shields.io/badge/license-MIT-blue?)

Ever tried writing an F5 iRule and thought, "will this work?" only to have F5
respond with, "Nah, invalid expression on line 42?" 😩

Wouldn't it be nice to catch those errors in your iRules **before** they break production?
Well, you're welcome! 🎁

Inspired by the awesome book [Writing an Interpreter in Go](https://interpreterbook.com),
this project aims to parse F5 iRules with style and grace! 🦸 (Well, at least
most of the time.)

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

Pro Tip: When using this in a CI/CD pipeline, be sure to call it with `-p` to
get those sweet error printouts you so desperately crave.

## 🦄 Disclaimer

Does it validate every possible command with perfect accuracy? Not quite.
Full syntax validation is *hard*, and I've realized that F5 iRules are
a bottomless pit of edge cases. 🕳️

Building a complete F5 iRule parser is like trying to solve a puzzle where
the pieces keep changing shape. But hey, I already have a parser that
covers most of the use cases I need and that's good enough for me! 🎉

## 🧑 Contributing

PRs are welcome! 💥
