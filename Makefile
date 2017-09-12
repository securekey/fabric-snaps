#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#


# Supported Targets:
# all : runs unit and integration tests
# depend: checks that test dependencies are installed
# depend-install: installs test dependencies
# unit-test: runs all the unit tests
# integration-test: runs all the integration tests
# checks: runs all check conditions (license, spelling, linting)
# snaps: generate snaps binary
# populate: populates generated files (not included in git) - currently only vendor
# populate-vendor: populate the vendor directory based on the lock 

CONTAINER_IDS = $(shell docker ps -a -q)
DEV_IMAGES = $(shell docker images dev-* -q)
PACKAGE_NAME = github.com/securekey/fabric-snaps
export GO_LDFLAGS=-s
export GO_DEP_COMMIT=v0.3.0 # the version of dep that will be installed by depend-install (or in the CI)
export FABRIC_VERSION=1.1.0-snapshot-4adceaa
export FABRIC_SNAP_DEPEND_INSTALL=true

snaps: clean populate
	@echo "Building snaps..."
	@mkdir -p build/snaps
	@docker run -i \
		-v $(abspath .):/opt/gopath/src/$(PACKAGE_NAME) \
		-v $(abspath build/snaps):/opt/snaps \
		hyperledger/fabric-tools:latest \
		/bin/bash -c "export FABRIC_VERSION=$(FABRIC_VERSION) ;/opt/gopath/src/$(PACKAGE_NAME)/scripts/build_snaps.sh"


testsnaps: clean populate
	@echo "Building test snaps..."
	@mkdir -p ./bddtests/fixtures/build/testsnaps
	@docker run -i \
		-v $(abspath .):/opt/gopath/src/$(PACKAGE_NAME) \
		-v $(abspath ./bddtests/fixtures/build/testsnaps):/opt/snaps \
		hyperledger/fabric-tools:latest \
		/bin/bash -c "export FABRIC_VERSION=$(FABRIC_VERSION) ;/opt/gopath/src/$(PACKAGE_NAME)/bddtests/fixtures/config/snaps/txnsnapinvoker/cds.sh"
	

depend:
	@scripts/dependencies.sh

depend-install:
	@FABRIC_SNAP_DEPEND_INSTALL="true" scripts/dependencies.sh

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
	@docker tag hyperledger/fabric-ccenv:latest hyperledger/fabric-ccenv:x86_64-$(FABRIC_VERSION)
	@scripts/integration.sh


all: clean checks snaps testsnaps unit-test integration-test

snaps-4-bdd: clean checks snaps testsnaps
	@mkdir ./bddtests/fixtures/config/extsysccs
	@cp -r build/snaps/* ./bddtests/fixtures/config/extsysccs/
	@cp -r ./bddtests/fixtures/build/testsnaps/* ./bddtests/fixtures/config/extsysccs/

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
