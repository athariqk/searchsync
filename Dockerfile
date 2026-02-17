
# --- Build stage ---
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Copy example env/configs to real names for build context
RUN cp .env.example .env || true
RUN cp replica.example.yaml replica.yaml || true
RUN CGO_ENABLED=0 GOOS=linux go build -o searchsync .

# --- Final image ---
FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/searchsync ./searchsync
COPY --from=builder /app/.env ./
COPY --from=builder /app/replica.yaml ./
COPY docker-entrypoint.sh ./docker-entrypoint.sh
RUN chmod +x ./docker-entrypoint.sh \
	&& adduser -D -u 10002 searchsyncuser \
	&& chown searchsyncuser /app/.env /app/replica.yaml 2>/dev/null || true \
	&& chmod 640 /app/.env /app/replica.yaml 2>/dev/null || true
USER searchsyncuser
EXPOSE 8090
ENTRYPOINT ["./docker-entrypoint.sh"]
