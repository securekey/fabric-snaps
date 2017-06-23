#  Copyright SecureKey Technologies Inc.
#  This file contains software code that is the intellectual property of SecureKey.
#  SecureKey reserves all rights in the code and you may not use it without
#	 written permission from SecureKey.

# This Makefile assumes a working Golang and Docker setup
# The following targets are supported:
# snaps: Build the snaps daemon binary
# docker: Build the snaps docker image
# unit-test: Run snaps unit tests
# clean: Remove build files and images

PROJECT_NAME = securekey/fabric-snaps
PACKAGE_NAME = github.com/$(PROJECT_NAME)
ARCH = $(shell uname -m)
IMAGE_NAME = snaps
BASEIMAGE_RELEASE = 0.3.1
GO_LDFLAGS = -linkmode external -extldflags '-static -lpthread'
CONTAINER_IDS = $(shell docker ps -a -q)
DEV_IMAGES = $(shell docker images dev-* -q)

all: clean docker unit-test

snaps:
	@echo "Building Snaps Services"
	@mkdir -p build/docker/bin build/docker/pkg
	@docker run -i \
		-v $(abspath .):/opt/gopath/src/$(PACKAGE_NAME) \
		-v $(abspath build/docker/bin):/opt/gopath/bin \
		-v $(abspath build/docker/$(ARCH)/pkg):/opt/gopath/pkg \
		hyperledger/fabric-baseimage:$(ARCH)-$(BASEIMAGE_RELEASE) \
		go install -ldflags "$(GO_LDFLAGS)" $(PACKAGE_NAME)/cmd/snapsd

docker: snaps
	@echo "Building Snaps Image"
	@mkdir -p build/image/snaps/payload
	@cp build/docker/bin/snapsd build/image/snaps/payload/
	@cp -R cmd/config/sampleconfig build/image/snaps/payload/
	@cp -R image/* build/image/snaps/
	@docker build -t $(IMAGE_NAME) build/image/snaps

unit-test: 
	@scripts/unit.sh

clean:
	@echo "Removing Snaps Services and Image"
	@rm -rf build
	@rm -f report.xml
ifneq ($(strip $(shell docker images $(IMAGE_NAME) -q)),)
	@docker rmi $(IMAGE_NAME) -f
endif
ifneq ($(strip $(DEV_IMAGES)),)
	@docker rmi $(DEV_IMAGES) -f
endif

clean-images:
	@echo "Stopping all containers, pruning containers and images, deleting dev images"
ifneq ($(strip $(CONTAINER_IDS)),)
	@docker stop $(CONTAINER_IDS)
endif
	@docker system prune -f
ifneq ($(strip $(DEV_IMAGES)),)
	@docker rmi $(DEV_IMAGES) -f
endif
ifneq ($(strip $(shell docker images $(IMAGE_NAME) -q)),)
	@docker rmi $(IMAGE_NAME) -f
endif



