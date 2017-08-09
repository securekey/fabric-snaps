#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
@all
@txnsnap
Feature:  Feature Invoke Transaction Snap 
	Scenario: Invoke Transaction Snap getPeersOfChannel function
        Given fabric has channel "mychannel" and p0 joined channel
        When client C1 query chaincode "txnsnap" on channel "" with args "getPeersOfChannel,mychannel" on p0
        And response from "txnsnap" to client C1 contains value "peer0.org1.example.com:7051"
		
	Scenario: Invoke Transaction Snap endorseAndCommitTransaction,endorseTransaction function
	    Given fabric has channel "mychannel" and p0 joined channel
	    And "test" chaincode "example_cc" version "v1" from path "github.com/example_cc" is installed and instantiated with args "init,a,100,b,200"
		When client C1 query chaincode "txnsnap" on channel "" with args "endorseAndCommitTransaction,mychannel,example_cc,invoke,move,a,b,1" on p0
        And client C1 query chaincode "txnsnap" on channel "" with args "endorseTransaction,mychannel,example_cc,invoke,query,b" on p0
        And response from "txnsnap" to client C1 contains value "201"

    Scenario: Invoke Transaction Snap verifyTransactionProposalSignature function
	    Given fabric has channel "mychannel" and p0 joined channel
	    And "test" chaincode "example_cc1" version "v1" from path "github.com/example_cc" is installed and instantiated with args "init,a,100,b,200"
        When client C1 query chaincode "txnsnap" on channel "" with args "endorseTransaction,mychannel,example_cc1,invoke,query,b" on p0
		And client C1 query chaincode "txnsnap" on channel "" with args "verifyTransactionProposalSignature,mychannel,txProposalBytes" on p0

    Scenario: Invoke Transaction Snap commitTransaction function
	    Given fabric has channel "mychannel" and p0 joined channel
	    And "test" chaincode "example_cc2" version "v1" from path "github.com/example_cc" is installed and instantiated with args "init,a,100,b,200"
        When client C1 query chaincode "txnsnap" on channel "" with args "endorseTransaction,mychannel,example_cc2,invoke,move,a,b,3" on p0
		And client C1 query chaincode "txnsnap" on channel "" with args "commitTransaction,mychannel,tpResponses,true" on p0
        And client C1 query chaincode "txnsnap" on channel "" with args "endorseTransaction,mychannel,example_cc2,invoke,query,b" on p0
        And response from "txnsnap" to client C1 contains value "203"