#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

wget -qO- https://storage.googleapis.com/golang/go1.7.3.linux-amd64.tar.gz | tar -xzC /usr/local/
export PATH=/usr/local/go/bin:$PATH
export GOPATH=/opt/gopath
peer chaincode package -n httpsnap -p github.com/securekey/fabric-snaps/httpsnap/cmd -v 1.0.0 /opt/snaps/httpsnap.golang
peer chaincode package -n txnsnap -p github.com/securekey/fabric-snaps/transactionsnap/cmd -v 1.0.0 /opt/snaps/txnsnap.golang
/bin/chmod 775 /opt/snaps/*