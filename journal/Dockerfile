FROM golang:1.26-alpine AS builder

RUN apk add --no-cache ca-certificates tzdata git

WORKDIR /workspace

COPY go.work go.work.sum ./
COPY go.mod go.sum ./
COPY authentication/go.mod authentication/go.sum ./authentication/
COPY bilcool-lib/go.mod bilcool-lib/go.sum ./bilcool-lib/
COPY bookings/go.mod bookings/go.sum ./bookings/
COPY event_ledger/go.mod event_ledger/go.sum ./event_ledger/
COPY journal/go.mod journal/go.sum ./journal/
COPY message_broker/go.mod message_broker/go.sum ./message_broker/
COPY testing/go.mod testing/go.sum ./testing/

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /journal ./journal/cmd/

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S app -G app

USER app

COPY --from=builder /journal /journal

ENTRYPOINT ["/journal"]
