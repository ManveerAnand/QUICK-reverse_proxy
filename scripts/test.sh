#!/bin/bash

# This script is used to run tests for the QUIC reverse proxy application.

# Exit immediately if a command exits with a non-zero status.
set -e

# Define the test directory
TEST_DIR="./internal"

# Run unit tests
echo "Running unit tests..."
go test -v $TEST_DIR/...

# Run integration tests
echo "Running integration tests..."
go test -v ./...

# Check for race conditions
echo "Checking for race conditions..."
go test -race -v $TEST_DIR/...

# Run linters
echo "Running linters..."
golangci-lint run

# Run code coverage
echo "Running code coverage..."
go test -coverprofile=coverage.out -v $TEST_DIR/...
go tool cover -html=coverage.out -o coverage.html

echo "All tests completed successfully."