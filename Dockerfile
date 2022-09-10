FROM golang:1.19-alpine as go-dependencies-cache

WORKDIR /src

COPY go.mod go.sum /src/

RUN apk add install git bash

RUN go mod download

FROM go-dependencies-cache as builder

ARG LDFLAGS

WORKDIR /src

COPY . .

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "${LDFLAGS}" -o "/artifacts/rfs" "${PWD}"

FROM scratch

ENV USER="/home/rfs"

COPY --from=builder "/artifacts/rfs" /bin/rfs

ENTRYPOINT ["/bin/rfs"]
