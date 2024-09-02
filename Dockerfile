FROM golang:alpine AS builder

RUN apk update && apk add --no-cache git bash

WORKDIR $GOPATH/src/quantifyearth/reclaimer
COPY . .

RUN go get -d -v

RUN go build -o /go/bin/reclaimer
WORKDIR /go/bin
