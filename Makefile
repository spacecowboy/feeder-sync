.PHONY: build
build: out/webserver out/store_transfer out/db_migrator

out/store_transfer: out generate
	go build -o out/store_transfer ./cmd/store_transfer

out/webserver: out generate
	go build -o out/webserver ./cmd/webserver

out/db_migrator: out
	go build -o out/db_migrator ./cmd/db_migrator

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

.PHONY:
generate:
	rm -rf build/gen
	sqlc generate
