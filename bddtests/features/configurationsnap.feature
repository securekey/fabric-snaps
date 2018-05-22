#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
@all
@configurationsnap
Feature:  Test configuration snap Features

	@oneconfig
	Scenario: Invoke Transaction Snap generateKeyPair and ECDSA function
        Given the channel "mychannel" is created and all peers have joined
		And we wait 5 seconds
		And client update config "./fixtures/config/snaps/snaps.json" with mspid "Org1MSP" on the "mychannel" channel
		And client queries chaincode "configurationsnap" with args "generateKeyPair,ECDSA,false" on all peers in the "peerorg1" org on the "mychannel" channel
        And response from "configurationsnap" to client C1 has key and key type is "ECDSA" on p0

	@twoconfig	
	Scenario: Invoke Transaction Snap generateKeyPair and RSA function
        Given the channel "mychannel" is created and all peers have joined
        And we wait 5 seconds
        And client update config "./fixtures/config/snaps/snaps.json" with mspid "Org1MSP" on the "mychannel" channel
		And client queries chaincode "configurationsnap" with args "generateKeyPair,RSA,false" on all peers in the "peerorg1" org on the "mychannel" channel
        And response from "configurationsnap" to client C1 has key and key type is "RSA" on p0


	@threeconfig	
	Scenario: Invoke Transaction Snap generateCSR and ECDSA function. Last argument in call is signature algorithm string
		Given the channel "mychannel" is created and all peers have joined
        And we wait 5 seconds
        And client update config "./fixtures/config/snaps/snaps.json" with mspid "Org1MSP" on the "mychannel" channel
		And client queries chaincode "configurationsnap" with args "generateCSR,ECDSA,false,ECDSAWithSHA1,csrCommonName" on all peers in the "peerorg1" org on the "mychannel" channel
        And response from "configurationsnap" to client C1 has CSR on p0

