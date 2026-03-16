# ── Build stage ───────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /build

COPY go.mod ./
COPY *.go ./

# go mod tidy downloads deps and generates go.sum in one step
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o weekly-post .

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:3.19

# ca-certificates: needed for HTTPS to Strava, Google, Telegram, OpenAI
# tzdata: needed for correct timezone handling
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/weekly-post .

CMD ["./weekly-post"]
