#!/bin/bash
#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

set -e

mkdir -p /opt/gopath/src/github.com/hyperledger
mkdir -p /opt/gopath/src/github.com/securekey

cp -r /opt/temp/src/github.com/securekey/fabric-snaps /opt/gopath/src/github.com/securekey

echo "Cloning fabric..."
cd /opt/gopath/src/github.com/hyperledger
git clone https://github.com/hyperledger/fabric.git
git config --global advice.detachedHead false
cd fabric && git checkout $FABRIC_VERSION

git config --global user.email "you@example.com"
git config --global user.name "Your Name"

##Private Data:

# #Collection Store and Collection configuration upon instantiate:
# #https://gerrit.hyperledger.org/r/#/c/14515/ - Open - [FAB-6574] Integrate simpleCollectionStore for gossip
# git fetch https://gerrit.hyperledger.org/r/fabric refs/changes/15/14515/7 && git cherry-pick FETCH_HEAD
# #https://gerrit.hyperledger.org/r/#/c/14517/ - Open - [FAB-6563] Merge External and Internal peers in config
# git fetch https://gerrit.hyperledger.org/r/fabric refs/changes/17/14517/5 && git cherry-pick FETCH_HEAD
# #https://gerrit.hyperledger.org/r/#/c/14291/ - Open - [FAB-5871] create collections at deploy time
# git fetch https://gerrit.hyperledger.org/r/fabric refs/changes/91/14291/23 && git cherry-pick FETCH_HEAD
# #https://gerrit.hyperledger.org/r/#/c/14367/ - Open - [FAB-5871] VSCC to ensure no collection exists
# git fetch https://gerrit.hyperledger.org/r/fabric refs/changes/67/14367/14 && git cherry-pick FETCH_HEAD
# #https://gerrit.hyperledger.org/r/#/c/14371/ - Open - [FAB-6563] CLI support to specify collections
# git fetch https://gerrit.hyperledger.org/r/fabric refs/changes/71/14371/19 && git cherry-pick FETCH_HEAD
# #https://gerrit.hyperledger.org/r/#/c/14519/ - Open - [FAB-6620] Prevent private data send in instantiate
# #git fetch https://gerrit.hyperledger.org/r/fabric refs/changes/19/14519/10 && git cherry-pick FETCH_HEAD
#
# #Purge of private data based on block-to-live:
# #https://gerrit.hyperledger.org/r/#/c/14347/ - Open - [FAB-6552] Block-to-live policy for pvtdata
# git fetch https://gerrit.hyperledger.org/r/fabric refs/changes/47/14347/2 && git cherry-pick FETCH_HEAD
# #https://gerrit.hyperledger.org/r/#/c/14349/ - Open - [FAB-6553] Ledger bookkeeping provider
# git fetch https://gerrit.hyperledger.org/r/fabric refs/changes/49/14349/1 && git cherry-pick FETCH_HEAD
# #https://gerrit.hyperledger.org/r/#/c/14351/ - Open - [FAB-6554] Expiry schedule keeper for pvt-statedb data
# git fetch https://gerrit.hyperledger.org/r/fabric refs/changes/51/14351/1 && git cherry-pick FETCH_HEAD
# #https://gerrit.hyperledger.org/r/#/c/14353/ - Open - [FAB-6555] Purge manager for pvt statedb data
# git fetch https://gerrit.hyperledger.org/r/fabric refs/changes/53/14353/1 && git cherry-pick FETCH_HEAD
# #https://gerrit.hyperledger.org/r/#/c/14355/ - Open - [FAB-6556] Enable purge from pvt statedb
# git fetch https://gerrit.hyperledger.org/r/fabric refs/changes/55/14355/1 && git cherry-pick FETCH_HEAD
# #https://gerrit.hyperledger.org/r/#/c/14511/ - Open - [FAB-6619] purge pvtdata from pvt block store
# git fetch https://gerrit.hyperledger.org/r/fabric refs/changes/11/14511/3 && git cherry-pick FETCH_HEAD
#
# #Other:
# #https://gerrit.hyperledger.org/r/#/c/14769/ - Open - [FAB-6600] Sample chaincode for prvt data
# git fetch https://gerrit.hyperledger.org/r/fabric refs/changes/69/14769/2 && git cherry-pick FETCH_HEAD
# #https://gerrit.hyperledger.org/r/#/c/14791/ - Open - [FAB-6717] - Implement Private Data Funcs in MockStub
# git fetch https://gerrit.hyperledger.org/r/fabric refs/changes/91/14791/1 && git cherry-pick FETCH_HEAD

##Filtered Channel Events:
#https://gerrit.hyperledger.org/r/#/c/15177/ - [FAB-5742] Add channel service listener sample
git fetch https://gerrit.hyperledger.org/r/fabric refs/changes/77/15177/4 && git cherry-pick FETCH_HEAD
#https://gerrit.hyperledger.org/r/#/c/15175/ - [FAB-5744] Add UT coverage for Channel Service server
git fetch https://gerrit.hyperledger.org/r/fabric refs/changes/75/15175/4 && git cherry-pick FETCH_HEAD
#https://gerrit.hyperledger.org/r/#/c/15173/ - [FAB-5738] Implement client logic for Channel Service
git fetch https://gerrit.hyperledger.org/r/fabric refs/changes/73/15173/4 && git cherry-pick FETCH_HEAD
#https://gerrit.hyperledger.org/r/#/c/15171/ - [FAB-5737] Implement server logic for Channel Service
git fetch https://gerrit.hyperledger.org/r/fabric refs/changes/71/15171/4 && git cherry-pick FETCH_HEAD
#https://gerrit.hyperledger.org/r/#/c/15183/ - [FAB-5785] Add Envelope based valid. to RSCC/defaultACL
git fetch https://gerrit.hyperledger.org/r/fabric refs/changes/83/15183/2 && git cherry-pick FETCH_HEAD



##Allow SCC to invoke another SCC:
#https://gerrit.hyperledger.org/r/#/c/15363/ - [FAB-5487] Allow SCC to invoke another SCC
git fetch https://gerrit.hyperledger.org/r/fabric refs/changes/63/15363/5 && git cherry-pick FETCH_HEAD


cd  /opt/gopath/src/github.com/securekey/fabric-snaps

# This can be deleted once we have Go 1.10
echo "Patching Go Compiler..."
patch -p1 $GOROOT/src/cmd/link/internal/ld/data.go ./scripts/patches/linker.patch
echo "Re-building Go Compiler..."
go install -a cmd

echo "Executing move script..."
./scripts/move_snaps.sh

cd /opt/gopath/src/github.com/hyperledger/fabric/plugins

echo "Building plugins..."
go build -tags $GO_BUILD_TAGS -buildmode=plugin -o ./httpsnap.so github.com/hyperledger/fabric/plugins/httpsnap/cmd
go build -tags $GO_BUILD_TAGS -buildmode=plugin -o ./transactionsnap.so github.com/hyperledger/fabric/plugins/transactionsnap/cmd
go build -tags $GO_BUILD_TAGS -buildmode=plugin -o ./membershipsnap.so github.com/hyperledger/fabric/plugins/membershipsnap/cmd
go build -tags $GO_BUILD_TAGS -buildmode=plugin -o ./txnsnapinvoker.so github.com/hyperledger/fabric/plugins/bddtests/fixtures/snapexample/txnsnapinvoker

cp httpsnap.so /opt/temp/src/github.com/securekey/fabric-snaps/build/snaps/
cp transactionsnap.so /opt/temp/src/github.com/securekey/fabric-snaps/build/snaps/
cp membershipsnap.so /opt/temp/src/github.com/securekey/fabric-snaps/build/snaps/
cp txnsnapinvoker.so /opt/temp/src/github.com/securekey/fabric-snaps/build/test/
