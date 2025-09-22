run: build
	./bin/flash
.PHONY: run

build:
	go build -o ./bin/flash
.PHONY: build
