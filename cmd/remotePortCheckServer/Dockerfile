# Use the official Go image as a build stage
FROM golang:1.22.6 AS build

# Set the current working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code to the container
COPY . .

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /remote-port-check-server ./cmd/remotePortCheckServer

# Use a lightweight image for the final stage
FROM gcr.io/distroless/static

# Copy the Go binary from the build stage
COPY --from=build /remote-port-check-server /remote-port-check-server

# Command to run the binary
CMD ["/remote-port-check-server"]

