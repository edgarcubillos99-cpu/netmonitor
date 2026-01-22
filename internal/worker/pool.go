package worker

import (
	"context"
	"log"
	"netmonitor/internal/models"
	"netmonitor/internal/repository"
	"netmonitor/internal/snmp"
	"netmonitor/internal/storage"
	"sync"
)

type Pool struct {
	Workers      int
	JobQueue     chan models.Device
	WaitGroup    sync.WaitGroup
	StateManager *storage.StateManager
	Influx       *storage.InfluxWriter
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
			log.Printf("[W%d] Err %s (%s): %v", id, device.Name, device.IP, err)
			p.WaitGroup.Done()
			continue
		}

		var results []storage.TrafficResult
		for _, m := range metrics {
			// 1. Omitir si la interfaz no está UP
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

			if im.InterfaceID == "" {
				im.InterfaceID = m.InterfaceID
			}

			res, ok := p.StateManager.GetAndUpdate(im)
			if ok {
				// 2. Omitir si el tráfico supera la modulación (capacidad máxima)
				if m.MaxSpeed > 0 {
					if res.InRateMbps > float64(m.MaxSpeed) || res.OutRateMbps > float64(m.MaxSpeed) {
						log.Printf("⚠️ [%s] %s: Tráfico irreal omitido (In: %.2f / Out: %.2f Mbps) > Capacidad: %d Mbps",
							device.Name, im.InterfaceID, res.InRateMbps, res.OutRateMbps, m.MaxSpeed)
						continue
					}
				}
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
	devices, err := repo.GetActiveDevices(ctx)
	if err != nil {
		log.Printf("Error obteniendo dispositivos de Mongo: %v", err)
		return
	}

	log.Printf("Despachando %d dispositivos activos...", len(devices))
	p.WaitGroup.Add(len(devices))

	for _, d := range devices {
		p.JobQueue <- d
	}
	p.WaitGroup.Wait()
	log.Println("✅ Ciclo completado.")
}
