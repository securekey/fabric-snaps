#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

go build -ldflags "-linkmode external -extldflags '-static'" -o bddtests/fixtures/config/snapsbinary/txnsnapinvoker bddtests/fixtures/snapexample/txnsnapinvoker/txnsnapinvoker.go