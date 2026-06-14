.PHONY: build run test generate clean docker-build

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
