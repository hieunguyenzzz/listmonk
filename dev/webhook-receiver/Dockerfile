FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY main.go .
RUN go build -o webhook-receiver main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/webhook-receiver .
EXPOSE 8888
CMD ["./webhook-receiver"]
