.PHONY: users web test

build:
	CGO_ENABLED=0 go build --ldflags '${EXTLDFLAGS}' -o ./bin/spookystore github.com/m-okeefe/spookystore/cmd/spookystore

web: 
	CGO_ENABLED=0 go build --ldflags '${EXTLDFLAGS}' -o ./bin/web github.com/m-okeefe/spookystore/cmd/web

test:
	go test -v --cover github.com/m-okeefe/spookystore/cmd/spookystore

