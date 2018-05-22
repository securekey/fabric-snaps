#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
@all
@configcli
Feature:  Test config cli Features

	@oneconfigcli
	Scenario: Use config cli to update and delete configs
        Given the channel "mychannel" is created and all peers have joined
		And we wait 5 seconds
		And client update config "./fixtures/config/configcli/org1-config.json" with mspid "Org1MSP" on the "mychannel" channel
        And client "query" config by peer id "peer0.org1.example.com" with mspid "Org1MSP" with app name "app1" with version "1" on the "mychannel" channel
        And response from cli query to client contains value "app1 data v1"
        And client "query" config by peer id "peer0.org1.example.com" with mspid "Org1MSP" with app name "app1" with version "2" on the "mychannel" channel
        And response from cli query to client contains value "app1 data v2"
        And client "delete" config by peer id "peer0.org1.example.com" with mspid "Org1MSP" with app name "app1" with version "1" on the "mychannel" channel
        And response from cli query to client not contains value "app1 data v1"
        And client "query" config by peer id "peer0.org1.example.com" with mspid "Org1MSP" with app name "app1" with version "2" on the "mychannel" channel
        And response from cli query to client contains value "app1 data v2"

    @twoconfigcli
    Scenario: Use config cli to update configs
      Given the channel "mychannel" is created and all peers have joined
      And we wait 5 seconds
      And client update config "./fixtures/config/configcli/org1-config.json" with mspid "Org1MSP" on the "mychannel" channel
      And client update config "./fixtures/config/configcli/org1-config-update.json" with mspid "Org1MSP" on the "mychannel" channel
      And client "query" config by peer id "peer0.org1.example.com" with mspid "Org1MSP" with app name "app1" with version "1" on the "mychannel" channel
      And response from cli query to client contains value "app1 data v1"
      And client "query" config by peer id "peer0.org1.example.com" with mspid "Org1MSP" with app name "app1" with version "2" on the "mychannel" channel
      And response from cli query to client contains value "updated app1 data v2"