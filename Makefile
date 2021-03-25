# Copyright 2020 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Common uses:
# - installing kubetest2: `make install INSTALL_DIR=$HOME/go/bin`
# installing a deployer: `make install-deployer-$(deployer-name) INSTALL_DIR=$HOME/go/bin`
# - cleaning up and starting over: `make clean`

# get the repo root and output path
REPO_ROOT:=$(shell pwd)
export REPO_ROOT
OUT_DIR?=$(REPO_ROOT)/bin
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
# disable CGO by default for static binaries
CGO_ENABLED=0
export PATH GOROOT GO111MODULE CGO_ENABLED
# work around broken PATH export
SPACE:=$(subst ,, )
SHELL:=env PATH=$(subst $(SPACE),\$(SPACE),$(PATH)) $(SHELL)
# ==============================================================================
# flags for reproducible go builds
BUILD_FLAGS?=-trimpath -ldflags="-buildid="

build-all:
	go build -v $(BUILD_FLAGS) ./...

install:
	go build -v $(BUILD_FLAGS) -o $(OUT_DIR)/$(BINARY_NAME) $(BINARY_PATH)
	$(INSTALL) -d $(INSTALL_DIR)
	$(INSTALL) $(OUT_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)

install-deployer-%: BINARY_PATH=./kubetest2-$*
install-deployer-%: BINARY_NAME=kubetest2-$*
install-deployer-%:
	go build -v $(BUILD_FLAGS) -o $(OUT_DIR)/$(BINARY_NAME) $(BINARY_PATH)
	$(INSTALL) -d $(INSTALL_DIR)
	$(INSTALL) $(OUT_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)

install-tester-%: BINARY_PATH=./kubetest2-tester-$*
install-tester-%: BINARY_NAME=kubetest2-tester-$*
install-tester-%:
	go build $(BUILD_FLAGS) -v $(BUILD_OPTS) -o $(OUT_DIR)/$(BINARY_NAME) $(BINARY_PATH)
	$(INSTALL) -d $(INSTALL_DIR)
	$(INSTALL) $(OUT_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)

install-all: TESTERS := $(wildcard kubetest2-tester-*)
install-all: DEPLOYERS := $(filter-out $(TESTERS), $(wildcard kubetest2-*))
install-all: TESTER_TARGETS := $(subst kubetest2-tester-,install-tester-, $(TESTERS))
install-all: DEPLOYER_TARGETS := $(subst kubetest2-,install-deployer-, $(DEPLOYERS))
install-all: install
	$(MAKE) -j $(DEPLOYER_TARGETS) $(TESTER_TARGETS)

quick-verify: install install-deployer-kind install-tester-exec
	kubetest2 kind --up --down --test=exec -- kubectl get all -A

ci-binaries:
	./hack/build/ci-binaries.sh

push-ci-binaries:
	./hack/ci/push-binaries/push-binaries.sh

# cleans the output directory
clean-output:
	rm -rf $(OUT_DIR)/

# standard cleanup target
clean: clean-output

fix:
	./hack/update/gofmt.sh
	./hack/update/tidy.sh

boilerplate:
	./hack/verify/boilerplate.py

lint:
	./hack/verify/lint.sh

shellcheck:
	./hack/verify/shellcheck.sh

tidy:
	./hack/verify/tidy.sh

unit:
	./hack/ci/unit.sh

verify:
	$(MAKE) -j lint shellcheck unit tidy boilerplate

.PHONY: build-all install install-deployer-% install-tester-% install-all ci-binaries push-ci-binaries quick-verify clean-output clean verify lint shellcheck
