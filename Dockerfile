FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/webpage-analyzer ./cmd/server

FROM alpine:3.22

WORKDIR /app

COPY --from=builder /out/webpage-analyzer ./webpage-analyzer
COPY static ./static

EXPOSE 8080

CMD ["./webpage-analyzer"]
