#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#


# Supported Targets:
# all : runs unit and integration tests
# depend: checks that test dependencies are installed
# unit-test: runs all the unit tests
# integration-test: runs all the integration tests
# checks: runs all check conditions (license, spelling, linting)
# snaps: generate snaps binary
# populate: populates generated files (not included in git) - currently only vendor
# populate-vendor: populate the vendor directory based on the lock
# channel-artifacts: generates the channel tx files used in the bdd tests

ARCH=$(shell uname -m)
CONTAINER_IDS = $(shell docker ps -a -q)
DEV_IMAGES = $(shell docker images dev-* -q)
PACKAGE_NAME = github.com/securekey/fabric-snaps
FABRIC_TOOLS_RELEASE=1.0.2
GO_BUILD_TAGS ?= "experimental"
FABRIC_VERSION ?= 4f7a7c8d696e866d06780e14b10704614a68564b
export GO_LDFLAGS=-s
export GO_DEP_COMMIT=v0.3.0 # the version of dep that will be installed by depend-install (or in the CI)

snaps: clean populate
	@echo "Building snap plugins"
	@mkdir -p build/snaps
	@mkdir -p build/test
	@docker run -i --rm \
		-e FABRIC_VERSION=$(FABRIC_VERSION) \
		-e GO_BUILD_TAGS=$(GO_BUILD_TAGS) \
		-v $(abspath .):/opt/temp/src/github.com/securekey/fabric-snaps \
		d1vyank/fabric-baseimage:x86_64-0.4.2 \
		/bin/bash -c "/opt/temp/src/$(PACKAGE_NAME)/scripts/build_plugins.sh"

channel-artifacts:
	@echo "Generating test channel .tx files"
	@docker run -i \
		-v $(abspath .):/opt/gopath/src/$(PACKAGE_NAME) \
		hyperledger/fabric-tools:$(ARCH)-$(FABRIC_TOOLS_RELEASE) \
		/bin/bash -c "/opt/gopath/src/$(PACKAGE_NAME)/scripts/generate_channeltx.sh"

depend:
	@scripts/dependencies.sh

checks: depend license lint spelling

.PHONY: license
license:
	@scripts/check_license.sh

lint: populate
	@scripts/check_lint.sh

spelling:
	@scripts/check_spelling.sh

unit-test: depend populate
	@scripts/unit.sh

integration-test: clean depend populate snaps pull-fabric-images
	@scripts/integration.sh

http-server:
	@go build -o build/test/httpserver ${PACKAGE_NAME}/bddtests/fixtures/httpserver

pull-fabric-images:
	@docker pull repo.onetap.ca:8443/next/hyperledger/fabric-ca
	@docker pull repo.onetap.ca:8443/next/hyperledger/fabric-orderer
	@docker pull repo.onetap.ca:8443/next/hyperledger/fabric-peer
	@docker pull repo.onetap.ca:8443/next/hyperledger/fabric-couchdb
	@docker pull repo.onetap.ca:8443/next/hyperledger/fabric-tools
	@docker pull repo.onetap.ca:8443/next/hyperledger/fabric-ccenv

all: clean checks snaps unit-test integration-test http-server

populate: populate-vendor

populate-vendor:
	@echo "Populating vendor ..."
	@dep ensure -vendor-only

clean:
	rm -Rf ./bddtests/fixtures/config/extsysccs
	rm -Rf ./bddtests/fixtures/build
	rm -Rf ./bddtests/docker-compose.log
	rm -Rf ./build
	rm -Rf vendor

clean-images:
	@echo "Stopping all containers, pruning containers and images, deleting dev images"
ifneq ($(strip $(CONTAINER_IDS)),)
	@docker stop $(CONTAINER_IDS)
endif
	@docker system prune -f
ifneq ($(strip $(DEV_IMAGES)),)
	@docker rmi $(DEV_IMAGES) -f
endif
