FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN for cmd in server seed migrate createhost; do \
      CGO_ENABLED=0 GOOS=linux go build -o /out/$cmd ./cmd/$cmd || exit 1; \
    done

# One-off DB tooling (migrate / seed / createhost); used by the "tools" compose service.
FROM alpine:3.20 AS tools
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /out/migrate /out/seed /out/createhost /app/

# Default target: the API server (must stay the last stage).
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/server /app/server
EXPOSE 8080
ENTRYPOINT ["/app/server"]
