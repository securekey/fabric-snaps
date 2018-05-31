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
		And client update config "./fixtures/config/configcli/org1-config.json" with mspid "Org1MSP" with orgid "peerorg1" on the "mychannel" channel
        And client "query" config by peer id "peer0.org1.example.com" with mspid "Org1MSP" with app name "app1" with app version "1" with comp name "" with comp version "" on the "mychannel" channel
        And response from cli query to client contains value "app1 data v1"
        And client "query" config by peer id "peer0.org1.example.com" with mspid "Org1MSP" with app name "app1" with app version "2" with comp name "" with comp version "" on the "mychannel" channel
        And response from cli query to client contains value "app1 data v2"
        And client "delete" config by peer id "peer0.org1.example.com" with mspid "Org1MSP" with app name "app1" with app version "1" with comp name "" with comp version "" on the "mychannel" channel
        And response from cli query to client not contains value "app1 data v1"
        And client "query" config by peer id "peer0.org1.example.com" with mspid "Org1MSP" with app name "app1" with app version "2" with comp name "" with comp version "" on the "mychannel" channel
        And response from cli query to client contains value "app1 data v2"

    @twoconfigcli
    Scenario: Use config cli to update configs
      Given the channel "mychannel" is created and all peers have joined
      And we wait 5 seconds
      And client update config "./fixtures/config/configcli/org1-config.json" with mspid "Org1MSP" with orgid "peerorg1" on the "mychannel" channel
      And client update config "./fixtures/config/configcli/org1-config-update.json" with mspid "Org1MSP" with orgid "peerorg1" on the "mychannel" channel
      And client "query" config by peer id "peer0.org1.example.com" with mspid "Org1MSP" with app name "app1" with app version "1" with comp name "" with comp version "" on the "mychannel" channel
      And response from cli query to client contains value "app1 data v1"
      And client "query" config by peer id "peer0.org1.example.com" with mspid "Org1MSP" with app name "app1" with app version "2" with comp name "" with comp version "" on the "mychannel" channel
      And response from cli query to client contains value "updated app1 data v2"

    @threeconfigcli
    Scenario: Use config cli to update configs
      Given the channel "mychannel" is created and all peers have joined
      And we wait 5 seconds
      And client update config "./fixtures/config/configcli/org1-peerless-config.json" with mspid "Org1MSP" with orgid "peerorg1" on the "mychannel" channel
      And client "query" config by peer id "" with mspid "Org1MSP" with app name "app1" with app version "1" with comp name "" with comp version "" on the "mychannel" channel
      And response from cli query to client contains value "config goes here"



    @fourconfigcli
    Scenario: Use config cli to get components
     Given the channel "mychannel" is created and all peers have joined
     And we wait 5 seconds
     And client update config "./fixtures/config/configcli/org1-peerless-config.json" with mspid "Org1MSP" with orgid "peerorg1" on the "mychannel" channel
     # query with component version
     And client "query" config by peer id "" with mspid "Org1MSP" with app name "app2" with app version "1" with comp name "comp1" with comp version "1" on the "mychannel" channel
     And response from cli query to client contains value "comp1 data ver 1"
     And response from cli query to client not contains value "comp1 data ver 2"
     # query without component version
     And client "query" config by peer id "" with mspid "Org1MSP" with app name "app2" with app version "1" with comp name "comp1" with comp version "" on the "mychannel" channel
     And response from cli query to client contains value "comp1 data ver 1"
     And response from cli query to client contains value "comp1 data ver 2"
     And response from cli query to client not contains value "comp2 data ver 1"
     # delete with component version
     And client "delete" config by peer id "" with mspid "Org1MSP" with app name "app2" with app version "1" with comp name "comp1" with comp version "1" on the "mychannel" channel
     And client "query" config by peer id "" with mspid "Org1MSP" with app name "app2" with app version "1" with comp name "comp1" with comp version "" on the "mychannel" channel
     And response from cli query to client not contains value "comp1 data ver 1"
     And response from cli query to client contains value "comp1 data ver 2"
     # delete without component version
     And client "delete" config by peer id "" with mspid "Org1MSP" with app name "app2" with app version "1" with comp name "comp1" with comp version "" on the "mychannel" channel
     And client "query" config by peer id "" with mspid "Org1MSP" with app name "app2" with app version "1" with comp name "comp1" with comp version "" on the "mychannel" channel
     And response from cli query to client not contains value "comp1 data ver 2"
     And client "query" config by peer id "" with mspid "Org1MSP" with app name "app2" with app version "1" with comp name "comp2" with comp version "" on the "mychannel" channel
     And response from cli query to client not contains value "comp2 data ver 2"