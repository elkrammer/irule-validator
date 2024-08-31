#!/usr/bin/env bash

go build
go test ./ast
go test ./lexer
go test ./parser

# Get a list of all files in the test-data directory
# test_files=(test-data/*)
test_files=(
  "complex.irule"
  "complex2.irule"
  "header.irule"
  "headers.irule"
  "if-else.irule"
  "if.irule"
  "switch.irule"
  "set.irule"
  "hello.tcl"
  "http-to-https.irule"
)

# Calculate the length of the longest test file name
max_length=0
for test_file_path in "${test_files[@]}"; do
  test_file=$(basename "$test_file_path")
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

# for test_file_path in "${test_files[@]}"; do
#   test_file=$(basename "$test_file_path")
#   result=$(./irule-validator "$test_file_path")
#   # Pad the test file name with spaces to align the output
#   printf "%-${max_length}s: %s\n" "$test_file" "$result"
# done
