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
CERT_PATH ?= ${PWD}/connection/cert
DOCKER_CERT_PATH ?= /home/rfs/cert

export PATH := $(shell go env GOPATH)/bin:$(PATH)

.PHONY: all build test clean run format test help cert

all: help

run: ## Run the binary under OUT as peer=PEER and config=CONFIGPATH.
	./${OUT} -n ${PEER} -c ${CONFIGPATH}

run/docker: cert/create ## Run docker image for peer=${PEER} mounting cert under ${CERT_PATH}:${DOCKER_CERT_PATH}.
	docker run \
		--rm -it \
		--privileged --cap-add SYS_ADMIN \
		--name "${APP_NAME}-${PEER}" \
		-v ${CERT_PATH}:${DOCKER_CERT_PATH} \
		${DOCKER_IMAGE} \
			--ca ${DOCKER_CERT_PATH}/ca.crt \
			--crt ${DOCKER_CERT_PATH}/${PEER}.crt \
			--key ${DOCKER_CERT_PATH}/${PEER}.key

build: cert/create ## Build rfs binary along with TLS certificate.
	@mkdir -p ${OUT_DIR}
	go build -ldflags "${LDFLAGS}" -o ${OUT} main.go

build/full: format verify test build ## Format, verify and test before building the binary.

build/docker: ## Build docker image for peer=PEER along with TLS certificate.
	DOCKER_BUILDKIT=1 \
	docker build \
		--file "${PWD}/Dockerfile" \
		--tag ${DOCKER_IMAGE} \
		--build-arg APP_NAME="${APP_NAME}" \
		--build-arg LDFLAGS="${LDFLAGS} -w -s -X main.peer=${PEER}" \
		--progress=auto \
		"${PWD}"

cert/generate-ca: ## Generate Certificate Authority using RSA under CERT_PATH with age=CERT_AGE and size=CERT_KEY_SIZE.
	cd ${CERT_PATH} && \
	openssl req -nodes -x509 ${CERT_AGE} \
		-subj "/C=PL/ST=Wielkopolskie/L=Poznan/O=Poznan University Of Technology/OU=Faculty Of Computing And Telecommunications/CN=cat.put.poznan.pl" \
		-newkey rsa:${CERT_KEY_SIZE} \
		-out ca.crt \
		-keyout ca.key

cert/create: cert/generate-peer-key cert/generate-peer-csr cert/sign-csr ## Create peer certificate.
	@printf '\n${YELLOW}Generated ${PEER} certificate, which will be used for TLS${RESET}\n'
	@printf '${YELLOW}Make sure the same CA (certificate authority) key is used to sign all peer certificates${RESET}\n'

cert/generate-peer-key: ## Generate private key using RSA for peer under CERT_PATH with size=CERT_KEY_SIZE.
	cd ${CERT_PATH} && \
	openssl genrsa -out ${PEER}.key ${CERT_KEY_SIZE}

cert/generate-peer-csr: ## Generate certificate signing request under CERT_PATH with age=CERT_AGE.
	cd ${CERT_PATH} && \
	openssl req -new ${CERT_AGE} \
		-subj "/C=PL/CN=${PEER}" \
		-key ${PEER}.key \
		-out ${PEER}.csr

cert/sign-csr: ## Sign certificate request under CERT_PATH with age=CERT_AGE.
	cd ${CERT_PATH} && \
	openssl x509 -req ${CERT_AGE} -sha256 -CAcreateserial \
		-in ${PEER}.csr \
		-CA ca.crt \
		-CAkey ca.key \
		-out ${PEER}.crt

format: ## Format go files.
	go fmt ./...

test: test/unit test/e2e ## Run unit all tests.

test/unit: ## Run unit tests.
	go test -v ./...

test/e2e: ## Run end to end tests.
	go test -v -tags e2e ./test

verify: ## Verify files in project.
	go vet ./...

protobuf: ## Regenerate protobuf autogenerated go definitions.
	@find ./protobuf -type f -name *.proto -printf "%f\n" | \
		xargs -I{} protoc {}\
			--proto_path=protobuf \
			--go_opt=module=github.com/Zamerykanizowana/replicated-file-system \
			--go_out=Mprotobuf/{}=:.

install/dev-dependencies: ## Install development dependencies: protoc and protoc-gen-go.
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

clean: ## Remove all certificates and built binaries.
	go clean
	rm -f ${OUT} ca.crt ca.key ca.srl ${CERT_PATH}/peer.crt ${CERT_PATH}/peer.key ${CERT_PATH}/peer.csr

help: ## Print this help message.
	@printf '\nUsage:\n  	${YELLOW}make${RESET} ${GREEN}<target>${RESET}\n\nTargets:\n'
	@grep -P '^[a-zA-Z_/-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort |\
 		awk 'BEGIN {FS = ":.*?## "}; {printf "	\033[36m%-30s\033[0m  %s\n", $$1, $$2}'

GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
CYAN   := $(shell tput -Txterm setaf 6)
RESET  := $(shell tput -Txterm sgr0)
