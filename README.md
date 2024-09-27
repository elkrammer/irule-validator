# 📏 iRule-Validator

Ever tried writing an F5 iRule and thought, "will this work?" only to have F5
respond with, "Nah, invalid expression on line 42?" 😩

Wouldn't it be nice to catch those errors in your iRules **before** they break production?
Well, you're welcome! 🎁

Inspired by the awesome book [Writing an Interpreter in Go](https://interpreterbook.com),
this project aims to parse F5 iRules with style and grace! 🦸 (Well, at least
most of the time.)

## 🚀 Usage

```bash
./irule-validator --help
Usage of ./irule-validator:
  -d, --debug          Run in debug mode
  -p, --print-errors   Print Errors
```

Pro Tip: When using this in a CI/CD pipeline, be sure to call it with `-p` to
get those sweet error printouts you so desperately crave.

## Disclaimer

Does it validate every possible command with perfect accuracy? Not quite.
Full syntax validation is *hard*, and I've realized that F5 iRules are
a bottomless pit of edge cases. 🕳️

Building a complete F5 iRule parser is like trying to solve a puzzle where
the pieces keep changing shape. But hey, we already have a parser that
covers most of the use cases I need — and that's a win! 🎉

## 🧑 Contributing

PRs are welcome! 💥
