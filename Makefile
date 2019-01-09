
SHELL = /bin/bash
OS = $(shell uname -s)

# Project variables
PACKAGE = github.com/banzaicloud/telescopes
BINARY_NAME = telescopes
#OPENAPI_DESCRIPTOR = docs/openapi/pipeline.yaml

# Build variables
BUILD_DIR ?= build
BUILD_PACKAGE = ${PACKAGE}/cmd/telescopes
VERSION ?= $(shell git rev-parse --abbrev-ref HEAD)
COMMIT_HASH ?= $(shell git rev-parse --short HEAD 2>/dev/null)
BUILD_DATE ?= $(shell date +%FT%T%z)
LDFLAGS += -X main.Version=${VERSION} -X main.CommitHash=${COMMIT_HASH} -X main.BuildDate=${BUILD_DATE}
export CGO_ENABLED ?= 0
ifeq (${VERBOSE}, 1)
	GOARGS += -v
endif

DEP_VERSION = 0.5.0
GOLANGCI_VERSION = 1.10.2
GOTESTSUM_VERSION = 0.3.2
MISSPELL_VERSION = 0.3.4
JQ_VERSION = 1.5
LICENSEI_VERSION = 0.0.7
OPENAPI_GENERATOR_VERSION = 3.3.0

GOLANG_VERSION = 1.11

GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./client/*")

include main-targets.mk

SWAGGER_REC_TMP_FILE = ./api/openapi-spec/recommender.json
SWAGGER_REC_FILE = ./api/openapi-spec/recommender.yaml


deps-swagger:
ifeq ($(shell which swagger),)
	go get -u github.com/go-swagger/go-swagger/cmd/swagger
endif
ifeq ($(shell which swagger2openapi),)
	npm install -g swagger2openapi
endif

deps: deps-swagger
	go get ./...


docker:
	docker build --rm -t $(IMAGE):$(TAG) .

push:
	docker push $(IMAGE):$(TAG)


swagger:
	swagger generate spec -m -b ./cmd/telescopes -o $(SWAGGER_REC_TMP_FILE)
	swagger2openapi -y $(SWAGGER_REC_TMP_FILE) > $(SWAGGER_REC_FILE)

