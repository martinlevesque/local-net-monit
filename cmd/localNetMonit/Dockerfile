FROM golang:1.22.6

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/localNetMonit ./cmd/localNetMonit

# Command to run the binary
CMD ["/app/localNetMonit"]

