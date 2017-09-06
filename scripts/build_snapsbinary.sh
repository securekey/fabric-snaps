#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

go build -ldflags "-linkmode external -extldflags '-static'" -o build/snapsbinary/httpsnap httpsnap/cmd/httpsnap.go
go build -ldflags "-linkmode external -extldflags '-static'" -o build/snapsbinary/transactionsnap  transactionsnap/cmd/transactionsnap.go