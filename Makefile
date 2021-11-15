
OS = $(shell uname -s)

# Project variables
BUILD_PACKAGE ?= ./cmd/telescopes
BINARY_NAME ?= telescopes
DOCKER_IMAGE = banzaicloud/telescopes

# Docker Push configuration
GCR_HOST ?= us.gcr.io
GCR_PROJECT_ID ?= platform-205701
GCR_IMAGE_REPOSITORY ?= harness/banzai-telescopes

# Build variables
BUILD_DIR ?= build
VERSION ?= $(shell git symbolic-ref -q --short HEAD | sed 's/[^a-zA-Z0-9]/-/g')
COMMIT_HASH ?= $(shell git rev-parse --short HEAD 2>/dev/null)
BUILD_DATE ?= $(shell date +%FT%T%z)
BRANCH ?= $(shell git symbolic-ref -q --short HEAD)
LDFLAGS := -X main.version=${VERSION} -X main.commitHash=${COMMIT_HASH} -X main.buildDate=${BUILD_DATE} -X main.branch=${BRANCH}
export CGO_ENABLED ?= 0
ifeq (${VERBOSE}, 1)
	GOARGS += -v
endif

CLOUDINFO_VERSION = 0.5.0

# Docker variables
DOCKER_TAG ?= ${VERSION}
GCR_IMAGE_LOCATION = ${GCR_HOST}/${GCR_PROJECT_ID}/${GCR_IMAGE_REPOSITORY}:${DOCKER_TAG}

GOTESTSUM_VERSION = 0.3.4
GOLANGCI_VERSION = 1.16.0
MISSPELL_VERSION = 0.3.4
JQ_VERSION = 1.5
LICENSEI_VERSION = 0.1.0
OPENAPI_GENERATOR_VERSION = PR1869

GOLANG_VERSION = 1.12

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

swagger:
	swagger generate spec -m -w ./cmd/telescopes -o $(SWAGGER_REC_TMP_FILE)
	swagger2openapi -y $(SWAGGER_REC_TMP_FILE) > $(SWAGGER_REC_FILE)

generate-client:
	swagger generate client -f $(SWAGGER_REC_TMP_FILE) -A recommender -t pkg/recommender-client/

.PHONY: generate-cloudinfo-client
generate-cloudinfo-client: ## Generate client from Cloudinfo OpenAPI spec
	if ! test -f cloudinfo-openapi.yaml; then echo "Please populate ${PWD}/cloudinfo-openapi.yaml file" && touch cloudinfo-openapi.yaml && exit 1; fi
	rm -rf .gen/cloudinfo
	docker run --rm -v ${PWD}:/local banzaicloud/openapi-generator-cli:${OPENAPI_GENERATOR_VERSION} generate \
	--additional-properties packageName=cloudinfo \
	--additional-properties withGoCodegenComment=true \
	-i /local/cloudinfo-openapi.yaml \
	-g go \
	-o /local/.gen/cloudinfo
	rm cloudinfo-openapi.yaml .gen/cloudinfo/.travis.yml .gen/cloudinfo/git_push.sh
