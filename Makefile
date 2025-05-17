# HELP
# This will output the help for each task
.PHONY: help build test install-go-test-coverage check-coverage

# Tasks
help: ## This help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help

GOBIN ?= $$(go env GOPATH)/bin

.PHONY: build
build: ## Build the project
	go build -o bin/valet main.go

.PHONY: clean
clean: ## Clean the project
	rm -rf bin
	rm -rf valet

.PHONY: test
test: ## Run the tests
	go test ./... -coverprofile=./cover.out -covermode=atomic -coverpkg=./...
	go tool cover -html=./cover.out -o ./cover.html

.PHONY: install-go-test-coverage
install-go-test-coverage: ## Install go-test-coverage
	go install github.com/vladopajic/go-test-coverage/v2@latest

.PHONY: check-coverage
check-coverage: install-go-test-coverage ## Check the coverage
	go test ./... -coverprofile=./cover.out -covermode=atomic -coverpkg=./...
	${GOBIN}/go-test-coverage --config=./.testcoverage.yml