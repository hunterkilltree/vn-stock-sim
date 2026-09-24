# syntax=docker/dockerfile:1

# --- build stage ---
FROM golang:1.24-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api

# --- runtime stage ---
FROM alpine:3.20
RUN adduser -D -u 10001 appuser
COPY --from=build /out/api /usr/local/bin/api
USER appuser
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
