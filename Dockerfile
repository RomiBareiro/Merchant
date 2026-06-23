# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /usr/local/bin/orchestration-api ./cmd/server

# Runtime stage
FROM scratch
COPY --from=builder /usr/local/bin/orchestration-api /orchestration-api
EXPOSE 4000
ENTRYPOINT ["/orchestration-api"]
