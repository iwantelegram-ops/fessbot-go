package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/celestix/gotgproto"
	"github.com/celestix/gotgproto/sessionMaker"
	"fessbot/internal/config"
	"fessbot/internal/db"
	"fessbot/internal/handlers"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("🤖 FessBot v2 (Go) starting...")

	// Load konfigurasi dari .env
	config.Load()

	// Koneksi ke MongoDB
	db.Connect()
	defer db.Disconnect()

	// Buat context yang bisa dibatalkan
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Inisialisasi bot client via gotgproto
	client, err := gotgproto.NewClient(
		config.APIID,
		config.APIHash,
		gotgproto.ClientTypeBot(config.BotToken),
		&gotgproto.ClientOpts{
			Session: sessionMaker.SqlSession("fessbot_session"),
			// Gunakan MongoDB session storage jika diperlukan:
			// Session: NewMongoSession(db.Sessions),
		},
	)
	if err != nil {
		log.Fatalf("Gagal membuat client Telegram: %v", err)
	}

	// Daftarkan semua handler
	dp := client.Dispatcher
	handlers.RegisterStartHandlers(dp)
	handlers.RegisterRepostHandlers(dp)
	handlers.RegisterOwnerHandlers(dp)

	// Jalankan scheduler owner name sync di background
	go handlers.UpdateOwnerNameScheduler(ctx, client.Bot)

	log.Println("🤖 FessBot v2 berjalan. Tekan Ctrl+C untuk berhenti.")

	// Jalankan bot (blocking hingga ctx dibatalkan)
	if err := client.Idle(); err != nil {
		log.Printf("Bot berhenti: %v", err)
	}

	log.Println("🤖 FessBot v2 dihentikan.")
}
