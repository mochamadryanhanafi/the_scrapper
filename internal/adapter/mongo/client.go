package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// NewClient membuat koneksi baru ke MongoDB.
func NewClient(ctx context.Context, uri string) (*mongo.Client, error) {
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.NewClient(clientOptions)
	if err != nil {
		return nil, err
	}

	// Hubungkan ke server MongoDB
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err = client.Connect(connectCtx)
	if err != nil {
		return nil, err
	}

	// Cek koneksi
	if err = client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	return client, nil
}
