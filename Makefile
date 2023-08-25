MAIN_PACKAGE_PATH := ./cmd/webserver
BINARY_NAME := webserver

out:
	mkdir out

.PHONY: build
build: out
	go build -o out/${BINARY_NAME} ${MAIN_PACKAGE_PATH}
# GOARCH=amd64 GOOS=linux go build -o ${BINARY_NAME}-linux main.go

.PHONY: test
test:
	go test -v -race -buildvcs ./...

.PHONY: clean
clean:
	rm -rf out
	go clean
