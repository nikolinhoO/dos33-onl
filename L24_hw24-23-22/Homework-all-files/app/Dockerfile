# Сборка приложения
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main .


# Финальный образ для запуска приложения
FROM alpine:3.21 AS start

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/main .

COPY --from=builder /app/public ./public

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

CMD ["./main"]
