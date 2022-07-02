FROM golang:1.18.2-alpine3.14 as go-dependencies-cache

WORKDIR /src

COPY go.mod go.sum /src/

RUN apk add git bash

RUN go mod download

FROM go-dependencies-cache as builder

ARG APP_NAME
ARG LDFLAGS

WORKDIR /src

COPY . .

RUN go build -ldflags "${LDFLAGS}" -o "/artifacts/${APP_NAME}" "${PWD}"


FROM alpine:3.16.0

ARG APP_NAME
ENV APP=${APP_NAME}

COPY --from=builder "/artifacts/${APP_NAME}" /bin

ENTRYPOINT ["sh", "-c", "$APP p2p -n $PEER_NAME"]
