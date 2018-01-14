#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
@all
@configurationsnap
Feature:  Test configuration snap Features

    @oneconfig
	Scenario: Invoke Transaction Snap endorseAndCommitTransaction,endorseTransaction function
	    Given fabric has channel "mychannel" and p0 joined channel

  		And client C1 invokes configuration snap on channel "mychannel" to load "txnsnap" configuration on p0
		And client C1 invokes configuration snap on channel "mychannel" to load "configurationsnap" configuration on p0
		And client C1 invokes configuration snap on channel "mychannel" to load "eventsnap" configuration on p0

		#Valid configuration
		And "test" chaincode "example_cc" version "v1" from path "github.com/example_cc" is installed and instantiated with args "init,a,100,b,200"
        When client C1 query chaincode "txnsnapinvoker" on channel "" with args "txnsnap,endorseAndCommitTransaction,mychannel,example_cc,invoke,move,a,b,0" on p0
        And client C1 query chaincode "txnsnapinvoker" on channel "" with args "txnsnap,endorseTransaction,mychannel,example_cc,invoke,query,b" on p0
        And response from "txnsnapinvoker" to client C1 contains value "200"
