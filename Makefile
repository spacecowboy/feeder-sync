.PHONY: build
build: out/webserver out/store_transfer

out/store_transfer: out
	go build -o out/store_transfer ./cmd/store_transfer

out/webserver: out
	go build -o out/webserver ./cmd/webserver

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
