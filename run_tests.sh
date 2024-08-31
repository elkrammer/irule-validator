#!/usr/bin/env bash

go build
go test ./ast
go test ./lexer
go test ./parser

test_files=(
  "complex.irule"
  "complex2.irule"
  "header.irule"
  "if-else.irule"
  "if.irule"
  "switch.irule"
  "set.irule"
  "hello.tcl"
  "http-to-https.irule"
)

# Calculate the length of the longest test file name
max_length=0
for test_file in "${test_files[@]}"; do
  if [ ${#test_file} -gt $max_length ]; then
    max_length=${#test_file}
  fi
done

# Iterate over each test file and run the validator
for test_file in "${test_files[@]}"; do
  result=$(./irule-validator "test-data/$test_file")
  # Pad the test file name with spaces to align the output
  printf "%-${max_length}s: %s\n" "$test_file" "$result"
done
