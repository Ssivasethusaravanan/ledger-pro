# ============================================================================
# Build Stage
# ============================================================================
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install git and root certificates
RUN apk update && apk add --no-cache git ca-certificates tzdata && update-ca-certificates

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
# CGO_ENABLED=0 ensures a static binary
# -ldflags="-w -s" strips debugging information to reduce image size
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/ledger_pro ./cmd/server/main.go

# ============================================================================
# Final Stage (Minimal Runtime)
# ============================================================================
FROM scratch

# Import certificates and timezone data from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Import the compiled binary from builder
COPY --from=builder /go/bin/ledger_pro /ledger_pro

# Expose the application port
EXPOSE 8080

# Run the binary
ENTRYPOINT ["/ledger_pro"]
