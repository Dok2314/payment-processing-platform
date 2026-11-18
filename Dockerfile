FROM golang:1.26.2 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/api ./cmd/api

FROM alpine:3.22 AS runtime

WORKDIR /app

COPY --from=builder /app/api ./api

ENTRYPOINT ["./api"]