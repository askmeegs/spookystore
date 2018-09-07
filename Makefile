.PHONY: build container test

USERS_EXECUTABLE ?= users
IMAGE ?= bin/$(USERS_EXECUTABLE)
REPO = m-okeefe/spookystore-$(USERS_EXECUTABLE)
TAG = 0.1

build:
	CGO_ENABLED=0 go build --ldflags '${EXTLDFLAGS}' -o ${IMAGE} github.com/m-okeefe/spookystore/cmd/users

test:
	CGO_ENABLED=0 go test --cover --race github.com/m-okeefe/spookystore

container:
	docker run -t -w /go/src/github.com/m-okeefe/spookystore -v `pwd`:/go/src/github.com/m-okeefe/spookystore golang:1.11.0 make