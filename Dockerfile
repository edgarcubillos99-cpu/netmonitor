# Etapa de construcción
FROM golang:1.25.3-alpine AS builder

# Instalar dependencias necesarias para compilar
RUN apk add --no-cache git ca-certificates

# Establecer directorio de trabajo
WORKDIR /app

# Copiar archivos de dependencias
COPY go.mod go.sum ./
RUN go mod download

# Copiar el código fuente
COPY . .

# Compilar la aplicación
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o collector ./cmd/collector

# Etapa final - imagen mínima
FROM alpine:latest

# Instalar ca-certificates para conexiones HTTPS/TLS
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copiar el binario compilado desde la etapa builder
COPY --from=builder /app/collector .

# Ejecutar la aplicación
CMD ["./collector"]

