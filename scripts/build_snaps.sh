#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

peer chaincode package -n httpsnap -p /opt/snapsbinary/httpsnap -v 1.1.0 /opt/snaps/httpsnap.golang -l binary
peer chaincode package -n txnsnap -p /opt/snapsbinary/transactionsnap -v 1.1.0 /opt/snaps/txnsnap.golang -l binary
/bin/chmod 775 /opt/snaps/*