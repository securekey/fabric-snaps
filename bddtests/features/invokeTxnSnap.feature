#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
@all
@txnsnap
Feature:  Feature Invoke Transaction Snap
	@smoke
	Scenario: Invoke Transaction Snap getPeersOfChannel function
        Given fabric has channel "mychannel" and p0 joined channel
		And client C1 invokes configuration snap on channel "mychannel" to load "eventsnap" configuration on p0
		And client C1 invokes configuration snap on channel "mychannel" to load "txnsnap" configuration on p0
		And client C1 invokes configuration snap on channel "mychannel" to load "configurationsnap" configuration on p0
	    And client C1 waits 15 seconds
		
		When client C1 query chaincode "txnsnapinvoker" on channel "" with args "txnsnap,getPeersOfChannel,mychannel" on p0
        And response from "txnsnapinvoker" to client C1 contains value p0
    @onetxn
	Scenario: Invoke Transaction Snap endorseAndCommitTransaction,endorseTransaction function
	    Given fabric has channel "mychannel" and p0 joined channel
		And client C1 invokes configuration snap on channel "mychannel" to load "eventsnap" configuration on p0
   		And client C1 invokes configuration snap on channel "mychannel" to load "txnsnap" configuration on p0
		And client C1 invokes configuration snap on channel "mychannel" to load "configurationsnap" configuration on p0
	    And client C1 waits 15 seconds


	    And "test" chaincode "example_cc" version "v1" from path "github.com/example_cc" is installed and instantiated with args "init,a,100,b,200"
        When client C1 query chaincode "txnsnapinvoker" on channel "" with args "txnsnap,endorseAndCommitTransaction,mychannel,example_cc,invoke,move,a,b,1" on p0
        And client C1 query chaincode "txnsnapinvoker" on channel "" with args "txnsnap,endorseTransaction,mychannel,example_cc,invoke,query,b" on p0
        And response from "txnsnapinvoker" to client C1 contains value "201"
	@twotxn
    Scenario: Invoke Transaction Snap verifyTransactionProposalSignature function
	    Given fabric has channel "mychannel" and p0 joined channel
		And client C1 invokes configuration snap on channel "mychannel" to load "eventsnap" configuration on p0
   		And client C1 invokes configuration snap on channel "mychannel" to load "txnsnap" configuration on p0
		And client C1 invokes configuration snap on channel "mychannel" to load "configurationsnap" configuration on p0
       And client C1 waits 15 seconds



	    And "test" chaincode "example_cc1" version "v1" from path "github.com/example_cc" is installed and instantiated with args "init,a,100,b,200"
        When client C1 query chaincode "txnsnapinvoker" on channel "" with args "txnsnap,endorseTransaction,mychannel,example_cc1,invoke,query,b" on p0
		And client C1 query chaincode "txnsnapinvoker" on channel "" with args "txnsnap,verifyTransactionProposalSignature,mychannel,txProposalBytes" on p0

    Scenario: Invoke Transaction Snap commitTransaction function
	    Given fabric has channel "mychannel" and p0 joined channel
		And client C1 invokes configuration snap on channel "mychannel" to load "eventsnap" configuration on p0
   		And client C1 invokes configuration snap on channel "mychannel" to load "txnsnap" configuration on p0
		And client C1 invokes configuration snap on channel "mychannel" to load "configurationsnap" configuration on p0
	    And client C1 waits 15 seconds
  
	   
	    And "test" chaincode "example_cc2" version "v1" from path "github.com/example_cc" is installed and instantiated with args "init,a,100,b,200"
        When client C1 query chaincode "txnsnapinvoker" on channel "" with args "txnsnap,endorseTransaction,mychannel,example_cc2,invoke,move,a,b,3" on p0
		And client C1 query chaincode "txnsnapinvoker" on channel "" with args "txnsnap,commitTransaction,mychannel,tpResponses,true" on p0
        And client C1 query chaincode "txnsnapinvoker" on channel "" with args "txnsnap,endorseTransaction,mychannel,example_cc2,invoke,query,b" on p0
        And response from "txnsnapinvoker" to client C1 contains value "203"
