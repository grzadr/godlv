# Build stage
FROM golang:1.26-bookworm AS builder

WORKDIR /app

# Install tools
ENV PATH="/go/bin:${PATH}"
RUN go install golang.org/x/tools/cmd/stringer@latest
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin latest

# Copy go mod file
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy lint configuration
COPY .golangci.yaml ./

# Copy source code
COPY . .

# Run code generation
RUN go generate ./...

# Run linting
RUN golangci-lint run

# Run tests
RUN go test ./...

# Build the application
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o godlv .


# Final stage
FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    curl \
    ca-certificates \
    python3 \
    && rm -rf /var/lib/apt/lists/*

RUN curl -L https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp \
    -o /usr/local/bin/yt-dlp \
    && chmod a+rx /usr/local/bin/yt-dlp

# Create a non-root user
RUN groupadd -g 1000 godlv && \
    useradd -u 1000 -g godlv -s /bin/bash -m godlv

# Copy binary from builder
COPY --from=builder /app/godlv /usr/local/bin/godlv

EXPOSE 8080

USER godlv

ENTRYPOINT ["godlv"]
