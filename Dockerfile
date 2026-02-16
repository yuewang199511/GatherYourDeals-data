FROM golang:1.23-alpine AS builder

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

# Copy the single config file from the repo root.
# Mount /data to persist the database, but the config comes from the image.
# Override by mounting your own config.yaml into /data/config.yaml.
COPY config.yaml /data/config.yaml

ENV GYD_CONFIG=/data/config.yaml
ENV GYD_DB=/data/gatheryourdeals.db

EXPOSE 8080

ENTRYPOINT ["gatheryourdeals"]
CMD ["serve"]
