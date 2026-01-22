package repository

import (
	"context"
	"netmonitor/internal/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DeviceRepo struct {
	client     *mongo.Client
	collection *mongo.Collection
}

// NewDeviceRepo crea una nueva conexión a MongoDB
func NewDeviceRepo(uri, dbName, collName string) (*DeviceRepo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	// Ping para verificar conexión
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	// Obtener colección de dispositivos
	coll := client.Database(dbName).Collection(collName)
	return &DeviceRepo{client: client, collection: coll}, nil
}

// GetActiveDevices obtiene los dispositivos activos de MongoDB
func (r *DeviceRepo) GetActiveDevices(ctx context.Context) ([]models.Device, error) {
	// Filtro: StatusIcmp = up
	filter := bson.M{"StatusIcmp": "up"}

	// Opcional: Proyección para traer solo campos necesarios y ahorrar ancho de banda
	opts := options.Find().SetProjection(bson.M{
		"ip": 1, "name": 1, "SnmpSettings": 1, "isActive": 1, "_id": 1,
	})

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var devices []models.Device
	if err = cursor.All(ctx, &devices); err != nil {
		return nil, err
	}
	return devices, nil
}
