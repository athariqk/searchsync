# Production-ready Dockerfile for searchsync (Go NSQ→Meilisearch sync)
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o searchsync main.go

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/searchsync ./searchsync
COPY config.yaml ./config.yaml
COPY .env* ./
RUN adduser -D -u 10002 searchsyncuser
USER searchsyncuser
EXPOSE 8090
ENTRYPOINT ["./searchsync"]
