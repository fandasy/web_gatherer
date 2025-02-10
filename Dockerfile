# Stage 1: Build the application
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache \
    gcc \
    musl-dev \
    git \
    make \
    cmake \
    g++ \
    zlib-dev \
    openssl-dev

# Clone and build Tdlib
WORKDIR /src
RUN git clone https://github.com/tdlib/td.git && \
    mkdir -p td/build && \
    cd td/build && \
    cmake -DCMAKE_BUILD_TYPE=Release .. && \
    make -j$(nproc) && \
    make install

WORKDIR /app
COPY . .

ENV CGO_ENABLED=1
ENV CGO_CFLAGS="-I/src/td -I/src/td/build"
ENV CGO_LDFLAGS="-L/src/td/build -ltdjson"

RUN go build -trimpath -ldflags="-s -w" -o /app/bin/main ./cmd/main.go

# Stage 2: Create a lightweight final image
FROM alpine:latest

RUN apk add --no-cache \
    libstdc++ \
    openssl \
    zlib

COPY --from=builder /app/bin/main /usr/local/bin/main
COPY config/local.yaml /app/config/local.yaml
COPY migrations /app/migrations

ENV CONFIG_PATH=/app/config/local.yaml

EXPOSE 49500

CMD ["/usr/local/bin/main"]