
boot-local-net-monit:
	go run ./cmd/localNetMonit

boot-port-check-server:
	go run ./cmd/remotePortCheckServer

test-dev:
	find . -name '*.go' | entr -r go test -v ./...


