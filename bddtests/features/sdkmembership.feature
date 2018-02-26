#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
@all
@sdkmembershipdiscovery
Feature:  Feature Call Membership SDK service that Invokes Membership Snap

	Scenario: Call Membership SDK service that Invokes Membership Snap getPeersOfChannel function behind the scenes, simulated by a client from org1
        Given the channel "mychannel" is created and all peers have joined
        And client invokes configuration snap on channel "mychannel" to load "eventsnap,txnsnap,configurationsnap" configuration on all peers
        And we wait 15 seconds

        When client C1 creates a new membership service provider with args "mychannel,peerorg1,10"
        And client C1 creates a new membership service with args "mychannel"
        And client C1 calls GetPeers function on membership service
        And we wait 15 seconds
        And response from membership service GetPeers function to client contains value "peer0.org1.example.com:7051"
