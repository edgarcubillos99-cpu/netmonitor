// aquí almacenamos las métricas en InfluxDB
package storage

import (
	"context"
	"log"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

// InfluxWriter maneja la escritura de métricas a InfluxDB
type InfluxWriter struct {
	client   influxdb2.Client
	writeAPI api.WriteAPIBlocking
	bucket   string
	org      string
}

// NewInfluxWriter crea un nuevo escritor de InfluxDB
func NewInfluxWriter(url, token, org, bucket string) *InfluxWriter {
	client := influxdb2.NewClient(url, token)
	writeAPI := client.WriteAPIBlocking(org, bucket)

	return &InfluxWriter{
		client:   client,
		writeAPI: writeAPI,
		bucket:   bucket,
		org:      org,
	}
}

// Write escribe los resultados de tráfico a InfluxDB
func (iw *InfluxWriter) Write(results []TrafficResult) {
	if len(results) == 0 {
		return
	}

	points := make([]*write.Point, 0, len(results))
	for _, result := range results {
		// Crear punto de datos para InfluxDB
		point := influxdb2.NewPoint(
			"network_traffic",
			map[string]string{
				"device":    result.DeviceID,
				"interface": result.InterfaceID,
			},
			map[string]interface{}{
				"in_rate_mbps":  result.InRateMbps,  // Megabits por segundo de entrada
				"out_rate_mbps": result.OutRateMbps, // Megabits por segundo de salida
				"in_bytes":      result.InBytes,     // Bytes totales de entrada en el intervalo
				"out_bytes":     result.OutBytes,    // Bytes totales de salida en el intervalo
			},
			result.Timestamp,
		)
		points = append(points, point)
	}

	// Escribir en batch
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := iw.writeAPI.WritePoint(ctx, points...); err != nil {
		log.Printf("Error escribiendo a InfluxDB: %v", err)
		return
	}

	log.Printf("Escritas %d métricas a InfluxDB", len(results))
}

// Close cierra la conexión con InfluxDB
func (iw *InfluxWriter) Close() {
	if iw.client != nil {
		iw.client.Close()
	}
}
