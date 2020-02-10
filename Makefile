export GOBIN ?= $(shell pwd)/bin

GOLINT = $(GOBIN)/golint

GO_FILES := $(shell \
	find . '(' -path './go/.*' -o -path './vendor' ')' -prune \
	-o -name '*.go' -print | cut -b3-)

.PHONY: build
build:
	go build github.com/henrywu2019/athenasql/go

.PHONY: install
install:
	go mod download

.PHONY: dependencies
dependencies:
	go mod download

.PHONY: checklic
checklic:
	@echo "Checking for license headers..."
	@cd scripts && ./checklic.sh | tee -a ../lint.log

.PHONY: test
test:
	go test github.com/henrywu2019/athenasql/go

.PHONY: cover
cover:
	go test -race -coverprofile=cover.out -coverpkg=go/... go/...
	go tool cover -html=cover.out -o cover.html

$(GOLINT):
	go install golang.org/x/lint/golint

.PHONY: lint
lint: $(GOLINT)
	@rm -rf lint.log
	@echo "Checking formatting..."
	@gofmt -d -s $(GO_FILES) 2>&1 | tee lint.log
	@echo "Checking vet..."
	@go vet go/... 2>&1 | tee -a lint.log
	@echo "Checking lint..."
	@$(GOLINT) go/... | tee -a lint.log
	@echo "Checking for unresolved FIXMEs..."
	@git grep -i fixme | grep -v -e vendor -e Makefile -e .md | tee -a lint.log
	@[ ! -s lint.log ]