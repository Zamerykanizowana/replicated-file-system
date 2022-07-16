FROM golang:1.18.2-alpine3.14 as go-dependencies-cache

WORKDIR /src

COPY go.mod go.sum /src/

RUN apk add git bash

RUN go mod download

FROM go-dependencies-cache as builder

ARG USER

RUN addgroup -S ${USER}
RUN adduser -S ${USER} -G ${USER}

ARG LDFLAGS

WORKDIR /src

COPY . .

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "${LDFLAGS}" -o "/artifacts/rfs" "${PWD}"

FROM scratch

ARG USER

COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

COPY --from=builder "/artifacts/rfs" /bin/rfs

USER ${USER}:${USER}

ENTRYPOINT ["/bin/rfs"]
