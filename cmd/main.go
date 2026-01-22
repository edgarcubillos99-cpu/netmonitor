package main

import (
	"context"
	"log"
	"netmonitor/internal/repository"
	"netmonitor/internal/storage"
	"netmonitor/internal/worker"
	"os"
	"strconv"
	"time"
)

func main() {
	// Configuración desde ENV
	mongoURI := os.Getenv("MONGO_URI") // ej: mongodb://admin:pass@172.16.9.150:27017
	influxURL := os.Getenv("INFLUX_URL")
	influxToken := os.Getenv("INFLUX_TOKEN")
	influxOrg := os.Getenv("INFLUX_ORG")
	influxBucket := os.Getenv("INFLUX_BUCKET")
	workerCountStr := os.Getenv("WORKER_COUNT") // Nueva variable de entorno

	// Validar variables de entorno requeridas
	if mongoURI == "" {
		log.Fatal("MONGO_URI no está configurado. Por favor, configura la variable de entorno.")
	}
	if influxURL == "" {
		log.Fatal("INFLUX_URL no está configurado. Por favor, configura la variable de entorno.")
	}
	if influxToken == "" {
		log.Fatal("INFLUX_TOKEN no está configurado. Por favor, configura la variable de entorno.")
	}
	if influxOrg == "" {
		log.Fatal("INFLUX_ORG no está configurado. Por favor, configura la variable de entorno.")
	}
	if influxBucket == "" {
		log.Fatal("INFLUX_BUCKET no está configurado. Por favor, configura la variable de entorno.")
	}

	// Lógica de configuración de Workers
	workers := 50 // Valor por defecto
	if workerCountStr != "" {
		val, err := strconv.Atoi(workerCountStr)
		if err != nil {
			log.Printf("⚠️ WORKER_COUNT inválido ('%s'). Usando valor por defecto: 50", workerCountStr)
		} else {
			workers = val
		}
	}

	// 1. Conexión MongoDB
	log.Println("Conectando a MongoDB...")
	repo, err := repository.NewDeviceRepo(mongoURI, "MDM", "devices")
	if err != nil {
		log.Fatalf("Fallo MongoDB: %v", err)
	}
	log.Println("Conectado a colección MDM.devices")

	// 2. InfluxDB & State
	stateManager := storage.NewStateManager()
	influxWriter := storage.NewInfluxWriter(influxURL, influxToken, influxOrg, influxBucket)

	// 3. Iniciar Worker Pool
	// Se usa la variable 'workers' en lugar del valor fijo
	log.Printf("Iniciando Worker Pool con %d workers...", workers)
	wp := worker.NewPool(workers, stateManager, influxWriter)
	wp.Start()

	// 4. Loop Principal (Ticker 3 min)
	ticker := time.NewTicker(3 * time.Minute)
	defer ticker.Stop()

	// Ejecución inmediata al arrancar
	ctx := context.Background()
	go wp.RunCycle(ctx, repo)

	// Ejecución programada
	for range ticker.C {
		log.Println("---⏰ Iniciando Ciclo Programado ---")
		go wp.RunCycle(ctx, repo)
	}
}
