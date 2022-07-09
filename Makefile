MAKEFLAGS += --no-print-directory

APP_NAME = rfs
OUT = ./bin/${APP_NAME}

GIT_COMMIT = $(shell git rev-parse --short=8 HEAD)
GIT_BRANCH = $(shell git rev-parse --abbrev-ref HEAD)

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
LDFLAGS := -X main.commit=${GIT_COMMIT} -X main.branch=${GIT_BRANCH}

DOCKER_IMAGE = "${APP_NAME}:${GIT_BRANCH}-${GIT_COMMIT}"

export PATH := $(shell go env GOPATH)/bin:$(PATH)

.PHONY: all build test clean run format test help

all: build run

run:
	./${OUT}

# Provide the name of the Fellowship member as the target variable.
run/docker/%:
	docker run \
		--rm -it \
		--name "${APP_NAME}-$(*F)" \
		-e "PEER_NAME=$(*F)" \
		${DOCKER_IMAGE}

run/parallel:
	jq -r '.peers | .[].name' config/config.json | parallel ./${OUT} -n {} | jq

build:
	mkdir -p build
	go build -ldflags "${LDFLAGS}" -o ${OUT} main.go

build/full: go/generate go/format go/verify go/test build

build/docker:
	DOCKER_BUILDKIT=1 \
	docker build \
		--file "${PWD}/Dockerfile" \
		--tag ${DOCKER_IMAGE} \
		--build-arg APP_NAME="${APP_NAME}" \
		--build-arg LDFLAGS="${LDFLAGS}" \
		--progress=auto \
		"${PWD}"

go/generate:
	go generate ./...

go/format:
	go fmt ./...
	#@goimports ./.. >/dev/null

go/test:
	go test -v -race ./...

go/verify:
	go vet ./...

go/protobuf:
	@find ./protobuf -type f -name *.proto -printf "%f\n" | \
		xargs -I{} protoc {}\
			--proto_path=protobuf \
			--go_opt=module=github.com/Zamerykanizowana/replicated-file-system \
			--go_out=Mprotobuf/{}=:.

install/build-dependencies:
	go install golang.org/x/tools/cmd/goimports@latest
	go install golang.org/x/tools/cmd/stringer@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

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
