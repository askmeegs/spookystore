.PHONY: users web test

users:
	CGO_ENABLED=0 go build --ldflags '${EXTLDFLAGS}' -o ./bin/users github.com/m-okeefe/spookystore/cmd/users

web: 
	CGO_ENABLED=0 go build --ldflags '${EXTLDFLAGS}' -o ./bin/web github.com/m-okeefe/spookystore/cmd/web

test:
	CGO_ENABLED=0 go test --cover github.com/m-okeefe/spookystore

