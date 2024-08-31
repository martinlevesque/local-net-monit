
dev:
	go run ./cmd/localNetMonit

test-dev:
	find . -name '*.go' | entr -r go test -v ./...


