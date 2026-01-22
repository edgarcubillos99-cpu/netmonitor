package worker

import (
	"context"
	"log"
	"netmonitor/internal/models"
	"netmonitor/internal/repository"
	"netmonitor/internal/snmp"
	"netmonitor/internal/storage"
	"sync"
	"time"
)

// Constante de seguridad: Ninguna interfaz de acceso debería superar esto.
// Ayuda a filtrar errores de cálculo por reinicios de contadores.
const GlobalMaxRateLimitMbps = 20000.0 // 20 Gbps

type Pool struct {
	Workers      int
	JobQueue     chan models.Device
	WaitGroup    sync.WaitGroup
	StateManager *storage.StateManager
	Influx       *storage.InfluxWriter

	// Control de concurrencia para evitar solapamiento de ciclos
	mu        sync.Mutex
	isRunning bool
}

func NewPool(workers int, stateManager *storage.StateManager, influx *storage.InfluxWriter) *Pool {
	return &Pool{
		Workers:      workers,
		JobQueue:     make(chan models.Device, workers*2),
		StateManager: stateManager,
		Influx:       influx,
	}
}

func (p *Pool) Start() {
	for i := 1; i <= p.Workers; i++ {
		go p.workerLogic(i)
	}
	log.Printf("✅ Pool iniciado con %d workers", p.Workers)
}

func (p *Pool) workerLogic(id int) {
	for device := range p.JobQueue {
		metrics, err := snmp.FetchMetrics(device)
		if err != nil {
			// Reducimos el log a advertencia para no saturar si un equipo está apagado
			// log.Printf("[W%d] Info: No se pudo conectar a %s (%s): %v", id, device.Name, device.IP, err)
			p.WaitGroup.Done()
			continue
		}

		var results []storage.TrafficResult
		for _, m := range metrics {
			// 1. Omitir si la interfaz no está UP administrativamente o operativamente
			if !m.IsUp {
				continue
			}

			im := storage.InterfaceMetric{
				DeviceID:    device.Name,
				InterfaceID: m.Alias,
				InOctets:    m.InOctets,
				OutOctets:   m.OutOctets,
				Timestamp:   m.Timestamp,
			}

			// Fallback si no hay Alias (Description), usar el ID numérico o nombre
			if im.InterfaceID == "" {
				im.InterfaceID = m.InterfaceID
			}

			res, ok := p.StateManager.GetAndUpdate(im)
			if ok {
				// --- INICIO CORRECCIÓN: SANITY CHECK ROBUSTO ---

				// A. Determinar capacidad real o asumir default seguro
				maxSpeed := float64(m.MaxSpeed)
				if maxSpeed == 0 {
					maxSpeed = 1000.0 // Asumir 1Gbps si el dispositivo no reporta velocidad (común en radios)
				}

				// B. Definir un umbral de tolerancia (ej. 150% de la capacidad teórica para permitir burst)
				toleranceLimit := maxSpeed * 1.5

				// C. Validación de "Realidad"
				// Si el tráfico es mayor al límite tolerado O mayor al límite físico global (20Gbps)
				if res.InRateMbps > toleranceLimit || res.OutRateMbps > toleranceLimit ||
					res.InRateMbps > GlobalMaxRateLimitMbps || res.OutRateMbps > GlobalMaxRateLimitMbps {

					log.Printf("⚠️ [%s] DROP ANOMALÍA en %s: In: %.2f / Out: %.2f Mbps (Capacidad: %.0f Mbps)",
						device.Name, im.InterfaceID, res.InRateMbps, res.OutRateMbps, maxSpeed)

					// IMPORTANTE: No guardamos este punto anómalo en InfluxDB
					continue
				}
				// --- FIN CORRECCIÓN ---

				results = append(results, res)
			}
		}

		if len(results) > 0 {
			p.Influx.Write(results)
		}
		p.WaitGroup.Done()
	}
}

func (p *Pool) RunCycle(ctx context.Context, repo *repository.DeviceRepo) {
	// 1. Protección contra solapamiento de ciclos (Race Condition)
	p.mu.Lock()
	if p.isRunning {
		p.mu.Unlock()
		log.Println("⚠️ ALERTA: El ciclo anterior aún no termina. Saltando esta ejecución para evitar corrupción de métricas.")
		return
	}
	p.isRunning = true
	p.mu.Unlock()

	// Asegurar que liberamos el flag al salir
	defer func() {
		p.mu.Lock()
		p.isRunning = false
		p.mu.Unlock()
	}()

	start := time.Now()
	devices, err := repo.GetActiveDevices(ctx)
	if err != nil {
		log.Printf("Error obteniendo dispositivos de Mongo: %v", err)
		return
	}

	count := len(devices)
	log.Printf("🔄 Iniciando ciclo: Despachando %d dispositivos...", count)

	p.WaitGroup.Add(count)

	for _, d := range devices {
		select {
		case p.JobQueue <- d:
			// Enviado correctamente
		case <-ctx.Done():
			log.Println("Contexto cancelado, deteniendo despacho.")
			p.WaitGroup.Done() // Ajustar contador si no se pudo enviar
			return
		}
	}

	p.WaitGroup.Wait()
	duration := time.Since(start)
	log.Printf("✅ Ciclo completado en %v. (Avg por dispositivo: %v)", duration, duration/time.Duration(count))
}
