MAKEFLAGS += --no-print-directory

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
LDFLAGS := \
	-X main.commit=$(shell git rev-parse --short=8 HEAD) \
	-X main.branch=$(shell git rev-parse --abbrev-ref HEAD)

OUT=build/rfs

export PATH := $(shell go env GOPATH)/bin:$(PATH)

.PHONY: all build test clean run format test help

all: build run

run:
	./${OUT}

run/parallel:
	cat config/config.json | jq -r '.peers | .[].name' | parallel ./build/rfs -n {} | jq

build: go/generate go/format
	mkdir -p build
	go build -ldflags "${LDFLAGS}" -o ${OUT} main.go

go/generate:
	go generate ./...

go/format:
	go fmt ./...
	#@goimports ./.. >/dev/null

go/test:
	go test -v -race ./...

go/protobuf:
	@find ./protobuf -type f -name *.proto -printf "%f\n" | \
		xargs -I{} protoc {}\
			--proto_path=protobuf \
			--go_opt=module=github.com/Zamerykanizowana/replicated-file-system \
			--go_out=Mprotobuf/{}=:.

install/build-dependencies:
	go install golang.org/x/tools/cmd/goimports@latest
	go install golang.org/x/tools/cmd/stringer@latest

clean:
	go clean
	rm ${OUT}

help:
	@printf '\nUsage:\n  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}\n\nTargets:\n'
	@LC_ALL=C $(MAKE) -pRrq -f $(lastword $(MAKEFILE_LIST)) : 2>/dev/null | \
		awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | \
		sort | \
		egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | \
		xargs -I {} echo '  ${CYAN}{}${RESET}'
	@echo ''


GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
CYAN   := $(shell tput -Txterm setaf 6)
RESET  := $(shell tput -Txterm sgr0)
