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
        Given fabric has channel "mychannel" and p0 joined channel
        And HTTPS Server has been started
        And "test" chaincode "httpsnaptest_cc" version "v1" from path "github.com/httpsnaptest_cc" is installed and instantiated with args ""
        When client C1 query chaincode "httpsnaptest_cc" on channel "mychannel" with args "httpsnap,https://test01.onetap.ca/hello" on p0
        And response from "httpsnaptest_cc" to client C1 contains value "Hello"
