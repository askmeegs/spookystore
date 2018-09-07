BIN_DIR=./bin
BINARIES=users
PROJECT?=spookystore
IMAGE_TAG?=0.1

build:
	if [ -z "$$GOPATH" ]; then echo "GOPATH is not set"; exit 1; fi
	@echo "Building statically compiled linux/amd64 binaries"
	set -x; \
	  GOOS=linux GOARCH=amd64 go install \
	  -a -tags netgo \
	  -ldflags="-w -X github.com/m-okeefe/spookystore/version.version=$$(git describe --always --dirty)" \
	    $(patsubst %, ./%, $(BINARIES)) && \
	rm -rf ${BIN_DIR} && mkdir -p ${BIN_DIR} && \
	cp $(patsubst %, $$GOPATH/bin/linux_amd64/%, $(BINARIES)) ${BIN_DIR}

container:
	set -x; BINS=(${BINARIES}); for b in $${BINS[*]}; do \
	  docker build \
			-f=Dockerfile.$$b \
			-t=gcr.io/${PROJECT}/$$b:$${IMAGE_TAG} \
			--build-arg REVISION_ID="$$(git describe --always --dirty)" \
			.; \
	done

