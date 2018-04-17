#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
@all
@acltestsnap
Feature:  Test ACL with resources

    @checkacl
    Scenario: Invoke acltestsnap to check an arbitrary resource
        Given the channel "mychannel" is created and all peers have joined
        When client queries system chaincode "acltestsnap" with args "acltestsnap,invoke" on org "peerorg1" peer on the "mychannel" channel
        And response from "acltestsnap" to client equal value "done"