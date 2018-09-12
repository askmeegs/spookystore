.PHONY: build container test

USERS_BIN ?= bin/spookystore-users
WEB_BIN ?= bin/$(WEB_EXEC)

users:
	CGO_ENABLED=0 go build --ldflags '${EXTLDFLAGS}' -o ${IMAGE} github.com/m-okeefe/spookystore/cmd/users

web: 
	CGO_ENABLED=0 go build --ldflags '${EXTLDFLAGS}' -o ${IMAGE} github.com/m-okeefe/spookystore/cmd/web

test:
	CGO_ENABLED=0 go test --cover github.com/m-okeefe/spookystore

