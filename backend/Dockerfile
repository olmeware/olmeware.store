# ---- build stage ----
FROM golang:1.26-alpine AS build
WORKDIR /src

# Cache modules first.
COPY go.mod go.sum ./
RUN go mod download

# Build a static binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/olmeware .

# ---- runtime stage ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
USER app
WORKDIR /app
COPY --from=build /out/olmeware /app/olmeware

ENV PORT=8000
EXPOSE 8000

# Configuration is provided via environment variables (see .env.local for the
# full list): DATABASE_URL, JWT_SECRET, STRIPE_*, COINBASE_*, FRONTEND_URL, etc.
ENTRYPOINT ["/app/olmeware"]
