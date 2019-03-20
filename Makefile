#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

# Supported Targets:
# all : runs unit and integration tests
# lint: runs the static code analyzer
# unit-test: runs all the unit tests
# integration-test: runs all the integration tests
# checks: runs all check conditions (license, spelling, linting)
# snaps: generate snaps binary
# channel-artifacts: generates the channel tx files used in the bdd tests

# Release Parameters
BASE_VERSION ?= 0.4.1
IS_RELEASE ?= false

export GO111MODULE=on

ifndef PROJECT_VERSION
ifneq ($(IS_RELEASE),true)
EXTRA_VERSION ?= snapshot-$(shell git rev-parse --short=7 HEAD)
PROJECT_VERSION=$(BASE_VERSION)-$(EXTRA_VERSION)
else
PROJECT_VERSION=$(BASE_VERSION)
endif
endif

FABRIC_NEXT_REPO ?= https://github.com/securekey/fabric-next.git

# This can be a commit hash or a tag (or any git ref)
FABRIC_NEXT_VERSION ?= 3ccc752793333eedb7023ddeeebb0ccc945cfd81
# When this tag is updated, we should also change bddtests/fixtures/.env
# to support running tests without 'make'
ifndef FABRIC_NEXT_IMAGE_TAG
  export FABRIC_NEXT_IMAGE_TAG = 1.4.0-0.0.2-snapshot-3ccc752
endif
# Namespace for the fabric images used in BDD tests
export FABRIC_NEXT_NS ?= securekey
# Namespace for the fabric-snaps image created by 'make docker'
DOCKER_OUTPUT_NS ?= securekey

export ARCH=$(shell go env GOARCH)
CONTAINER_IDS = $(shell docker ps -a -q)
DEV_IMAGES = $(shell docker images dev-* -q)

PROJECT_NAME = securekey/fabric-snaps
PACKAGE_NAME = github.com/$(PROJECT_NAME)


FABRIC_SNAPS_IMAGE_NAME ?= fabric-snaps


#fabric build snaps image parameters
FABRIC_BUILD_SNAPS_IMAGE_NS ?= securekey
FABRIC_BUILD_SNAPS_IMAGE ?= fabric-baseimage
FABRIC_BUILD_SNAPS_IMAGE_VERSION ?= $(ARCH)-0.4.14

GO_BUILD_TAGS ?= "pkcs11"

DOCKER_COMPOSE_CMD ?= docker-compose

export GO_LDFLAGS=-s

GOLANGCI=golangci/golangci-lint:v1.15.0

snaps: version clean
	@echo "Building snap plugins"
	@mkdir -p build/snaps
	@mkdir -p build/test

	@docker run -i --rm \
		-e FABRIC_NEXT_VERSION=$(FABRIC_NEXT_VERSION) \
		-e GO_BUILD_TAGS=$(GO_BUILD_TAGS) \
		-e FABRIC_NEXT_REPO=$(FABRIC_NEXT_REPO) \
		-e GOPROXY=$(GOPROXY) \
		-v ${HOME}/.ssh:/root/.ssh \
		-v $(abspath .):/opt/temp/src/github.com/securekey/fabric-snaps \
		$(FABRIC_BUILD_SNAPS_IMAGE_NS)/$(FABRIC_BUILD_SNAPS_IMAGE):$(FABRIC_BUILD_SNAPS_IMAGE_VERSION) \
		/bin/bash -c "/opt/temp/src/$(PACKAGE_NAME)/scripts/build_plugins.sh"

channel-artifacts:
	@echo "Generating test channel .tx files"
	@docker run -i \
		-v $(abspath .):/opt/gopath/src/$(PACKAGE_NAME) \
		securekey/fabric-tools:$(ARCH)-$(FABRIC_NEXT_IMAGE_TAG) \
		/bin/bash -c "/opt/gopath/src/$(PACKAGE_NAME)/scripts/generate_channeltx.sh"

docker: all
	@docker build -f ./images/fabric-snaps/Dockerfile --no-cache -t $(DOCKER_OUTPUT_NS)/$(FABRIC_SNAPS_IMAGE_NAME):$(ARCH)-$(PROJECT_VERSION) \
	--build-arg FABRIC_NEXT_PEER_IMAGE=$(FABRIC_NEXT_NS)/fabric-peer-softhsm \
	--build-arg ARCH=$(ARCH) \
	--build-arg FABRIC_NEXT_IMAGE_TAG=$(FABRIC_NEXT_IMAGE_TAG) .

checks: license lint spelling check-metrics-doc

.PHONY: license
license: version
	@scripts/check_license.sh

lint:
	@echo "Executing target lint..."
	@docker run -i --rm \
			-e GOPROXY=https://athens:Na5ZcpmKjPM7XZTW@eng-athens.onetap.ca \
			-v $(abspath .):/go/src/github.com/securekey/fabric-snaps \
			golang:1.11.5 \
			/bin/bash -x -c /go/src/github.com/securekey/fabric-snaps/scripts/check_lint.sh

spelling:
	@scripts/check_spelling.sh

unit-test:
	@scripts/unit.sh

pkcs11-unit-test:
	@cd ./bddtests/fixtures && $(DOCKER_COMPOSE_CMD) -f docker-compose-pkcs11-unit-test.yml up --force-recreate --abort-on-container-exit

integration-test: clean snaps cliconfig
	@scripts/integration.sh

http-server:
	@go build -o build/test/httpserver ${PACKAGE_NAME}/bddtests/fixtures/httpserver

cliconfig:
	@go build -o ./build/configcli ./configurationsnap/cmd/configcli

all: version clean checks snaps unit-test pkcs11-unit-test integration-test http-server


check-metrics-doc:
	@echo "METRICS: Checking for outdated reference documentation.."
	@scripts/metrics_doc.sh check

generate-metrics-doc:
	@echo "Generating metrics reference documentation..."
	@scripts/metrics_doc.sh generate

.PHONY: version
version:
	@scripts/check_version.sh

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
