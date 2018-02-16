#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
@all
@txnsnap
Feature:  Feature Invoke Transaction Snap
    
    @onetxn
	Scenario: Invoke Transaction Snap commitTransaction,endorseTransaction function
	    Given the channel "mychannel" is created and all peers have joined
        And client invokes configuration snap on channel "mychannel" to load "eventsnap,txnsnap,configurationsnap" configuration on all peers
        And we wait 15 seconds
		And "test" chaincode "example_cc" is installed from path "github.com/example_cc" to all peers
        And "test" chaincode "example_cc" is instantiated from path "github.com/example_cc" on the "mychannel" channel with args "init,a,100,b,200" with endorsement policy "" with collection policy ""
        And chaincode "example_cc" is warmed up on all peers on the "mychannel" channel
		When client queries system chaincode "txnsnapinvoker" with args "txnsnap,commitTransaction,mychannel,example_cc,invoke,move,a,b,1" on org "peerorg1" peer on the "mychannel" channel
		And client queries system chaincode "txnsnapinvoker" with args "txnsnap,endorseTransaction,mychannel,example_cc,invoke,query,b" on org "peerorg1" peer on the "mychannel" channel
        And response from "txnsnapinvoker" to client contains value "201"

	@twotxn
    Scenario: Invoke Transaction Snap verifyTransactionProposalSignature function
	    Given the channel "mychannel" is created and all peers have joined
        And client invokes configuration snap on channel "mychannel" to load "eventsnap,txnsnap,configurationsnap" configuration on all peers
        And we wait 15 seconds
	    And "test" chaincode "example_cc1" is installed from path "github.com/example_cc" to all peers
        And "test" chaincode "example_cc1" is instantiated from path "github.com/example_cc" on the "mychannel" channel with args "init,a,100,b,200" with endorsement policy "" with collection policy ""
        And chaincode "example_cc1" is warmed up on all peers on the "mychannel" channel
		When client queries system chaincode "txnsnapinvoker" with args "txnsnap,endorseTransaction,mychannel,example_cc1,invoke,query,b" on org "peerorg1" peer on the "mychannel" channel
        And client queries system chaincode "txnsnapinvoker" with args "txnsnap,verifyTransactionProposalSignature,mychannel,txProposalBytes" on org "peerorg1" peer on the "mychannel" channel