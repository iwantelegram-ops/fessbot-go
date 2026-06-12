package db

import (
	"context"
	"log"
	"time"

	"fessbot/internal/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	Client       *mongo.Client
	Database     *mongo.Database
	Partners     *mongo.Collection
	Posts        *mongo.Collection
	Users        *mongo.Collection
	BlacklistCol *mongo.Collection
	SettingsCol  *mongo.Collection
	BroadcastCol *mongo.Collection
	ActivityCol  *mongo.Collection
	NotifCol     *mongo.Collection
)

func Connect() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var err error
	Client, err = mongo.Connect(ctx, options.Client().ApplyURI(config.MongoURI))
	if err != nil {
		log.Fatalf("[db] Gagal connect MongoDB: %v", err)
	}
	if err = Client.Ping(ctx, nil); err != nil {
		log.Fatalf("[db] MongoDB tidak merespons: %v", err)
	}

	Database = Client.Database("fessbot")
	Partners = Database.Collection("partners")
	Posts = Database.Collection("posts")
	Users = Database.Collection("users")
	BlacklistCol = Database.Collection("blacklist")
	SettingsCol = Database.Collection("settings")
	BroadcastCol = Database.Collection("broadcasts")
	ActivityCol = Database.Collection("activity")
	NotifCol = Database.Collection("notifications")

	ensureIndexes()
	log.Println("[db] MongoDB terhubung ✅")
}

func ensureIndexes() {
	ctx := context.Background()

	createIndex(ctx, Posts, bson.D{{Key: "partner_id", Value: 1}})
	createIndex(ctx, Posts, bson.D{{Key: "added_at", Value: -1}})
	createIndex(ctx, ActivityCol, bson.D{{Key: "ts", Value: -1}})
	createIndex(ctx, ActivityCol, bson.D{{Key: "partner_id", Value: 1}, {Key: "ts", Value: -1}})
	createIndex(ctx, Users, bson.D{{Key: "joined", Value: 1}})
	createIndex(ctx, BroadcastCol, bson.D{{Key: "sent_at", Value: -1}})
}

func createIndex(ctx context.Context, col *mongo.Collection, keys bson.D) {
	_, _ = col.Indexes().CreateOne(ctx, mongo.IndexModel{Keys: keys})
}

func Disconnect() {
	if Client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = Client.Disconnect(ctx)
	}
}
