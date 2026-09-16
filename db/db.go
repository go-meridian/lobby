package db

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"Lobby/config"
)

// MDB MongoDB 数据库实例
var MDB *mongo.Database

// Init 初始化 MongoDB
func Init(cfg *config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Client().ApplyURI(cfg.Mongo.URI)
	client, err := mongo.Connect(opts)
	if err != nil {
		return fmt.Errorf("db.Init mongo.Connect error: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("db.Init mongo.Ping error: %w", err)
	}

	MDB = client.Database(cfg.Mongo.Database)
	return nil
}
