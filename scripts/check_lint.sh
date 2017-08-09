#!/bin/bash
#
# Copyright Greg Haskins, IBM Corp, SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# This script runs Go linting and vetting tools

set -e


echo "Running linters..."

   echo "Checking golint"
   OUTPUT="$(golint $(go list ./... | grep -v /vendor/))"
   if [[ $OUTPUT ]]; then
      echo "YOU MUST FIX THE FOLLOWING THE FOLLOWING GOLINT SUGGESTIONS:"
      printf "$OUTPUT\n"
      exit 1
   fi

   echo "Checking govet"
   OUTPUT="$(go vet  $(go list ./... | grep -v /vendor/))"
   if [[ $OUTPUT ]]; then
      echo "YOU MUST FIX THE FOLLOWING THE FOLLOWING GOVET SUGGESTIONS:"
      printf "$OUTPUT\n"
      exit 1
   fi

   echo "Checking gofmt"
   found=`gofmt -l \`find $i -name "*.go" |grep -v "./vendor"\` 2>&1`
   if [ $? -ne 0 ]; then
      echo "The following files need reformatting with 'gofmt -w <file>':"
      printf "$badformat\n"
      exit 1
   fi
