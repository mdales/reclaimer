FROM golang:alpine AS builder

RUN apk update && apk add --no-cache git

WORKDIR $GOPATH/src/quantifyearth/reclaimer
COPY . .

RUN go get -d -v

RUN go build -o /go/bin/reclaimer


FROM scratch
COPY --from=builder /go/bin/reclaimer /go/bin/reclaimer
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT ["/go/bin/reclaimer"]
