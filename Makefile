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

# Release Parameters
BASE_VERSION = 0.2.4
IS_RELEASE = false

ifneq ($(IS_RELEASE),true)
EXTRA_VERSION ?= snapshot-$(shell git rev-parse --short=7 HEAD)
PROJECT_VERSION=$(BASE_VERSION)-$(EXTRA_VERSION)
else
PROJECT_VERSION=$(BASE_VERSION)
endif

# This can be a commit hash or a tag (or any git ref)
FABRIC_NEXT_VERSION = 64b5e1bb6af714362f0b205d2061a3afdec5c755
# When this tag is updated, we should also change bddtests/fixtures/.env
# to support running tests without 'make'
export FABRIC_NEXT_IMAGE_TAG = 1.2.0-0.1.3-snapshot-64b5e1b
# Namespace for the fabric images used in BDD tests
export FABRIC_NEXT_NS ?= securekey
# Namespace for the fabric-snaps image created by 'make docker'
DOCKER_OUTPUT_NS ?= securekey

export ARCH=$(shell go env GOARCH)
CONTAINER_IDS = $(shell docker ps -a -q)
DEV_IMAGES = $(shell docker images dev-* -q)

PROJECT_NAME = securekey/fabric-snaps
PACKAGE_NAME = github.com/$(PROJECT_NAME)


#fabric base image parameters
FABRIC_BASE_IMAGE_NS=securekey
FABRIC_BASE_IMAGE=fabric-baseimage
FABRIC_BASE_IMAGE_VERSION=$(ARCH)-0.4.10

GO_BUILD_TAGS ?= "pkcs11"

FABRIC_SNAPS_POPULATE_VENDOR ?= true

DOCKER_COMPOSE_CMD ?= docker-compose

export GO_LDFLAGS=-s
export GO_DEP_COMMIT=v0.5.0 # the version of dep that will be installed by depend-install (or in the CI)

snaps: clean populate
	@echo "Building snap plugins"
	@mkdir -p build/snaps
	@mkdir -p build/test
	@docker run -i --rm \
		-e FABRIC_NEXT_VERSION=$(FABRIC_NEXT_VERSION) \
		-e GO_BUILD_TAGS=$(GO_BUILD_TAGS) \
		-v $(abspath .):/opt/temp/src/github.com/securekey/fabric-snaps \
		$(FABRIC_BASE_IMAGE_NS)/$(FABRIC_BASE_IMAGE):$(FABRIC_BASE_IMAGE_VERSION) \
		/bin/bash -c "/opt/temp/src/$(PACKAGE_NAME)/scripts/build_plugins.sh"

channel-artifacts:
	@echo "Generating test channel .tx files"
	@docker run -i \
		-v $(abspath .):/opt/gopath/src/$(PACKAGE_NAME) \
		securekey/fabric-tools:$(ARCH)-$(FABRIC_NEXT_IMAGE_TAG) \
		/bin/bash -c "/opt/gopath/src/$(PACKAGE_NAME)/scripts/generate_channeltx.sh"

depend:
	@scripts/dependencies.sh

docker: all
	@docker build -f ./images/fabric-snaps/Dockerfile --no-cache -t $(DOCKER_OUTPUT_NS)/fabric-snaps:$(ARCH)-$(PROJECT_VERSION) \
	--build-arg FABRIC_NEXT_PEER_IMAGE=$(FABRIC_NEXT_NS)/fabric-peer-softhsm \
	--build-arg ARCH=$(ARCH) \
	--build-arg FABRIC_NEXT_IMAGE_TAG=$(FABRIC_NEXT_IMAGE_TAG) .

checks: depend license lint spelling check-dep

.PHONY: license
license:
	@scripts/check_license.sh

lint: populate
	- @scripts/check_lint.sh

spelling:
	@scripts/check_spelling.sh

.PHONY: check-dep
check-dep:
	@dep check -skip-vendor

unit-test: depend populate lint
	@scripts/unit.sh

pkcs11-unit-test: depend populate
	@cd ./bddtests/fixtures && $(DOCKER_COMPOSE_CMD) -f docker-compose-pkcs11-unit-test.yml up --force-recreate --abort-on-container-exit

integration-test: clean depend populate snaps cliconfig
	@scripts/integration.sh

http-server:
	@go build -o build/test/httpserver ${PACKAGE_NAME}/bddtests/fixtures/httpserver

cliconfig:
	@go build -o ./build/configcli ./configurationsnap/cmd/configcli

all: clean checks snaps unit-test pkcs11-unit-test integration-test http-server

populate: populate-vendor

populate-vendor:
ifeq ($(FABRIC_SNAPS_POPULATE_VENDOR),true)
		@echo "Populating vendor ..."
		@dep ensure -vendor-only
		@scripts/prune_licenses.sh
endif


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
	@docker rmi $(docker images securekey/* -aq)
