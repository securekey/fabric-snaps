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