.PHONY: build run test generate clean docker-build sync sync-force weekly

BINARY=coachd

build:
	go build -o bin/$(BINARY) ./cmd/coachd

run: build
	./bin/$(BINARY)

test:
	go test -race ./...

generate:
	oapi-codegen --config oapi-codegen.yaml openapi-spec.json

clean:
	rm -rf bin/

docker-build:
	docker build -t coachd:latest .

sync: build
	./bin/$(BINARY) -sync

sync-force: build
	./bin/$(BINARY) -sync -force

weekly: build
	./bin/$(BINARY) -weekly
