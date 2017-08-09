#
# Copyright SecureKey Technologies Inc. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

CONTAINER_IDS = $(shell docker ps -a -q)
DEV_IMAGES = $(shell docker images dev-* -q)

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

integration-test: depend
	@scripts/integration.sh

all: checks unit-test integration-test

clean-images:
	@echo "Stopping all containers, pruning containers and images, deleting dev images"
ifneq ($(strip $(CONTAINER_IDS)),)
	@docker stop $(CONTAINER_IDS)
endif
	@docker system prune -f
ifneq ($(strip $(DEV_IMAGES)),)
	@docker rmi $(DEV_IMAGES) -f
endif
