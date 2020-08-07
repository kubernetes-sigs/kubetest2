# Common uses:
# - installing kubetest2: `make install INSTALL_DIR=$HOME/go/bin`
# installing a deployer: `make install-deployer-$(deployer-name) INSTALL_DIR=$HOME/go/bin`
# - cleaning up and starting over: `make clean`

# get the repo root and output path
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
# ========================= Setup Go With Gimme ================================
# go version to use for build etc.
# setup correct go version with gimme
PATH:=$(shell . hack/build/setup-go.sh && echo "$${PATH}")
# go1.9+ can autodetect GOROOT, but if some other tool sets it ...
GOROOT:=
# enable modules
GO111MODULE=on
export PATH GOROOT GO111MODULE
# work around broken PATH export
SHELL:=env PATH=$(PATH) $(SHELL)
# ==============================================================================

build-all:
	go build -v ./...

install:
	go build -v -o $(OUT_DIR)/$(BINARY_NAME) $(BINARY_PATH)
	$(INSTALL) -d $(INSTALL_DIR)
	$(INSTALL) $(OUT_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)

install-deployer-%: BINARY_PATH=./kubetest2-$*
install-deployer-%: BINARY_NAME=kubetest2-$*
install-deployer-%:
	go build -v -o $(OUT_DIR)/$(BINARY_NAME) $(BINARY_PATH)
	$(INSTALL) -d $(INSTALL_DIR)
	$(INSTALL) $(OUT_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)

install-tester-%: BINARY_PATH=./kubetest2-tester-$*
install-tester-%: BINARY_NAME=kubetest2-tester-$*
install-tester-%:
	go build -v -o $(OUT_DIR)/$(BINARY_NAME) $(BINARY_PATH)
	$(INSTALL) -d $(INSTALL_DIR)
	$(INSTALL) $(OUT_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)

quick-verify: install install-deployer-kind install-tester-exec
	kubetest2 kind --up --down --test=exec -- kubectl get all -A

# cleans the output directory
clean-output:
	rm -rf $(OUT_DIR)/

# standard cleanup target
clean: clean-output

fix:
	./hack/update/gofmt.sh
	./hack/update/tidy.sh

lint:
	./hack/verify/lint.sh

shellcheck:
	./hack/verify/shellcheck.sh

tidy:
	./hack/verify/tidy.sh

unit:
	./hack/ci/unit.sh

verify:
	$(MAKE) -j lint shellcheck unit tidy

.PHONY: build-all install install-deployer-% install-tester-% quick-verify clean-output clean verify lint shellcheck
