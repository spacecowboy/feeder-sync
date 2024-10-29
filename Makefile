.PHONY: build
build: out/webserver out/db_migrator out/janitor

out/webserver: out generate
	go build -o out/webserver ./cmd/webserver

out/db_migrator: out
	go build -o out/db_migrator ./cmd/db_migrator

out/janitor: out
	go build -o out/janitor ./cmd/janitor

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
