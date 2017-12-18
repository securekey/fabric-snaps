#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
@all
@configurationsnap
Feature:  Test configuration snap Features

    Scenario: Test function "getPublicKeyForLogging" in configuration snap
		Given fabric has channel "mychannel" and p0 joined channel
		When client C1 query chaincode "configurationsnap" on channel "mychannel" with args "getPublicKeyForLogging" on p0
		#Below key needs to be updated if test key value in .scripts/build_snaps ldflags gets changed
        And response from "configurationsnap" to client C1 contains value "SAMPLE-KEY-1234"
        And response from "configurationsnap" to client C1 contains value "SAMPLE-KEYID-1234"

    @oneconfig
	Scenario: Invoke Transaction Snap endorseAndCommitTransaction,endorseTransaction function
	    Given fabric has channel "mychannel" and p0 joined channel
		And client C1 invokes configuration snap on channel "mychannel" to load "txnsnap" configuration on p0
		And client C1 invokes configuration snap on channel "mychannel" to load "configurationsnap" configuration on p0
		And client C1 waits 10 seconds
		When client C1 query chaincode "configurationsnap" on channel "mychannel" with args "refresh" on p0
		And response from "configurationsnap" to client C1 contains value "5"
		#Update config set different refresh interval
		And client C1 copies "./fixtures/config/snaps/configurationsnap/testconfigs/config.yaml" to "./fixtures/config/snaps/configurationsnap/config.yaml"
		And client C1 waits 10 seconds
		And client C1 invokes configuration snap on channel "mychannel" to load "configurationsnap" configuration on p0
		When client C1 query chaincode "configurationsnap" on channel "mychannel" with args "refresh" on p0
		#Verify that refresh interval was updated
		And response from "configurationsnap" to client C1 contains value "8"
		And client C1 waits 20 seconds
		#Reset original refresh interval of 5 secs
		And client C1 copies "./fixtures/config/snaps/configurationsnap/testconfigs/configreset.yaml" to "./fixtures/config/snaps/configurationsnap/config.yaml"
