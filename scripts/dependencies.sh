#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# This script installs dependencies for testing tools

echo "Installing dependencies..."
go get -u github.com/axw/gocov/...
go get -u github.com/AlekSi/gocov-xml
