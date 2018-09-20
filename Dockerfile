
FROM alpine
RUN apk add --update ca-certificates && \
      rm -rf /var/cache/apk/* /tmp/*

FROM golang:1.11-alpine3.7 as build
COPY . $GOPATH/src/github.com/m-okeefe/spookystore
WORKDIR $GOPATH/src/github.com/m-okeefe/spookystore
RUN go build -o ./bin/spookystore ./cmd/spookystore
COPY ./cmd/spookystore/inventory/products.json ./inventory/products.json
ENTRYPOINT ["./bin/spookystore"]
EXPOSE 8001

