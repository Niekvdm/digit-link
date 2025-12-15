# Frontend build stage
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

# Copy frontend package files
COPY frontend/package.json frontend/yarn.lock ./
RUN yarn install

# Copy frontend source and build
COPY frontend/ ./
RUN yarn build

# Go build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies for CGO (required by go-sqlite3)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Copy built frontend to the embed location
COPY --from=frontend-builder /app/frontend/dist/ ./internal/server/public/

# Build server binary with CGO enabled (required for sqlite3)
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o digit-link-server ./cmd/server

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/digit-link-server .

# Create data directory for SQLite database
RUN mkdir -p /data

EXPOSE 8080

CMD ["./digit-link-server"]
