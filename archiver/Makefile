blob-archiver:
	env GO111MODULE=on GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) go build -v $(LDFLAGS) -o ./bin/blob-archiver ./cmd/main.go

clean:
	rm -f bin/blob-archiver

test:
	go test -v -race ./...

.PHONY: \
	blob-archiver \
	clean \
	test