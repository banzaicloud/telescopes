EXECUTABLE ?= spot-recommender
IMAGE ?= banzaicloud/$(EXECUTABLE)
TAG ?= dev-$(shell git log -1 --pretty=format:"%h")

LD_FLAGS = -X "main.version=$(TAG)"
PACKAGES = $(shell go list ./... | grep -v /vendor/)

.PHONY: _no-target-specified
_no-target-specified:
	$(error Please specify the target to make - `make list` shows targets.)

.PHONY: list
list:
	@$(MAKE) -pRrn : -f $(MAKEFILE_LIST) 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | sort

all: clean deps fmt vet docker push

clean:
	go clean -i ./...

deps:
	go get ./...

fmt:
	go fmt $(PACKAGES)

vet:
	go vet $(PACKAGES)

docker:
	docker build --rm -t $(IMAGE):$(TAG) .

push:
	docker push $(IMAGE):$(TAG)

run-dev:
	. .env
	go run $(wildcard *.go)
