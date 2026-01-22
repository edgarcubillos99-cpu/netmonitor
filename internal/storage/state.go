package storage

import (
	"sync"
	"time"
)

// InterfaceMetric representa una métrica de interfaz obtenida de SNMP
type InterfaceMetric struct {
	DeviceID    string
	InterfaceID string
	InOctets    uint64
	OutOctets   uint64
	Timestamp   time.Time
}

// TrafficResult contiene el resultado calculado de tráfico
type TrafficResult struct {
	DeviceID    string
	InterfaceID string
	InRateMbps  float64
	OutRateMbps float64
	InBytes     uint64
	OutBytes    uint64
	Timestamp   time.Time
}

// StateManager mantiene el estado anterior de las métricas
type StateManager struct {
	mu    sync.RWMutex
	state map[string]InterfaceMetric // Clave: "deviceID:interfaceID"
}

// NewStateManager crea un nuevo gestor de estado
func NewStateManager() *StateManager {
	return &StateManager{
		state: make(map[string]InterfaceMetric),
	}
}

// GetAndUpdate implementa la lógica simplificada de "netmon" original.
func (sm *StateManager) GetAndUpdate(metric InterfaceMetric) (TrafficResult, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	key := metric.DeviceID + ":" + metric.InterfaceID
	prev, exists := sm.state[key]

	sm.state[key] = metric

	if !exists {
		return TrafficResult{}, false
	}

	timeDiff := metric.Timestamp.Sub(prev.Timestamp).Seconds()
	// Validación: Evitar tiempos muy pequeños que causen divisiones enormes
	if timeDiff < 1.0 {
		return TrafficResult{}, false
	}

	if metric.InOctets < prev.InOctets || metric.OutOctets < prev.OutOctets {
		return TrafficResult{}, false
	}

	deltaIn := metric.InOctets - prev.InOctets
	deltaOut := metric.OutOctets - prev.OutOctets

	inRateMbps := (float64(deltaIn) * 8) / (timeDiff * 1_000_000)
	outRateMbps := (float64(deltaOut) * 8) / (timeDiff * 1_000_000)

	// --- NUEVA VALIDACIÓN: SANITY CHECK ---
	// Si la velocidad supera 200 Gbps (200,000 Mbps), es un error del router.
	// Ajusta este valor según la capacidad máxima real de tu red.
	const MaxValidSpeedMbps = 200000.0

	if inRateMbps > MaxValidSpeedMbps || outRateMbps > MaxValidSpeedMbps {
		// Loguear si quieres, pero retornamos false para ignorar el dato
		return TrafficResult{}, false
	}
	// --------------------------------------

	return TrafficResult{
		DeviceID:    metric.DeviceID,
		InterfaceID: metric.InterfaceID,
		InRateMbps:  inRateMbps,
		OutRateMbps: outRateMbps,
		InBytes:     deltaIn,
		OutBytes:    deltaOut,
		Timestamp:   metric.Timestamp,
	}, true
}
