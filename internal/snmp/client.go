package snmp

import (
	"fmt"
	"netmonitor/internal/models"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
)

// OIDs constantes
const (
	oidIfHCInOctets  = ".1.3.6.1.2.1.31.1.1.1.6"
	oidIfHCOutOctets = ".1.3.6.1.2.1.31.1.1.1.10"
	oidIfDescr       = ".1.3.6.1.2.1.31.1.1.1.18"
	oidIfHighSpeed   = ".1.3.6.1.2.1.31.1.1.1.15" // Velocidad en Mbps
	oidIfOperStatus  = ".1.3.6.1.2.1.2.2.1.8"     // 1=up, 2=down
	oidIfInOctets    = ".1.3.6.1.2.1.2.2.1.10"    // 32-bit fallback
	oidIfOutOctets   = ".1.3.6.1.2.1.2.2.1.16"    // 32-bit fallback
)

type MetricResult struct {
	InterfaceID string
	Alias       string
	InOctets    uint64
	OutOctets   uint64
	MaxSpeed    uint64 // Capacidad de la interfaz para comparar modulación
	IsUp        bool   // Estado operacional
	Timestamp   time.Time
}

func FetchMetrics(device models.Device) ([]MetricResult, error) {
	var version gosnmp.SnmpVersion
	switch device.GetVersion() {
	case "1":
		version = gosnmp.Version1
	case "2c", "2":
		version = gosnmp.Version2c
	case "3":
		version = gosnmp.Version3
	default:
		version = gosnmp.Version2c
	}

	params := &gosnmp.GoSNMP{
		Target:         device.IP,
		Port:           device.GetPort(),
		Community:      device.GetCommunity(),
		Version:        version,
		Timeout:        time.Millisecond * 1500, // Optimización 1: Reducción de espera a 1.5s
		MaxRepetitions: 50,                      // Optimización 2: Traer más datos por paquete
	}

	if err := params.Connect(); err != nil {
		return nil, fmt.Errorf("connect err: %w", err)
	}
	defer params.Conn.Close()

	tempData := make(map[string]*MetricResult)

	// Manejador de PDUs recibidos
	handler := func(pdu gosnmp.SnmpPDU) error {
		oidParts := strings.Split(pdu.Name, ".")
		if len(oidParts) < 1 {
			return nil
		}
		ifIndex := oidParts[len(oidParts)-1]

		if _, exists := tempData[ifIndex]; !exists {
			tempData[ifIndex] = &MetricResult{InterfaceID: ifIndex, Timestamp: time.Now()}
		}

		switch {
		case strings.HasPrefix(pdu.Name, oidIfInOctets):
			// Solo asignar si no tenemos ya el valor de 64 bits (que es preferido)
			if tempData[ifIndex].InOctets == 0 {
				tempData[ifIndex].InOctets = gosnmp.ToBigInt(pdu.Value).Uint64()
			}
		case strings.HasPrefix(pdu.Name, oidIfOutOctets):
			if tempData[ifIndex].OutOctets == 0 {
				tempData[ifIndex].OutOctets = gosnmp.ToBigInt(pdu.Value).Uint64()
			}
		case strings.HasPrefix(pdu.Name, oidIfHighSpeed):
			tempData[ifIndex].MaxSpeed = gosnmp.ToBigInt(pdu.Value).Uint64()
		case strings.HasPrefix(pdu.Name, oidIfOperStatus):
			status := gosnmp.ToBigInt(pdu.Value).Int64()
			tempData[ifIndex].IsUp = (status == 1)
		case strings.HasPrefix(pdu.Name, oidIfDescr):
			switch v := pdu.Value.(type) {
			case []byte:
				tempData[ifIndex].Alias = string(v)
			case string:
				tempData[ifIndex].Alias = v
			default:
				tempData[ifIndex].Alias = fmt.Sprintf("%v", v)
			}
		}
		return nil
	}

	// Ejecución secuencial de caminatas
	// Nota: ifHCInOctets e ifHCOutOctets son las más críticas.
	targetOIDs := []string{
		oidIfHCInOctets,
		oidIfHCOutOctets,
		oidIfHighSpeed,
		oidIfOperStatus,
		oidIfDescr,
		oidIfInOctets,
		oidIfOutOctets,
	}

	for _, rootOid := range targetOIDs {
		err := params.BulkWalk(rootOid, handler)
		if err != nil {
		}
	}

	var results []MetricResult
	for _, m := range tempData {
		// Solo enviamos si tiene datos o está activo, para evitar basura
		if m.InOctets > 0 || m.OutOctets > 0 || m.IsUp {
			results = append(results, *m)
		}
	}

	return results, nil
}
