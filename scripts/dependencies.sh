#!/bin/bash
# This script installs dependencies for testing tools

echo "Installing dependencies..."
go get -u github.com/axw/gocov/...
go get -u github.com/AlekSi/gocov-xml
