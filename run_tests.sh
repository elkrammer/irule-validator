#!/usr/bin/env bash

run_and_check() {
  if ! "$@"; then
    echo "Test '$*' failed"
    exit 1
  fi
}

go build
run_and_check go test ./ast
run_and_check go test ./lexer
run_and_check go test ./parser

# Get a list of all files in the test-data directory
test_files=(test-data/*)
exclude_files=(
  "active_members.irule"
  "class_match.irule"
  "complex3.irule"
  "cookie.irule"
)

# Initialize counters and arrays to store results
total_tests=0
successful_tests=0
success_output=()
failure_output=()

# Iterate over each test file and run the validator
for test_file_path in "${test_files[@]}"; do
  test_file=$(basename "$test_file_path")

  # Skip excluded files by checking each element of the exclude array
  skip_file=false
  for exclude in "${exclude_files[@]}"; do
    if [ "$test_file" == "$exclude" ]; then
      skip_file=true
      break
    fi
  done

  if [ "$skip_file" = true ]; then
    continue
  fi

  result=$(./irule-validator -p "$test_file_path")
  exit_code=$? # Capture the exit code immediately

  # Store the output in appropriate array based on success or failure
  if [ $exit_code -eq 0 ]; then
    success_output+=("$result")
    successful_tests=$((successful_tests + 1))
  else
    failure_output+=("$result")
  fi

  # Increment total tests
  total_tests=$((total_tests + 1))
done

# Print the success outputs first
for success in "${success_output[@]}"; do
  echo "$success"
done

# Print the failure outputs next
for failure in "${failure_output[@]}"; do
  echo "$failure"
done

# Print the summary
printf "%*s\n" 60 "" | tr ' ' '-'
echo "Test Data results: $successful_tests/$total_tests"
