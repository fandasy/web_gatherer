# Stage 1: Build the application
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o myapp ./cmd/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/myapp .

COPY migrations ./migrations
COPY config/local.yaml ./config/local.yaml

ENV CONFIG_PATH=/app/config/local.yaml

EXPOSE 8082

RUN chmod +x ./myapp

CMD ["./myapp"]
