#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# && [ $vendorPathRelative != *"fabric-sdk-go/internal"* ] && [ $vendorPathRelative != *"fabric-sdk-go/third_party"* ]
readFiles(){
    for file in "$1"/*
    do
    if [ -d "$file" ]
    then
      readFiles "$file"
	else
	vendorPathRelative=$(echo "$file" |grep -P '\/vendor.*' -o)
	fabricVendorFile=$(echo "$fabricPath$vendorPathRelative")
	if [ -f "$fabricVendorFile" ] || [[ $vendorPathRelative == *"vendor/github.com/hyperledger/fabric/"* ]]
	then
		#echo "delete $file"
		rm -rf $file
	fi
    fi
    done
}
fabricPath=$1
vendorPath=$2
echo "fabricPath: $fabricPath"
echo "vendorPath: $vendorPath"
readFiles "$vendorPath"
