GITCOMMIT ?= $(shell git rev-parse HEAD)
LDFLAGS := -ldflags "-X github.com/base-org/blob-archiver/api/version.GitCommit=$(GITCOMMIT)"

blob-api:
	env GO111MODULE=on GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) go build -v $(LDFLAGS) -o ./bin/blob-api ./cmd/main.go

clean:
	rm -f bin/blob-api

test:
	go test -v -race ./...

.PHONY: \
	blob-api \
	clean \
	test
