# syntax=docker/dockerfile:1

FROM golang:1.22-bookworm AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN go build -o /out/caiatech-datalab-api ./cmd/api

FROM debian:bookworm-slim
WORKDIR /app
RUN useradd -r -u 10001 app

COPY --from=build /out/caiatech-datalab-api /app/api
COPY migrations /app/migrations

USER 10001
EXPOSE 8080

ENV DATALAB_LISTEN_ADDR=:8080
ENV DATALAB_MIGRATIONS_DIR=/app/migrations

ENTRYPOINT ["/app/api"]
