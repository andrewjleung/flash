run: ./bin/flash
	./bin/flash
.PHONY: run

./bin/flash:
	go build -o ./bin/flash
.PHONY: build
