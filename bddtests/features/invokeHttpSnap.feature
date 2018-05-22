#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
@all
@httpsnap
Feature:  Invoke Http Snap
    @smoke
    Scenario: Invoke Http Snap
        Given the channel "mychannel" is created and all peers have joined
        And we wait 5 seconds
        And client update config "./fixtures/config/snaps/snaps.json" with mspid "Org1MSP" on the "mychannel" channel
        And "test" chaincode "httpsnaptest_cc" is installed from path "github.com/httpsnaptest_cc" to all peers
        And "test" chaincode "httpsnaptest_cc" is instantiated from path "github.com/httpsnaptest_cc" on the "mychannel" channel with args "init,a,100,b,200" with endorsement policy "" with collection policy ""
        And chaincode "httpsnaptest_cc" is warmed up on all peers on the "mychannel" channel
        When client queries chaincode "httpsnaptest_cc" with args "httpsnap,https://test01.onetap.ca:8443/hello" on all peers in the "peerorg1" org on the "mychannel" channel
		And response from "httpsnaptest_cc" to client contains value "Hello"
