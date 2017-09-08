#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

CONTAINER_IDS = $(shell docker ps -a -q)
DEV_IMAGES = $(shell docker images dev-* -q)
PACKAGE_NAME = github.com/securekey/fabric-snaps
export GO_LDFLAGS=-s

snaps: clean
	@echo "Building snaps..."
	@mkdir -p build/snaps
	@docker run -i \
		-v $(abspath .):/opt/gopath/src/$(PACKAGE_NAME) \
		-v $(abspath build/snaps):/opt/snaps \
		hyperledger/fabric-tools:latest \
		/bin/bash -c "/opt/gopath/src/$(PACKAGE_NAME)/scripts/build_snaps.sh"


testsnaps: clean
	@echo "Building test snaps..."
	@mkdir -p ./bddtests/fixtures/build/testsnaps
	@docker run -i \
		-v $(abspath .):/opt/gopath/src/$(PACKAGE_NAME) \
		-v $(abspath ./bddtests/fixtures/build/testsnaps):/opt/snaps \
		hyperledger/fabric-tools:latest \
		/bin/bash -c "/opt/gopath/src/$(PACKAGE_NAME)/bddtests/fixtures/config/snaps/txnsnapinvoker/cds.sh"
	

depend:
	@scripts/dependencies.sh

checks: depend license lint spelling

.PHONY: license
license:
	@scripts/check_license.sh

lint:
	@scripts/check_lint.sh

spelling:
	@scripts/check_spelling.sh

unit-test: depend
	@scripts/unit.sh

integration-test: clean depend snaps-4-bdd
	@scripts/integration.sh


all: clean checks snaps testsnaps unit-test integration-test

snaps-4-bdd: clean checks snaps testsnaps
	@mkdir ./bddtests/fixtures/config/extsysccs
	@cp -r build/snaps/* ./bddtests/fixtures/config/extsysccs/
	@cp -r ./bddtests/fixtures/build/testsnaps/* ./bddtests/fixtures/config/extsysccs/

clean:
	rm -Rf ./bddtests/fixtures/config/extsysccs
	rm -Rf ./bddtests/fixtures/build
	rm -Rf ./bddtests/docker-compose.log
	rm -Rf ./build

clean-images:
	@echo "Stopping all containers, pruning containers and images, deleting dev images"
ifneq ($(strip $(CONTAINER_IDS)),)
	@docker stop $(CONTAINER_IDS)
endif
	@docker system prune -f
ifneq ($(strip $(DEV_IMAGES)),)
	@docker rmi $(DEV_IMAGES) -f
endif
