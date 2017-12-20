#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
@all
@eventsnap
Feature:  Event Snap
    @oneeventsnap
    Scenario: Register with Local Event Service for Chaincode Events
        Given fabric has channel "mychannel" and p0 joined channel
        
   		And client C1 invokes configuration snap on channel "mychannel" to load "txnsnap" configuration on p0
		When client C1 query chaincode "configurationsnap" on channel "mychannel" with args "refresh" on p0
		And client C1 invokes configuration snap on channel "mychannel" to load "configurationsnap" configuration on p0
        And client C1 invokes configuration snap on channel "mychannel" to load "eventsnap" configuration on p0
        And client C1 waits 15 seconds

        # First clean up from any previous tests
        And client C1 unregisters for chaincode events on channel "mychannel" for chaincode "eventconsumersnap" and event filter "event1"
        And client C1 deletes all chaincode events on channel "mychannel"

        # Register for chaincode events
        Then client C1 registers for chaincode events on channel "mychannel" for chaincode "eventconsumersnap" and event filter "event1"
        And client C1 invokes chaincode "eventconsumersnap" on channel "mychannel" with args "put,key1,value1,event1" on p0
        And client C1 invokes chaincode "eventconsumersnap" on channel "mychannel" with args "put,key1,value1,event2" on p0
        And client C1 waits 2 seconds
        Then client C1 queries for chaincode events on channel "mychannel"
        And client C1 receives a response containing 1 chaincode events for chaincode "eventconsumersnap" and event filter "event1"
        And client C1 unregisters for chaincode events on channel "mychannel" for chaincode "eventconsumersnap" and event filter "event1"

    Scenario: Register with Local Event Service for Tx Status Events
        Given fabric has channel "mychannel" and p0 joined channel
   		And client C1 invokes configuration snap on channel "mychannel" to load "txnsnap" configuration on p0
		When client C1 query chaincode "configurationsnap" on channel "mychannel" with args "refresh" on p0
		And client C1 invokes configuration snap on channel "mychannel" to load "configurationsnap" configuration on p0
        And client C1 invokes configuration snap on channel "mychannel" to load "eventsnap" configuration on p0
        And client C1 waits 15 seconds


        # First clean up from any previous tests
        And client C1 deletes all Tx status events on channel "mychannel"

        # Register for TxStatus events
        And client C1 invokes chaincode "eventconsumersnap" on channel "mychannel" with args "put,key1,value1,event1" and registers for a Tx event
        And client C1 waits 2 seconds
        Then client C1 queries for Tx status events on channel "mychannel"
        And client C1 receives a response containing a Tx Status event for the last transaction ID

    Scenario: Register with Local Event Service for Filtered Block Events
        Given fabric has channel "mychannel" and p0 joined channel
   		And client C1 invokes configuration snap on channel "mychannel" to load "txnsnap" configuration on p0
		When client C1 query chaincode "configurationsnap" on channel "mychannel" with args "refresh" on p0
		And client C1 invokes configuration snap on channel "mychannel" to load "configurationsnap" configuration on p0
        And client C1 invokes configuration snap on channel "mychannel" to load "eventsnap" configuration on p0
        And client C1 waits 15 seconds


        # First clean up from any previous tests
        And client C1 unregisters for filtered block events on channel "mychannel"
        And client C1 deletes all filtered block events on channel "mychannel"

        # Register for filtered block events
        Then client C1 registers for filtered block events on channel "mychannel"
        And client C1 invokes chaincode "eventconsumersnap" on channel "mychannel" with args "put,key1,value1,event1" on p0
        And client C1 invokes chaincode "eventconsumersnap" on channel "mychannel" with args "put,key2,value2,event2" on p0
        And client C1 waits 2 seconds
        Then client C1 queries for filtered block events on channel "mychannel"
        And client C1 receives a response containing 2 filtered block events
        And client C1 unregisters for filtered block events on channel "mychannel"

    Scenario: Register with Local Event Service for Block Events
        Given fabric has channel "mychannel" and p0 joined channel
   		And client C1 invokes configuration snap on channel "mychannel" to load "txnsnap" configuration on p0
		When client C1 query chaincode "configurationsnap" on channel "mychannel" with args "refresh" on p0
		And client C1 invokes configuration snap on channel "mychannel" to load "configurationsnap" configuration on p0
        And client C1 invokes configuration snap on channel "mychannel" to load "eventsnap" configuration on p0
        And client C1 waits 15 seconds

        # First clean up from any previous tests
        And client C1 unregisters for block events on channel "mychannel"
        And client C1 deletes all block events on channel "mychannel"
        
		And client C1 waits 20 seconds

        # Register for block events
        Then client C1 registers for block events on channel "mychannel"
        And client C1 invokes chaincode "eventconsumersnap" on channel "mychannel" with args "put,key1,value1,event1" on p0
        And client C1 invokes chaincode "eventconsumersnap" on channel "mychannel" with args "put,key2,value2,event2" on p0
        And client C1 waits 2 seconds
        Then client C1 queries for block events on channel "mychannel"
        And client C1 receives a response containing 2 block events
        And client C1 unregisters for block events on channel "mychannel"
