FROM golang:1.27-alpine AS builder

RUN apk add --no-cache ca-certificates tzdata git

WORKDIR /workspace

COPY go.mod go.sum ./
COPY authentication/go.mod authentication/go.sum ./authentication/
COPY bilcool-lib/go.mod bilcool-lib/go.sum ./bilcool-lib/
COPY bookings/go.mod bookings/go.sum ./bookings/
COPY event_ledger/go.mod event_ledger/go.sum ./event_ledger/
COPY journal/go.mod journal/go.sum ./journal/
COPY message_broker/go.mod message_broker/go.sum ./message_broker/
COPY testing/go.mod testing/go.sum ./testing/

RUN go work init . ./authentication ./bilcool-lib ./bookings ./event_ledger ./journal ./message_broker ./testing && \
    go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /authentication ./authentication/cmd/

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S -g 10001 app && adduser -S -u 10001 -G app app

USER 10001

COPY --from=builder /authentication /authentication

EXPOSE 8080

ENTRYPOINT ["/authentication"]
