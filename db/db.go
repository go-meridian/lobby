package db

import (
	"context"
	"time"

	"lobby/model/codeerror"

	"lobby/config"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MDB MongoDB 数据库实例
var MDB *mongo.Database

// Init 初始化 MongoDB
func Init(cfg *config.Config) *codeerror.CodeError {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Client().ApplyURI(cfg.Mongo.URI)
	client, err := mongo.Connect(opts)
	if err != nil {
		return codeerror.DBError.Msg("db.Init mongo.Connect error: " + err.Error())
	}

	if err := client.Ping(ctx, nil); err != nil {
		return codeerror.DBError.Msg("db.Init mongo.Ping error: " + err.Error())
	}

	MDB = client.Database(cfg.Mongo.Database)
	return nil
}
