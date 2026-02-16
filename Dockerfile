# Production-ready Dockerfile for searchsync (Go NSQ→Meilisearch sync)
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o searchsync .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/searchsync ./searchsync
COPY config.example.yaml ./config.example.yaml
COPY docker-entrypoint.sh ./docker-entrypoint.sh
RUN chmod +x ./docker-entrypoint.sh
RUN adduser -D -u 10002 searchsyncuser
USER searchsyncuser
EXPOSE 8090
ENTRYPOINT ["./docker-entrypoint.sh"]
