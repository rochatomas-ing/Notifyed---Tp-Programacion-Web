# Etapa 1: Compilación
FROM golang:1.22-alpine AS builder
WORKDIR /app

# Copiar dependencias
COPY go.mod go.sum ./
RUN go mod download

# Copiar código fuente
COPY . .

# Compilar ejecutable estático optimizado
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/api .

# Etapa 2: Imagen final liviana y segura
FROM alpine:3.20
WORKDIR /app

# Instalar certificados SSL para peticiones HTTPS externas
RUN apk --no-cache add ca-certificates

# Crear usuario sin permisos de root
RUN adduser -D -u 1000 app
USER app

# Copiar UNICAMENTE el binario al PATH del sistema
COPY --from=builder /app/api /usr/local/bin/api

# 2. Copiar los archivos estáticos del frontend para que Go los encuentre
COPY --from=builder /app/static ./static

EXPOSE 8080

CMD ["api"]