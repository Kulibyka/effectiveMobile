# syntax=docker/dockerfile:1

FROM golang:1.25.1-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o bin/subscribe-manager ./cmd/subscribe-manager \
    && CGO_ENABLED=0 GOOS=linux go build -o bin/migrator ./cmd/migrator

FROM alpine:3.19
WORKDIR /app

RUN adduser -D appuser
USER appuser

COPY --from=builder /app/bin/subscribe-manager ./subscribe-manager
COPY --from=builder /app/bin/migrator ./migrator
COPY config ./config
COPY migrations ./migrations
COPY docs ./docs

EXPOSE 8081

ENV CONFIG_PATH=/app/config/docker.yaml
ENV MIGRATIONS_PATH=/app/migrations

CMD ["./subscribe-manager"]
