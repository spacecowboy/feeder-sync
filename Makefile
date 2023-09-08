MAIN_PACKAGE_PATH := ./cmd/webserver
BINARY_NAME := webserver

.PHONY: build
build: out/${BINARY_NAME}

out/${BINARY_NAME}: out
	go build -o out/${BINARY_NAME} ${MAIN_PACKAGE_PATH}
# GOARCH=amd64 GOOS=linux go build -o ${BINARY_NAME}-linux main.go

out:
	mkdir out

.PHONY: test
test:
	go test -v -race -buildvcs ./...

.PHONY: deploy
deploy: build
	scripts/deploy

.PHONY: clean
clean:
	rm -rf out
	go clean
