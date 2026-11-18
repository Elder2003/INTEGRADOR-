# Stage 1: Builder
FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/worker ./cmd/worker
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/test-ai ./cmd/test-ai

# Stage 2: Runtime
FROM alpine:3.19 AS runtime

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/bin/api .
COPY --from=builder /build/bin/worker .

CMD ["./api"]

# Deterministic OpenAI/Ollama-compatible double used only by docker-compose.test.yml.
FROM alpine:3.19 AS test-ai

WORKDIR /app
COPY --from=builder /build/bin/test-ai .

CMD ["./test-ai"]
