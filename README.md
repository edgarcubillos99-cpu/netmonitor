# Worker-traffict-interfaces-graph

Worker-traffict-interfaces-graph es un servicio backend desarrollado en Go diseñado para la recolección masiva y concurrente de métricas de tráfico de red. El sistema consulta dispositivos activos mediante el protocolo SNMP, calcula tasas de transferencia (Mbps) y almacena los resultados en una base de datos de series de tiempo (InfluxDB) para su posterior visualización.

## 🚀 Inicio Rápido

### 1. Configurar Variables de Entorno

Crea un archivo `.env` en la raíz del proyecto basándote en `env.template`:

```bash
cp env.template .env
```

Edita `.env` con tus credenciales:

```env
MONGO_URI=Cadena de conexión a MongoDB
INFLUX_URL=URL del servidor InfluxDB
INFLUX_TOKEN=Token de autenticación para InfluxDB
INFLUX_ORG=Organización en InfluxDB
INFLUX_BUCKET=Bucket donde se guardarán los datos
WORKER_COUNT=Número de goroutines simultáneas para el pool
```

### Despliegue con Docker

El proyecto incluye un archivo docker-compose.yml para desplegar la aplicación junto con una instancia local de InfluxDB.

```bash
docker-compose up -d --build
```
Esto levantará:

app: El servicio colector (construido desde el Dockerfile).

influxdb: Instancia de InfluxDB 2.7 en el puerto 8086.

### Nota

si se desea usar un servidor influxdb existente. Edita el archivo docker-compose.yml para eliminar la definición del servicio de influxdb local y su dependencia.

El archivo docker-compose.yml resultante debería verse similar a esto:

```
services:
  app:
    build: .
    environment:
      - MONGO_URI=${MONGO_URI}
      - INFLUX_URL=${INFLUX_URL}      # Tomará el valor de .env
      - INFLUX_TOKEN=${INFLUX_TOKEN}
      - INFLUX_ORG=${INFLUX_ORG}
      - INFLUX_BUCKET=${INFLUX_BUCKET}
      - WORKER_COUNT=${WORKER_COUNT:-50}
    # depends_on eliminado
    # servicio influxdb eliminado
```
