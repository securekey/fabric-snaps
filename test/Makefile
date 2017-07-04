#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

# This Makefile assumes a working Golang and Docker setup
#
# The following targets are supported:
#
# docker: Builds the snapsd and snap2 docker images where:
# - snapsd: contains the peer and the public snaps process
# - snap2: contains a single snap: snap2
# clean: Remove build files and images

PROJECT_NAME = securekey/fabric-snaps
PACKAGE_NAME = github.com/$(PROJECT_NAME)
ARCH = $(shell uname -m)
IMAGE1_NAME = snapsd
IMAGE2_NAME = snap2
BASEIMAGE_RELEASE = 0.3.1
GO_LDFLAGS = -linkmode external -extldflags '-static -lpthread'
CONTAINER_IDS = $(shell docker ps -a -q)
DEV_IMAGES = $(shell docker images dev-* -q)

all: clean docker

docker: snapsd-docker snap2-docker

snapsd:
	@echo "Building snapsd"
	@mkdir -p build/docker/snapsd/bin build/docker/snapsd/pkg
	@docker run -i \
		-v $(abspath ../):/opt/gopath/src/$(PACKAGE_NAME) \
		-v $(abspath build/docker/snapsd/bin):/opt/gopath/bin \
		-v $(abspath build/docker/snapsd/$(ARCH)/pkg):/opt/gopath/pkg \
		hyperledger/fabric-baseimage:$(ARCH)-$(BASEIMAGE_RELEASE) \
		go install -ldflags "$(GO_LDFLAGS)" $(PACKAGE_NAME)/cmd/snapsd

snapsd-docker: snapsd
	@echo "Building snapsd Docker"
	@mkdir -p build/image/snapsd/payload
	@cp build/docker/snapsd/bin/snapsd build/image/snapsd/payload/
	@cp -R snaps/snapsd/config build/image/snapsd/payload/
	@cp -R snaps/snapsd/image/* build/image/snapsd/
	@docker build -t $(IMAGE1_NAME) build/image/snapsd

snap2:
	@echo "Building Snap2"
	@mkdir -p build/docker/snap2/bin build/docker/snap2/pkg
	@docker run -i \
		-v $(abspath ../):/opt/gopath/src/$(PACKAGE_NAME) \
		-v $(abspath build/docker/snap2/bin):/opt/gopath/bin \
		-v $(abspath build/docker/snap2/$(ARCH)/pkg):/opt/gopath/pkg \
		hyperledger/fabric-baseimage:$(ARCH)-$(BASEIMAGE_RELEASE) \
		go install -ldflags "$(GO_LDFLAGS)" $(PACKAGE_NAME)/test/snaps/snap2

snap2-docker: snap2
	@echo "Building Snap2 Docker"
	@mkdir -p build/image/snap2/payload
	@cp build/docker/snap2/bin/snap2 build/image/snap2/payload/
	@cp -R snaps/snap2/config build/image/snap2/payload/
	@cp -R snaps/snap2/image/* build/image/snap2/
	@docker build -t $(IMAGE2_NAME) build/image/snap2

clean:
	@echo "Removing Snaps Services and Image"
	@rm -rf build
ifneq ($(strip $(shell docker images $(IMAGE1_NAME) -q)),)
	@docker rmi $(IMAGE1_NAME) -f
endif
ifneq ($(strip $(shell docker images $(IMAGE2_NAME) -q)),)
	@docker rmi $(IMAGE2_NAME) -f
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
ifneq ($(strip $(shell docker images $(IMAGE1_NAME) -q)),)
	@docker rmi $(IMAGE1_NAME) -f
endif
ifneq ($(strip $(shell docker images $(IMAGE2_NAME) -q)),)
	@docker rmi $(IMAGE2_NAME) -f
endif
