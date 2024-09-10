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
  "load-balancing.irule"
  "ssl-offload.irule"
)

# Calculate the length of the longest test file name
max_length=0
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

  if [ ${#test_file} -gt $max_length ]; then
    max_length=${#test_file}
  fi
done

# Initialize counters
total_tests=0
successful_tests=0

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

  result=$(./irule-validator "$test_file_path")
  exit_code=$? # Capture the exit code immediately

  # Pad the test file name with spaces to align the output
  printf "%-${max_length}s: %s\n" "$test_file" "$result"

  # Increment total tests
  total_tests=$((total_tests + 1))

  # Check if the exit code indicates success
  if [ $exit_code -eq 0 ]; then
    successful_tests=$((successful_tests + 1))
  fi
done

# Print the summary
echo "----------------------------------"
echo "Test Data results: $successful_tests/$total_tests"
