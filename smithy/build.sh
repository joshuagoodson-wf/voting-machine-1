#!/usr/bin/env bash

# Get godep
which godep > /dev/null || {
    go get github.com/tools/godep
}

set -e

# Run build
godep go build -v ./...
