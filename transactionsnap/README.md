# Transaction Snap

##### Unsafe GetState

This function allows a caller to query the peer's state database without producing a read set. Only CouchDB is supported.

To use this feature invoke the transaction snap with the following args:
`response := stub.InvokeChaincode("txnsnap", [][]byte{[]byte("unsafeGetState"), []byte(channelID), []byte(ccID), []byte(key)}, "")`
If successful, the response payload will contain the value corresponding to the requested key.
For a complete example, refer to the BDD tests.

Note: Omitting a value from the read set is considered unsafe because it bypasses the peer's built-in commit-time checks against dirty and phantom reads. The caller is responsible for preventing these issues.
