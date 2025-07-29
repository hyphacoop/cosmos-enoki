#!/usr/bin/make -f

PACKAGES_SIMTEST=$(shell go list ./... | grep '/simulation')
VERSION := $(shell echo $(shell git describe --tags))
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
COMMIT := $(shell git log -1 --format='%H')
LEDGER_ENABLED ?= true
SDK_PACK := $(shell go list -m github.com/cosmos/cosmos-sdk | sed  's/ /\@/g')
BINDIR ?= $(GOPATH)/bin
SIMAPP = ./app

# for dockerized protobuf tools
# DOCKER := $(shell which docker)
# HTTPS_GIT := github.com/hyphacoop/cosmos-enoki.git

export GO111MODULE = on

# don't override user values
ifeq (,$(VERSION))
  VERSION := $(shell git describe --tags --always)
  # if VERSION is empty, then populate it with branch's name and raw commit hash
  ifeq (,$(VERSION))
    VERSION := $(BRANCH)-$(COMMIT)
  endif
endif

# process build tags

build_tags = netgo
ifeq ($(LEDGER_ENABLED),true)
  ifeq ($(OS),Windows_NT)
    GCCEXE = $(shell where gcc.exe 2> NUL)
    ifeq ($(GCCEXE),)
      $(error gcc.exe not installed for ledger support, please install or set LEDGER_ENABLED=false)
    else
      build_tags += ledger
    endif
  else
    UNAME_S = $(shell uname -s)
    ifeq ($(UNAME_S),OpenBSD)
      $(warning OpenBSD detected, disabling ledger support (https://github.com/cosmos/cosmos-sdk/issues/1988))
    else
      GCC = $(shell command -v gcc 2> /dev/null)
      ifeq ($(GCC),)
        $(error gcc not installed for ledger support, please install or set LEDGER_ENABLED=false)
      else
        build_tags += ledger
      endif
    endif
  endif
endif

ifeq ($(WITH_CLEVELDB),yes)
  build_tags += gcc
endif
build_tags += $(BUILD_TAGS)
build_tags := $(strip $(build_tags))

whitespace :=
empty = $(whitespace) $(whitespace)
comma := ,
build_tags_comma_sep := $(subst $(empty),$(comma),$(build_tags))

# process linker flags

# flags '-s -w' resolves an issue with xcode 16 and signing of go binaries
# ref: https://github.com/golang/go/issues/63997
ldflags = -X github.com/cosmos/cosmos-sdk/version.Name=enoki \
		  -X github.com/cosmos/cosmos-sdk/version.AppName=enokid \
		  -X github.com/cosmos/cosmos-sdk/version.Version=$(VERSION) \
		  -X github.com/cosmos/cosmos-sdk/version.Commit=$(COMMIT) \
		  -X "github.com/cosmos/cosmos-sdk/version.BuildTags=$(build_tags_comma_sep)" \
		  -s -w
extldflags = -z noexecstack

ifeq ($(WITH_CLEVELDB),yes)
  ldflags += -X github.com/cosmos/cosmos-sdk/types.DBBackend=cleveldb
endif
ifeq ($(LINK_STATICALLY),true)
	extldflags += -Wl,-z,muldefs -static
	ldflags += -linkmode=external
endif
ldflags += -extldflags "$(extldflags)"
ldflags += $(LDFLAGS)
ldflags := $(strip $(ldflags))

BUILD_FLAGS := -tags "$(build_tags_comma_sep)" -ldflags '$(ldflags)' -trimpath

all: install lint test

build: go.sum
ifeq ($(OS),Windows_NT)
	$(error wasmd server not supported. Use "make build-windows-client" for client)
	exit 1
else
	go build -mod=readonly $(BUILD_FLAGS) -o build/enokid ./cmd/enokid
	# go build $(BUILD_FLAGS) -o build/enokid ./cmd/enokid
endif

install: go.sum
	go install -mod=readonly $(BUILD_FLAGS) ./cmd/enokid
	# go install $(BUILD_FLAGS) ./cmd/enokid

local-image:
	docker build -t enoki:local .

########################################
### Tools & dependencies

go-mod-cache: go.sum
	@echo "--> Download go modules to local cache"
	@go mod download

go.sum: go.mod
	@echo "--> Ensure dependencies have not been modified"
	@go mod verify

draw-deps:
	@# requires brew install graphviz or apt-get install graphviz
	go install github.com/RobotsAndPencils/goviz@latest
	@goviz -i ./cmd/enokid -d 2 | dot -Tpng -o dependency-graph.png

clean:
	rm -rf snapcraft-local.yaml build/

distclean: clean
	rm -rf vendor/

########################################
### Testing

test: test-unit
test-all: test-race test-cover

test-unit:
	@VERSION=$(VERSION) go test -mod=readonly -tags='ledger test_ledger_mock' ./...

test-race:
	@VERSION=$(VERSION) go test -mod=readonly -race -tags='ledger test_ledger_mock' ./...

test-cover:
	@go test -mod=readonly -timeout 30m -race -coverprofile=coverage.txt -covermode=atomic -tags='ledger test_ledger_mock' ./...

benchmark:
	@go test -mod=readonly -bench=. ./...

###############################################################################
###                                Linting                                  ###
###############################################################################

format-tools:
	go install mvdan.cc/gofumpt@v0.4.0
	go install github.com/client9/misspell/cmd/misspell@v0.3.4
	go install github.com/daixiang0/gci@v0.11.2

lint: format-tools
	golangci-lint run --tests=false
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "./tests/system/vendor*" -not -path "*.git*" -not -path "*_test.go" | xargs gofumpt -d

format: format-tools
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "./tests/system/vendor*" -not -path "*.git*" -not -path "./client/lcd/statik/statik.go" | xargs gofumpt -w
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "./tests/system/vendor*" -not -path "*.git*" -not -path "./client/lcd/statik/statik.go" | xargs misspell -w
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "./tests/system/vendor*" -not -path "*.git*" -not -path "./client/lcd/statik/statik.go" | xargs gci write --skip-generated -s standard -s default -s "prefix(cosmossdk.io)" -s "prefix(github.com/cosmos/cosmos-sdk)" -s "prefix(github.com/CosmWasm/wasmd)" --custom-order

mod-tidy:
	go mod tidy
	cd interchaintest && go mod tidy

.PHONY: format-tools lint format mod-tidy

###############################################################################
###                                     e2e                                 ###
###############################################################################

ictest-basic:
	@echo "Running basic e2e test"
	@cd interchaintest && go test -race -v -run TestBasicChain .

ictest-ibc:
	@echo "Running IBC e2e test"
	@cd interchaintest && go test -race -v -run TestIBCBasic .

ictest-wasm:
	@echo "Running cosmwasm e2e test"
	@cd interchaintest && go test -race -v -run TestCosmWasmIntegration .

ictest-packetforward:
	@echo "Running packet forward middleware e2e test"
	@cd interchaintest && go test -race -v -run TestPacketForwardMiddleware .

ictest-ratelimit:
	@echo "Running rate limit e2e test"
	@cd interchaintest && go test -race -v -run TestIBCRateLimit .

ictest-feemarket:
	@echo "Running feemarket e2e test"
	@cd interchaintest && go test -race -v -count=1 -run TestFeemarket .

ictest-clean:
	@echo "Cleaning up interchaintest cache"
	@cd interchaintest && go clean -testcache

ictest-full: ictest-clean ictest-basic ictest-ibc ictest-wasm ictest-packetforward ictest-ratelimit ictest-feemarket

.PHONY: ictest-basic ictest-ibc ictest-wasm ictest-packetforward ictest-ratelimit ictest-clean ictest-feemarket ictest-full

###############################################################################
###                              image testnet                              ###
###############################################################################


get-localic:
	@echo "Installing local-interchain"
	git clone --depth 1 --branch v10.0.0 https://github.com/cosmos/interchaintest.git interchaintest-downloader
	cd interchaintest-downloader/local-interchain && make install
	@sleep 0.1
	@echo ✅ local-interchain installed $(shell which local-ic)
	rm -rf interchaintest-downloader

is-localic-installed:
ifeq (,$(shell which local-ic))
	make get-localic
else
	@echo ✅ local-interchain already installed $(shell which local-ic)
endif

setup-ic-testnet: mod-tidy is-localic-installed local-image

ic-testnet: setup-ic-testnet
	local-ic start enoki

ic-testnet-ibc: setup-ic-testnet
	local-ic start enoki-ibc

ic-testnet-gaia: setup-ic-testnet
	local-ic start enoki-gaia

.PHONY: get-localic is-localic-installed setup-ic-testnet ic-testnet ic-testnet-ibc ic-testnet-gaia

###############################################################################
###                              shell testnet                              ###
###############################################################################

sh-testnet: mod-tidy
	HOME_DIR=".enoki" CHAIN_ID="test-enoki-1" BLOCK_TIME="2000ms" CLEAN=true sh scripts/test_node.sh

.PHONY: sh-testnet

###############################################################################
###                                     help                                ###
###############################################################################

help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Available targets:"
	@echo "  install             : Install the binary"
	@echo "  local-image         : Install the docker image"
	@echo "  ic-testnet          : Local testnet"
	@echo "  ic-testnet-ibc      : Local testnet with IBC channel to second Enoki devnet"
	@echo "  ic-testnet-gaia     : Local testnet with IBC channel to Gaia devnet"
	@echo "  sh-testnet          : Shell local testnet"
	@echo "  ictest-full         : Run all e2e tests"

.PHONY: help
