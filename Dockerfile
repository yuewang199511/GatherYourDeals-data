FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 go build -o gatheryourdeals ./cmd/gatheryourdeals

# --- Runtime image ---
FROM alpine:latest

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/gatheryourdeals /usr/local/bin/gatheryourdeals

# Data directory for the database file
RUN mkdir -p /data
WORKDIR /data

EXPOSE 8080

ENTRYPOINT ["gatheryourdeals", "--config", "/data/config.yaml"]
CMD ["serve"]
