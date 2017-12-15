.DEFAULT_GOAL := help
.PHONY: help

OS := $(shell uname -s)
GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

build: ## Builds binary package
	go build .

deps: ## Install dependencies required for building
	which dep > /dev/null || go get -u github.com/golang/dep/cmd/dep

help: ## Generates this help message
	@grep -E '^[0-9a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "�33[36m%-30s�33[0m %sn", $$1, $$2}'

list:
	@$(MAKE) -pRrn : -f $(MAKEFILE_LIST) 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | sort

fmt:
	@gofmt -w ${GOFILES_NOVENDOR}
