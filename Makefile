
boot:
	REMOTE_PORT_CHECKER_BASE_URL="http://localhost:8081" MONITOR_PUBLIC_PORTS=true go run ./cmd/localNetMonit

port-check-server:
	go run ./cmd/remotePortCheckServer

test-dev:
	find . -name '*.go' | entr -r go test -v ./...


