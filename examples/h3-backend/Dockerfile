FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY main.go ./
RUN CGO_ENABLED=0 go build -o h3-backend main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/h3-backend .

EXPOSE 8081/udp
CMD ["./h3-backend"]
