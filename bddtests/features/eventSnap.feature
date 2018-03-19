#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
@all
@eventsnap
Feature:  Event Snap
    # @eventsnapone
    # Scenario: Register with Local Event Service for Chaincode Events
    #     Given the channel "mychannel" is created and all peers have joined
    #     And client invokes configuration snap on channel "mychannel" to load "eventsnap,txnsnap,configurationsnap" configuration on all peers
    #     And we wait 15 seconds

    #     # First clean up from any previous tests
    #     And client C1 unregisters for chaincode events on channel "mychannel" for chaincode "eventconsumersnap" and event filter "event1"
    #     And client C1 deletes all chaincode events on channel "mychannel"

    #     # Register for chaincode events
    #     Then client C1 registers for chaincode events on channel "mychannel" for chaincode "eventconsumersnap" and event filter "event1"
    #     And client invokes chaincode "eventconsumersnap" with args "put,key1,value1,event1" on all peers in the "peerorg1" org on the "mychannel" channel
    #     And client invokes chaincode "eventconsumersnap" with args "put,key1,value1,event2" on all peers in the "peerorg1" org on the "mychannel" channel
    #     And we wait 2 seconds
    #     Then client C1 queries for chaincode events on channel "mychannel"
    #     And client C1 receives a response containing 1 chaincode events for chaincode "eventconsumersnap" and event filter "event1"
    #     And client C1 unregisters for chaincode events on channel "mychannel" for chaincode "eventconsumersnap" and event filter "event1"

    # @eventsnaptwo
    # Scenario: Register with Local Event Service for Tx Status Events
    #     Given the channel "mychannel" is created and all peers have joined
    #     And client invokes configuration snap on channel "mychannel" to load "eventsnap,txnsnap,configurationsnap" configuration on all peers
    #     And we wait 15 seconds

    #     # First clean up from any previous tests
    #     And client C1 deletes all Tx status events on channel "mychannel"

    #     # Register for TxStatus events
    #     And client C1 invokes chaincode "eventconsumersnap" on channel "mychannel" with args "put,key1,value1,event1" and registers for a Tx event
    #     And we wait 2 seconds
    #     Then client C1 queries for Tx status events on channel "mychannel"
    #     And client C1 receives a response containing a Tx Status event for the last transaction ID

    # @eventsnapthree
    # Scenario: Register with Local Event Service for Filtered Block Events
    #     Given the channel "mychannel" is created and all peers have joined
    #     And client invokes configuration snap on channel "mychannel" to load "eventsnap,txnsnap,configurationsnap" configuration on all peers
    #     And we wait 15 seconds

    #     # First clean up from any previous tests
    #     And client C1 unregisters for filtered block events on channel "mychannel"
    #     And client C1 deletes all filtered block events on channel "mychannel"

    #     # Register for filtered block events
    #     Then client C1 registers for filtered block events on channel "mychannel"
    #     And client invokes chaincode "eventconsumersnap" with args "put,key1,value1,event1" on all peers in the "peerorg1" org on the "mychannel" channel
    #     And client invokes chaincode "eventconsumersnap" with args "put,key2,value2,event2" on all peers in the "peerorg1" org on the "mychannel" channel
    #     And we wait 2 seconds
    #     Then client C1 queries for filtered block events on channel "mychannel"
    #     # Test case need to be fixed: https://jira.securekey.com/browse/DEV-5035
    #     # And client C1 receives a response containing 2 filtered block events
    #     And client C1 unregisters for filtered block events on channel "mychannel"

    # @eventsnapfour
    # Scenario: Register with Local Event Service for Block Events
    #     Given the channel "mychannel" is created and all peers have joined
    #     And client invokes configuration snap on channel "mychannel" to load "eventsnap,txnsnap,configurationsnap" configuration on all peers
    #     And we wait 15 seconds


    #     # First clean up from any previous tests
    #     And client C1 unregisters for block events on channel "mychannel"
    #     And client C1 deletes all block events on channel "mychannel"

	#     And we wait 20 seconds

    #     # Register for block events
    #     Then client C1 registers for block events on channel "mychannel"
    #     And client invokes chaincode "eventconsumersnap" with args "put,key1,value1,event1" on all peers in the "peerorg1" org on the "mychannel" channel
    #     And client invokes chaincode "eventconsumersnap" with args "put,key2,value2,event2" on all peers in the "peerorg1" org on the "mychannel" channel
    #     And we wait 2 seconds
    #     Then client C1 queries for block events on channel "mychannel"
    #     And client C1 receives a response containing 2 block events
    #     And client C1 unregisters for block events on channel "mychannel"
