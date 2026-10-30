.PHONY: build run clean

build:
	@echo "Building..."
	@mkdir -p bin
	@go build -o bin/example-bot .
	@echo "Finished. Check the bin directory."

run: build
	@./bin/example-bot

clean:
	@echo "Cleaning up..."
	@rm -rf bin
	@echo "Done."
