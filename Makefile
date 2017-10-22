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
export GO_LDFLAGS=-s
export GO_DEP_COMMIT=v0.3.0 # the version of dep that will be installed by depend-install (or in the CI)

# Upstream fabric patching (overridable)
THIRDPARTY_FABRIC_CA_BRANCH ?= master
THIRDPARTY_FABRIC_CA_COMMIT ?= 2f9617379ec6c253e610ac02b60b3f963f95ad1d
THIRDPARTY_FABRIC_BRANCH    ?= master
THIRDPARTY_FABRIC_COMMIT    ?= 505eb68f64493db86859b649b91e7b7068139e6f

snaps: clean populate
	@echo "Building snaps..."
	@mkdir -p build/snaps
	@docker run -i \
		-v $(abspath .):/opt/gopath/src/$(PACKAGE_NAME) \
		-v $(abspath build/snaps):/opt/snaps \
		-v $(abspath build/fabricversion.txt):/opt/fabricversion.txt \
		repo.onetap.ca:8443/next/hyperledger/fabric-tools:x86_64-latest \
		/bin/bash -c "export FABRIC_VERSION=1.1.0 ;/opt/gopath/src/$(PACKAGE_NAME)/scripts/build_snaps.sh"


testsnaps: clean populate
	@echo "Building test snaps..."
	@mkdir -p ./bddtests/fixtures/build/testsnaps
	@docker run -i \
		-v $(abspath .):/opt/gopath/src/$(PACKAGE_NAME) \
		-v $(abspath ./bddtests/fixtures/build/testsnaps):/opt/snaps \
		-v $(abspath build/fabricversion.txt):/opt/fabricversion.txt \
		repo.onetap.ca:8443/next/hyperledger/fabric-tools:x86_64-latest \
		/bin/bash -c "export FABRIC_VERSION=1.1.0 ;/opt/gopath/src/$(PACKAGE_NAME)/bddtests/fixtures/config/snaps/txnsnapinvoker/cds.sh"

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

integration-test: clean depend populate snaps-4-bdd
	@scripts/integration.sh

http-server:
	@go build -o build/test/httpserver ${PACKAGE_NAME}/bddtests/fixtures/httpserver


all: clean checks snaps testsnaps unit-test integration-test http-server

snaps-4-bdd: clean checks snaps testsnaps
	@mkdir ./bddtests/fixtures/config/extsysccs
	@cp -r build/snaps/* ./bddtests/fixtures/config/extsysccs/
	@cp -r ./bddtests/fixtures/build/testsnaps/* ./bddtests/fixtures/config/extsysccs/

populate: populate-vendor

populate-vendor:
	@echo "Populating vendor ..."
	@dep ensure -vendor-only

thirdparty-pin:
	@echo "Pinning third party packages ..."
	@UPSTREAM_COMMIT=$(THIRDPARTY_FABRIC_COMMIT) UPSTREAM_BRANCH=$(THIRDPARTY_FABRIC_BRANCH) scripts/third_party_pins/fabric/apply_upstream.sh
	@UPSTREAM_COMMIT=$(THIRDPARTY_FABRIC_CA_COMMIT) UPSTREAM_BRANCH=$(THIRDPARTY_FABRIC_CA_BRANCH) scripts/third_party_pins/fabric-ca/apply_upstream.sh

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
