module github.com/securekey/fabric-snaps/util/statemgr

replace github.com/hyperledger/fabric => gerrit.securekey.com/fabric-next v0.0.0-20190329193704-1e0210cad6dd

replace github.com/securekey/fabric-snaps => ../..

replace github.com/securekey/fabric-snaps/membershipsnap/pkg/membership => ../../membershipsnap/pkg/membership

replace github.com/securekey/fabric-snaps/util/rolesmgr => ../../util/rolesmgr

replace github.com/securekey/fabric-snaps/util/statemgr => ./

require (
	github.com/hyperledger/fabric v1.4.0
	github.com/hyperledger/fabric-sdk-go v0.0.0-20190125204638-b490519efff9
	github.com/securekey/fabric-snaps v0.4.0
	github.com/spf13/viper v0.0.0-20171227194143-aafc9e6bc7b7
	github.com/stretchr/testify v1.3.0
)
