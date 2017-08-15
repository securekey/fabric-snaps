#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

export PATH=/usr/local/go/bin:$PATH
NAME=txnsnap
CDS=/opt/extsysccs/$NAME.golang
SOURCE=github.com/securekey/fabric-snaps/transactionsnap/cmd

peer chaincode package -n $NAME -p $SOURCE -v 1.0.0 $CDS

chmod 775 $CDS
