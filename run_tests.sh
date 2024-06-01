#!/usr/bin/env sh

go build
go test ./ast
go test ./lexer
go test ./parser
go test ./evaluator
