# ── Build stage ───────────────────────────────────────────────────────────────
FROM golang:1.26.1-alpine AS builder

WORKDIR /build

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -o weekly-post .

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /build/weekly-post .

RUN chown appuser:appgroup /app/weekly-post

USER appuser

CMD ["./weekly-post"]
