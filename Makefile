
boot:
	MONITOR_PUBLIC_PORTS=false go run ./cmd/localNetMonit

port-check-server:
	go run ./cmd/remotePortCheckServer

test-dev:
	find . -name '*.go' | entr -r go test -v ./...


