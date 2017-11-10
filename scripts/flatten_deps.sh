#!/bin/bash
#
# Copyright SecureKey Technologies Inc.
# This file contains software code that is the intellectual property of SecureKey.
# SecureKey reserves all rights in the code and you may not use it without written permission from SecureKey.
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
