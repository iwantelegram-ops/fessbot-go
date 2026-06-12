package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

var (
	APIID                int
	APIHash              string
	BotToken             string
	MongoURI             string
	MainChannelID        int64
	MainChannelUsername  string
	OwnerID              int64
	BotUsername          string
	OwnerUsername        string
	OwnerName            string
	BotName              string
	BotDesc              string
	FloodSleepThreshold  int
	BroadcastDelay       float64
)

func Load() {
	if err := godotenv.Load(); err != nil {
		log.Println("[config] .env tidak ditemukan, menggunakan environment variable")
	}

	APIID = mustInt("API_ID")
	APIHash = mustStr("API_HASH")
	BotToken = mustStr("BOT_TOKEN")
	MongoURI = mustStr("MONGO_URI")
	MainChannelID = int64(mustInt("MAIN_CHANNEL_ID"))
	MainChannelUsername = getStr("MAIN_CHANNEL_USERNAME", "")
	OwnerID = int64(mustInt("OWNER_ID"))
	BotUsername = getStr("BOT_USERNAME", "")
	OwnerUsername = getStr("OWNER_USERNAME", "")
	OwnerName = getStr("OWNER_NAME", "Owner")
	BotName = getStr("BOT_NAME", "FessBot")
	BotDesc = getStr("BOT_DESC", "Auto Repost Bot")
	FloodSleepThreshold = getInt("FLOOD_SLEEP_THRESHOLD", 60)
	BroadcastDelay = getFloat("BROADCAST_DELAY", 0.05)
}

func mustStr(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("[config] ENV %s wajib diisi", key)
	}
	return v
}

func mustInt(key string) int {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("[config] ENV %s wajib diisi", key)
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Fatalf("[config] ENV %s harus berupa integer: %v", key, err)
	}
	return n
}

func getStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}
