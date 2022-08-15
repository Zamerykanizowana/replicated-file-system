MAKEFLAGS += --no-print-directory

PEER ?= Aragorn
CONFIGPATH ?= config/config.json

APP_NAME = rfs
OUT_DIR = bin
OUT = ./${OUT_DIR}/${APP_NAME}

GIT_COMMIT = $(shell git rev-parse --short=8 HEAD)
GIT_BRANCH = $(shell git rev-parse --abbrev-ref HEAD)

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
LDFLAGS := -X main.commit=${GIT_COMMIT} -X main.branch=${GIT_BRANCH}

DOCKER_IMAGE = "${APP_NAME}:${GIT_BRANCH}-${GIT_COMMIT}"

CERT_AGE := -days 365
CERT_KEY_SIZE = 4096
CERT_PATH = p2p/cert

export PATH := $(shell go env GOPATH)/bin:$(PATH)

.PHONY: all build test clean run format test help cert

all: build run

run:
	./${OUT} -n ${PEER} -c ${CONFIGPATH}

# Provide the name of the Fellowship member as the target variable.
run/docker/%:
	docker run \
		--rm -it \
		--name "${APP_NAME}-$(*F)" \
		-e "PEER_NAME=$(*F)" \
		${DOCKER_IMAGE}

build: cert/create
	mkdir -p ${OUT_DIR}
	go build -ldflags "${LDFLAGS}" -o ${OUT} main.go

build/full: go/format go/verify go/test build

# build/docker/%s should build a ready to run peer with a specified name through PEER env.
build/docker: cert/create
	DOCKER_BUILDKIT=1 \
	docker build \
		--file "${PWD}/Dockerfile" \
		--tag ${DOCKER_IMAGE} \
		--build-arg APP_NAME="${APP_NAME}" \
		--build-arg LDFLAGS="${LDFLAGS} -w -s -X main.peer=${PEER}" \
		--progress=auto \
		"${PWD}"

cert/generate-ca:
	cd ${CERT_PATH} && \
	openssl req -nodes -x509 ${CERT_AGE} \
		-subj "/C=PL/ST=Wielkopolskie/L=Poznan/O=Poznan University Of Technology/OU=Faculty Of Computing And Telecommunications/CN=cat.put.poznan.pl" \
		-newkey rsa:${CERT_KEY_SIZE} \
		-out ca.crt \
		-keyout ca.key

cert/create: cert/generate-peer-key cert/generate-peer-csr cert/sign-csr
	@printf '\n${YELLOW}Generated peer certificate, which will be embedded and used for TLS${RESET}\n'
	@printf '${YELLOW}Make sure the same CA (certificate authority) key is used to sign all peer certificates${RESET}\n'

cert/generate-peer-key:
	cd ${CERT_PATH} && \
	openssl genrsa -out peer.key ${CERT_KEY_SIZE}

cert/generate-peer-csr:
	cd ${CERT_PATH} && \
	openssl req -new ${CERT_AGE} \
		-subj "/C=PL/CN=${PEER}" \
		-key peer.key \
		-out peer.csr

cert/sign-csr:
	cd ${CERT_PATH} && \
	openssl x509 -req ${CERT_AGE} -sha256 -CAcreateserial \
		-in peer.csr \
		-CA ca.crt \
		-CAkey ca.key \
		-out peer.crt

go/format:
	go fmt ./...

go/test:
	go test -v -race ./...

go/test-e2e:
	go test -tags e2e -v ./p2p

go/verify:
	go vet ./...

go/protobuf:
	@find ./protobuf -type f -name *.proto -printf "%f\n" | \
		xargs -I{} protoc {}\
			--proto_path=protobuf \
			--go_opt=module=github.com/Zamerykanizowana/replicated-file-system \
			--go_out=Mprotobuf/{}=:.

install/build-dependencies:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

clean:
	go clean
	rm -f ${OUT} ca.crt ca.key ca.srl ${CERT_PATH}/peer.crt ${CERT_PATH}/peer.key ${CERT_PATH}/peer.csr

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
