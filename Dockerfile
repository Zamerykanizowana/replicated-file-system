FROM golang:1.19-bullseye as go-dependencies-cache

WORKDIR /src

COPY go.mod go.sum /src/

RUN apt-get install git bash

RUN go mod download

FROM go-dependencies-cache as builder

ARG LDFLAGS

WORKDIR /src

COPY . .

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "${LDFLAGS}" -o "/artifacts/rfs" "${PWD}"

FROM debian:bullseye

COPY --from=builder "/artifacts/rfs" /bin/rfs

RUN apt-get update
RUN apt-get install -y bash fuse coreutils

ENV USER="/home/rfs"

ENTRYPOINT ["/bin/rfs"]
