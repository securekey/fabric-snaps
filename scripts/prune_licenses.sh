#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
set -e


# This script prunes license files that dep refuses to:
# https://golang.github.io/dep/docs/Gopkg.toml.html#prune
#
# From the doc:
# "Out of an abundance of caution, dep non-optionally preserves
# files that may have legal significance."
#

rm -rf vendor/github.com/docker/docker/contrib
