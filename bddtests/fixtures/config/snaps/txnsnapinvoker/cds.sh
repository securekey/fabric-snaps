#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

export PATH=/usr/local/go/bin:$PATH
NAME=txnsnapinvoker
CDS=/opt/extsysccs/$NAME.golang

peer chaincode package -n $NAME -p /opt/snapsbinary/txnsnapinvoker -v 1.1.0 $CDS -l binary

chmod 775 $CDS
