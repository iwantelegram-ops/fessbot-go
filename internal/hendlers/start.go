package handlers

import (
	"context"
	"fmt"
	"log"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/dispatcher/handlers"
	"github.com/celestix/gotgproto/dispatcher/handlers/filters"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/markup"
	"github.com/youruser/fessbot/internal/config"
	"github.com/youruser/fessbot/internal/db"
)

var startLog = log.New(log.Writer(), "[start] ", log.LstdFlags)

// RegisterStartHandlers mendaftarkan handler /start dan recheck join
func RegisterStartHandlers(d dispatcher.Dispatcher) {
	d.AddHandlerToGroup(handlers.NewMessage(
		filters.Message.Commands("start"),
		handleStart,
	), 0)
	d.AddHandlerToGroup(handlers.NewCallbackQuery(
		filters.CallbackQuery.Equal([]byte("recheck_join")),
		handleRecheckJoin,
	), 0)
}

func handleStart(b *ext.Bot, u *ext.Update) error {
	ctx := context.Background()
	msg := u.EffectiveMessage
	if msg == nil {
		return dispatcher.EndGroups
	}

	userID := u.EffectiveUser.GetID()
	userName := u.EffectiveUser.FirstName

	sender := b.Sender.To(message.UserID(userID))

	// Owner dapat tampilan khusus
	if userID == config.OwnerID {
		totalP, _ := db.CountPartners()
		activeP, _ := db.GetActivePartners()
		totalR, _ := db.Posts.CountDocuments(context.Background(), map[string]interface{}{})

		text := fmt.Sprintf(
			"⚡ <b>FessBot v2 — Control Panel</b>\n"+
				"<code>%s</code>\n\n"+
				"📡 Partner   <code>%d</code> aktif · <code>%d</code> total\n"+
				"📦 Repost    <code>%d</code> all-time\n\n"+
				"Semua sistem berjalan normal. 🟢\n"+
				"Gunakan menu di bawah. 👇",
			"────────────────────────────",
			len(activeP), totalP, totalR,
		)
		_ = sender.Text(ctx, text)
		return dispatcher.EndGroups
	}

	// Cek maintenance
	active, reason := db.GetMaintenance()
	if active {
		_ = sender.Text(ctx, fmt.Sprintf("🔧 <b>Bot sedang maintenance</b>\n\n<i>%s</i>\n\nCoba lagi beberapa saat ya! 🙏", reason))
		return dispatcher.EndGroups
	}

	// Cek keanggotaan channel
	joined := checkUserMembership(ctx, b, userID)
	_ = db.UpsertUser(userID, map[string]interface{}{
		"joined":   joined,
		"username": u.EffectiveUser.Username,
		"name":     userName,
	})

	if !joined {
		_ = sender.Row(
			markup.URL("📢 Join Channel Utama", fmt.Sprintf("https://t.me/%s", config.MainChannelUsername)),
			markup.CallbackButton("✅ Sudah Join — Cek Ulang", []byte("recheck_join")),
		).Text(ctx,
			fmt.Sprintf("👋 <b>Halo, %s!</b>\n\nUntuk menggunakan <b>FessBot</b>, kamu perlu join channel utama dulu.\n\nKetuk <b>Join Channel Utama</b> di bawah, lalu ketuk <b>Sudah Join — Cek Ulang</b>. 👇", userName),
		)
		return dispatcher.EndGroups
	}

	// User sudah join
	botLink := fmt.Sprintf("https://t.me/%s?startchannel=true&admin=post_messages+edit_messages+delete_messages+invite_users", config.BotUsername)
	_ = sender.Row(
		markup.URL("➕ Jadikan Bot Admin di Channel", botLink),
	).Text(ctx,
		fmt.Sprintf("⚡ <b>Halo, %s!</b>\n\n<b>FessBot</b> otomatis meneruskan foto &amp; video dari channelmu ke channel utama.\n\n"+
			"<b>Cara setup:</b>\n1️⃣  Tambahkan bot sebagai <b>Admin</b> di channelmu\n"+
			"2️⃣  Channel otomatis terdaftar\n3️⃣  Konten di-repost real-time ✅\n\n"+
			"Tekan tombol di bawah untuk menambahkan bot sebagai admin. 👇", userName),
	)
	startLog.Printf("User %d (%s) membuka /start", userID, userName)
	return dispatcher.EndGroups
}

func handleRecheckJoin(b *ext.Bot, u *ext.Update) error {
	ctx := context.Background()
	cb := u.CallbackQuery
	if cb == nil {
		return dispatcher.EndGroups
	}
	userID := cb.From.GetID()
	joined := checkUserMembership(ctx, b, userID)

	if !joined {
		_ = b.AnswerCallbackQuery(ctx, cb.GetQueryID(), "Kamu belum join channel utama!", true, "", 0)
		return dispatcher.EndGroups
	}

	_ = db.UpsertUser(userID, map[string]interface{}{"joined": true})
	_ = b.AnswerCallbackQuery(ctx, cb.GetQueryID(), "✅ Berhasil! Kamu sudah terdaftar.", false, "", 0)
	return dispatcher.EndGroups
}

// checkUserMembership memeriksa apakah user adalah member channel utama
func checkUserMembership(ctx context.Context, b *ext.Bot, userID int64) bool {
	// Implementasi sebenarnya memerlukan panggilan channels.GetParticipant
	// Placeholder:
	_ = ctx
	_ = b
	_ = userID
	return true
}
