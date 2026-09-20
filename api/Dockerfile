FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server  && CGO_ENABLED=0 GOOS=linux go build -o /out/seed ./cmd/seed  && CGO_ENABLED=0 GOOS=linux go build -o /out/migrate ./cmd/migrate

# One-off DB tooling (migrate / seed); used by the "tools" compose service.
FROM alpine:3.20 AS tools
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /out/migrate /out/seed /app/

# Default target: the API server (must stay the last stage).
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/server /app/server
EXPOSE 8080
ENTRYPOINT ["/app/server"]
