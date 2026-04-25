.PHONY: clean run test

clean:
	go clean
	go mod tidy
	rm -f coverage.out coverage.html

run:
	chmod +x scripts/run.sh
	./scripts/run.sh

test:
	go test -race ./...
