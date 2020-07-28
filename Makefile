# Common uses:
# - installing kubetest2: `make install INSTALL_DIR=$HOME/go/bin`
# installing a deployer: `make install-deployer-$(deployer-name) INSTALL_DIR=$HOME/go/bin`
# - cleaning up and starting over: `make clean`

# get the repo root and output path, go_container.sh respects these
REPO_ROOT:=$(shell pwd)
export REPO_ROOT
OUT_DIR=$(REPO_ROOT)/bin
INSTALL?=install
# make install will place binaries here
# the default path attempts to mimic go install
INSTALL_DIR?=$(shell $(REPO_ROOT)/hack/build/goinstalldir.sh)
# the output binary name, overridden when cross compiling
BINARY_NAME?=kubetest2
BINARY_PATH?=.
# the container cli to use e.g. docker,podman
DOCKER?=$(shell which docker || which podman || echo "docker")
export DOCKER

install:
	$(REPO_ROOT)/hack/go_container.sh go build -v -o /out/$(BINARY_NAME) $(BINARY_PATH)
	$(INSTALL) -d $(INSTALL_DIR)
	$(INSTALL) $(OUT_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)

install-deployer-%: BINARY_PATH=./kubetest2-$*
install-deployer-%: BINARY_NAME=kubetest2-$*
install-deployer-%:
	$(REPO_ROOT)/hack/go_container.sh go build -v -o /out/$(BINARY_NAME) $(BINARY_PATH)
	$(INSTALL) -d $(INSTALL_DIR)
	$(INSTALL) $(OUT_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)

quick-verify: install install-deployer-kind
	kubetest2 kind --up --down --test=exec -- kubectl get all -A

# cleans the cache volume
clean-cache:
	$(DOCKER) volume rm -f kind-build-cache >/dev/null

# cleans the output directory
clean-output:
	rm -rf $(OUT_DIR)/

# standard cleanup target
clean: clean-output clean-cache

format:
	./hack/update/gofmt.sh

lint:
	./hack/verify/lint.sh

shellcheck:
	./hack/verify/shellcheck.sh

unit:
	./hack/ci/unit.sh

verify:
	$(MAKE) -j lint shellcheck

.PHONY: install install-deployer-% quick-verify clean-cache clean-output clean verify lint shellcheck
