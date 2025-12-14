# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build server binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o digit-link-server ./cmd/server

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/digit-link-server .

EXPOSE 8080

CMD ["./digit-link-server"]
