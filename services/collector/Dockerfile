FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o collector .

FROM alpine:3.19

WORKDIR /app
COPY --from=builder /app/collector .

EXPOSE 8080

CMD ["./collector"]
