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
        And we wait 5 seconds
        And client update config "./fixtures/config/snaps/snaps.json" with mspid "Org1MSP" on the "mychannel" channel
		And "test" chaincode "example_cc" is installed from path "github.com/example_cc" to all peers
        And "test" chaincode "example_cc" is instantiated from path "github.com/example_cc" on the "mychannel" channel with args "init,a,100,b,200" with endorsement policy "" with collection policy ""
        And chaincode "example_cc" is warmed up on all peers on the "mychannel" channel
		When client queries system chaincode "txnsnapinvoker" with args "txnsnap,commitTransaction,mychannel,example_cc,invoke,move,a,b,1" on org "peerorg1" peer on the "mychannel" channel
		And client queries system chaincode "txnsnapinvoker" with args "txnsnap,endorseTransaction,mychannel,example_cc,invoke,query,b" on org "peerorg1" peer on the "mychannel" channel
        And response from "txnsnapinvoker" to client equal value "201"

	@twotxn
    Scenario: Invoke Transaction Snap verifyTransactionProposalSignature function
	    Given the channel "mychannel" is created and all peers have joined
        And we wait 5 seconds
        And client update config "./fixtures/config/snaps/snaps.json" with mspid "Org1MSP" on the "mychannel" channel
	    And "test" chaincode "example_cc1" is installed from path "github.com/example_cc" to all peers
        And "test" chaincode "example_cc1" is instantiated from path "github.com/example_cc" on the "mychannel" channel with args "init,a,100,b,200" with endorsement policy "" with collection policy ""
        And chaincode "example_cc1" is warmed up on all peers on the "mychannel" channel
        And client queries system chaincode "txnsnapinvoker" with args "txnsnap,verifyTransactionProposalSignature,mychannel,txProposalBytes" on org "peerorg1" peer on the "mychannel" channel

@unsafeQuery
Scenario: Invoke Transaction Snap verifyTransactionProposalSignature function
  Given the channel "mychannel" is created and all peers have joined
    And we wait 5 seconds
    And client update config "./fixtures/config/snaps/snaps.json" with mspid "Org1MSP" on the "mychannel" channel
    And "test" chaincode "readtest_cc" is installed from path "github.com/readtest_cc" to all peers
    And "test" chaincode "readtest_cc" is instantiated from path "github.com/readtest_cc" on the "mychannel" channel with args "init,k1,hello,k2,world" with endorsement policy "" with collection policy ""
    And chaincode "readtest_cc" is warmed up on all peers on the "mychannel" channel
When client invokes chaincode "readtest_cc" with args "concat,mychannel,readtest_cc,k1,k2,k3" on a peer in the "peerorg1" org on the "mychannel" channel it gets response "helloworld" and the read set is empty
