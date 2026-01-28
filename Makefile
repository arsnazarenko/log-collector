BUILD_DIR = build
COLLECTOR_CMD = cmd/collector/main.go
GENERATOR_CMD = cmd/generator/main.go

all: generate-api generate-swagger build build-generator

generate-api: api/openapi/v1/api.yaml
	go generate ./api/...

generate-swagger: api/openapi/v1/api.yaml
	docker run --rm -i yousan/swagger-yaml-to-html < api/openapi/v1/api.yaml > api/openapi/v1/index.html

test:
	go test -v -cover ./...

build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -o $(BUILD_DIR)/collector $(COLLECTOR_CMD)

build-generator:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -o $(BUILD_DIR)/generator $(GENERATOR_CMD)

run: build
	./build/main

clean:
	rm -rf ./build

.PHONY: all generate-api generate-swagger test clean build build-generator run clean


