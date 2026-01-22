package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Estructura basada en colección MongoDB
type Device struct {
	ID          primitive.ObjectID `bson:"_id"`
	IP          string             `bson:"ip"`
	Name        string             `bson:"name"`
	Make        string             `bson:"make"`
	Description string             `bson:"description"`
	StatusIcmp  string             `bson:"StatusIcmp"`

	// Mapeo deSnmpSettings
	SnmpSettings struct {
		Community *string `bson:"community,omitempty"` // Puntero para detectar null/missing
		Port      *uint16 `bson:"port,omitempty"`      // Puerto SNMP (default: 161)
		Protocol  string  `bson:"protocol,omitempty"`  // Protocolo (UDP por defecto)
		Version   string  `bson:"version,omitempty"`   // Versión SNMP (v2c por defecto)
	} `bson:"SnmpSettings"`
}

// GetCommunity obtiene la comunidad SNMP con fallback seguro
func (d Device) GetCommunity() string {
	if d.SnmpSettings.Community != nil && *d.SnmpSettings.Community != "" {
		return *d.SnmpSettings.Community
	}
	return "public" // Fallback seguro
}

// GetPort obtiene el puerto SNMP con fallback a 161 (puerto estándar)
func (d Device) GetPort() uint16 {
	if d.SnmpSettings.Port != nil && *d.SnmpSettings.Port > 0 {
		return *d.SnmpSettings.Port
	}
	return 161 // Puerto estándar SNMP
}

// GetVersion obtiene la versión SNMP con fallback a v2c
func (d Device) GetVersion() string {
	if d.SnmpSettings.Version != "" {
		return d.SnmpSettings.Version
	}
	return "2c" // Versión por defecto
}
