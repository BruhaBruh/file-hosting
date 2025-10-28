# Stage 1: Builder
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Копируем go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./

# Загружаем зависимости
RUN go mod download

# Копируем весь исходный код
COPY . .

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app ./cmd/app/main.go


# Stage 2: Runtime
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Копируем бинарник из builder
COPY --from=builder /build/app .
COPY --from=builder /build/entrypoint.sh .

COPY configs/ ./configs.default/
COPY public/ ./public.default/

EXPOSE 8080

RUN chmod +x ./entrypoint.sh

CMD ["./entrypoint.sh"]
