module github.com/securekey/fabric-snaps/util/statemgr

replace github.com/hyperledger/fabric => gerrit.securekey.com/fabric-mod v0.0.0-20190228191641-1628bcab7f94

replace github.com/securekey/fabric-snaps => ../..

require (
	github.com/hyperledger/fabric v1.4.0
	github.com/hyperledger/fabric-sdk-go v0.0.0-20190125204638-b490519efff9
	github.com/securekey/fabric-snaps v0.4.0
	github.com/spf13/viper v0.0.0-20171227194143-aafc9e6bc7b7
	github.com/stretchr/testify v1.3.0
)
