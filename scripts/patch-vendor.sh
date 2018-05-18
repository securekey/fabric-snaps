#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
set -e

echo "Applying patches to vendor directory"

patch -p1 ./vendor/github.com/hyperledger/fabric-sdk-go/pkg/fab/comm/connector.go ./scripts/patches/fix-panic-on-close.patch
patch -p1 ./vendor/github.com/hyperledger/fabric-sdk-go/pkg/fab/comm/connector.go ./scripts/patches/resolve_meta_linter_connector.patch
patch -p1 ./vendor/github.com/hyperledger/fabric-sdk-go/pkg/fab/comm/connector.go ./scripts/patches/fix-connector-deadlock.patch
