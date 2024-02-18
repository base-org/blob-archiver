blob-validator:
	env GO111MODULE=on GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) go build -v $(LDFLAGS) -o ./bin/blob-validator ./cmd/main.go

clean:
	rm -f bin/blob-validator

test:
	go test -v -race ./...

.PHONY: \
	blob-validator \
	clean \
	test
