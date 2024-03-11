.PHONY: clean build run

build:
	@echo "Building..."
	@go build -o bin/example-bot cmd/example-bot/main.go
	@echo "Finished script. Check bin directory."
clean:
	@echo "Cleaning up..."
	@rm bin/example-bot
	@echo "Done!"
run: build
	@bin/example-bot
