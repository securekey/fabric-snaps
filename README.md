# Snap

Snap is precompiled fabric extension that implements shim.ChaincodeStubInterface

### Snap Features

- Access the Peer keys
- Make calls outside of the Fabric container
- Invoke a local or remote SCC or User CC, on any channel, using SDK
- Invoke another Snap locally
- Listen to local or remote Fabric event, using SDK
- Support the Peer management services (health check, etc.)

### List of implemented methods from shim.ChaincodeStubInterface

- GetArgs() [][]byte 
- GetStringArgs() []string 
- GetFunctionAndParameters()

When non implemented method from shim.ChaincodeStubInterface was invoked it will return 'Required functionality was not implemented' within error field.

### Snap Invocation

Currently snaps are invoked from fabric peer SCC.

### gRPC Support ##

GRPC definition was specified in "github.com/securekey/fabric-snaps/api/protos"


### Testing

Snap has unit tests that validate:

- implemented methods from shim.ChaincodeStubInterface 
- unimplemented methods from shim.ChaincodeStubInterface 
- invoke on registered snap
- invoke on non registered (non existing) snap
- invoke on registered but not configured snap

### Snap Deployment (TODO)

