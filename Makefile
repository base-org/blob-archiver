build:
	make -C ./archiver blob-archiver
	make -C ./api blob-api
.PHONY: build

build-docker:
	docker-compose build
.PHONY: build-docker

clean:
	make -C ./archiver clean
	make -C ./api clean
.PHONY: clean

test:
	make -C ./archiver test
	make -C ./api test
.PHONY: test

integration:
	docker-compose down
	docker-compose up -d minio create-buckets
	RUN_INTEGRATION_TESTS=true go test -v ./...
.PHONY: integration

fmt:
	gofmt -s -w .
.PHONY: fmt

check: fmt clean build build-docker lint test integration
.PHONY: check

lint:
	golangci-lint run -E goimports,sqlclosecheck,bodyclose,asciicheck,misspell,errorlint --timeout 5m -e "errors.As" -e "errors.Is" ./...
.PHONY: lint