package handlers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/dispatcher/handlers"
	"github.com/celestix/gotgproto/dispatcher/handlers/filters"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/markup"
	"github.com/gotd/td/tg"
	"fessbot/internal/config"
	"fessbot/internal/db"
	"fessbot/internal/utils"
	"go.mongodb.org/mongo-driver/bson"
)

var repostLog = log.New(log.Writer(), "[repost] ", log.LstdFlags)

// RegisterRepostHandlers mendaftarkan semua handler repost
func RegisterRepostHandlers(d dispatcher.Dispatcher) {
	d.AddHandlerToGroup(handlers.NewMessage(filters.Message.Channel, onChannelMessage), 1)
	d.AddHandlerToGroup(handlers.NewChatMemberUpdated(onBotAdminChange), 1)
	d.AddHandlerToGroup(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix("confirm_sync_"), onConfirmSync), 1)
	d.AddHandlerToGroup(handlers.NewCallbackQuery(filters.CallbackQuery.Prefix("confirm_nosync_"), onConfirmNoSync), 1)
	d.AddHandlerToGroup(handlers.NewMessage(
		filters.Message.All,
		func(b *ext.Bot, u *ext.Update) error {
			if u.EffectiveMessage == nil {
				return dispatcher.EndGroups
			}
			text := ""
			if m := u.EffectiveMessage.Message; m != nil {
				text = m.GetMessage()
			}
			if text == "/daftarkan" {
				return cmdDaftarkan(b, u)
			}
			return dispatcher.ContinueGroups
		},
	), 1)
}

// onBotAdminChange dipanggil saat status admin bot di channel berubah
func onBotAdminChange(b *ext.Bot, u *ext.Update) error {
	update, ok := u.Update.(*tg.UpdateChannelParticipant)
	if !ok {
		return dispatcher.ContinueGroups
	}

	ctx := context.Background()
	channelID := int64(-1000000000000) - int64(update.ChannelID)
	newP := update.NewParticipant
	oldP := update.PrevParticipant

	isNowAdmin := isAdminParticipant(newP)
	wasAdmin := isAdminParticipant(oldP)

	if isNowAdmin && !wasAdmin {
		// Bot baru dijadikan admin
		inviterID := update.ActorID
		ownerName := "Unknown"
		if update.Actor != nil {
			if u, ok := update.Actor.(*tg.User); ok {
				ownerName = u.FirstName
				if u.LastName != "" {
					ownerName += " " + u.LastName
				}
			}
		}

		channelTitle := ""
		username := ""
		if update.Chat != nil {
			if ch, ok := update.Chat.(*tg.Channel); ok {
				channelTitle = ch.Title
				username = ch.Username
			}
		}

		inviteLink, _ := getOrCreateInviteLink(ctx, b, channelID)
		_ = db.UpsertPartner(channelID, bson.M{
			"owner_id":     inviterID,
			"owner_name":   ownerName,
			"channel_name": channelTitle,
			"username":     username,
			"invite_link":  inviteLink,
			"paused":       true,
			"reason":       "Menunggu konfirmasi owner",
			"added_at":     time.Now().UTC(),
			"total_posts":  0,
		})
		db.LogActivity("partner_added", channelID, bson.M{"owner_id": inviterID})
		repostLog.Printf("Channel terdaftar: %s (%d)", channelTitle, channelID)

		if inviterID != 0 {
			sender := b.Sender.To(message.UserID(inviterID))
			_ = sender.Row(
				markup.CallbackButton("✅ Ya, Aktifkan Sekarang", []byte(fmt.Sprintf("confirm_sync_%d", channelID))),
				markup.CallbackButton("⏸️ Nanti Saja", []byte(fmt.Sprintf("confirm_nosync_%d", channelID))),
			).Text(ctx,
				fmt.Sprintf("🎉 **Channel berhasil terhubung!**\n\n📡 **%s** sudah terdaftar di FessBot.\n\nAktifkan sekarang agar postinganmu mulai di-repost ke channel utama?", channelTitle),
			)
		}

	} else if !isNowAdmin && wasAdmin {
		// Bot dicopot dari admin
		partner, err := db.GetPartner(channelID)
		if err != nil || partner == nil {
			return dispatcher.ContinueGroups
		}
		_ = db.UpsertPartner(channelID, bson.M{"paused": true, "reason": "Bot dicopot dari admin channel"})
		db.LogActivity("bot_removed", channelID, nil)

		if partner.OwnerID != 0 {
			shouldNotify := db.GetNotifSetting(partner.OwnerID, "status_notif", true)
			if shouldNotify {
				sender := b.Sender.To(message.UserID(partner.OwnerID))
				ctx2 := context.Background()
				_ = sender.Text(ctx2,
					fmt.Sprintf("⚠️ **Bot dicopot dari admin channel.**\n\n📡 **%s**\n\nRepost otomatis dihentikan. Tambahkan bot kembali sebagai admin untuk melanjutkan.", partner.ChannelName),
				)
			}
		}
	}

	return dispatcher.EndGroups
}

func isAdminParticipant(p tg.ChannelParticipantClass) bool {
	if p == nil {
		return false
	}
	switch p.(type) {
	case *tg.ChannelParticipantAdmin, *tg.ChannelParticipantCreator:
		return true
	}
	return false
}

// onConfirmSync — user mengkonfirmasi aktifkan channel
func onConfirmSync(b *ext.Bot, u *ext.Update) error {
	cb := u.CallbackQuery
	if cb == nil {
		return dispatcher.EndGroups
	}
	ctx := context.Background()

	data := string(cb.Data)
	channelIDStr := strings.TrimPrefix(data, "confirm_sync_")
	var channelID int64
	fmt.Sscanf(channelIDStr, "%d", &channelID)

	partner, err := db.GetPartner(channelID)
	if err != nil || partner == nil || partner.OwnerID != cb.From.ID {
		_ = b.AnswerCallbackQuery(ctx, cb.GetQueryID(), "Channel tidak ditemukan.", true, "", 0)
		return dispatcher.EndGroups
	}

	_ = db.UpsertPartner(channelID, bson.M{"paused": false, "reason": ""})
	db.LogActivity("partner_activated", channelID, nil)

	_ = b.AnswerCallbackQuery(ctx, cb.GetQueryID(), "Aktif!", false, "", 0)
	repostLog.Printf("Channel %d diaktifkan oleh owner %d", channelID, cb.From.ID)
	return dispatcher.EndGroups
}

// onConfirmNoSync — user memilih tidak aktifkan dulu
func onConfirmNoSync(b *ext.Bot, u *ext.Update) error {
	cb := u.CallbackQuery
	if cb == nil {
		return dispatcher.EndGroups
	}
	ctx := context.Background()

	data := string(cb.Data)
	channelIDStr := strings.TrimPrefix(data, "confirm_nosync_")
	var channelID int64
	fmt.Sscanf(channelIDStr, "%d", &channelID)

	partner, err := db.GetPartner(channelID)
	if err != nil || partner == nil || partner.OwnerID != cb.From.ID {
		_ = b.AnswerCallbackQuery(ctx, cb.GetQueryID(), "Channel tidak ditemukan.", true, "", 0)
		return dispatcher.EndGroups
	}

	_ = db.UpsertPartner(channelID, bson.M{"paused": true, "reason": "Tidak diaktifkan oleh owner"})
	_ = b.AnswerCallbackQuery(ctx, cb.GetQueryID(), "Bisa diaktifkan nanti.", false, "", 0)
	return dispatcher.EndGroups
}

// cmdDaftarkan — daftarkan channel manual via forward
func cmdDaftarkan(b *ext.Bot, u *ext.Update) error {
	msg := u.EffectiveMessage
	if msg == nil {
		return dispatcher.EndGroups
	}
	// Implementasi cek forward_from_chat
	// di gotd/td hal ini dicek dari FwdFrom field
	ctx := context.Background()
	replyMsg := msg.Message
	if replyMsg == nil {
		_ = sendText(ctx, b, msg.Message.PeerID, "📋 **Cara daftarkan channel manual:**\n\n`1.` Forward satu postingan dari channelmu ke sini\n`2.` Reply pesan forward itu dengan `/daftarkan`")
		return dispatcher.EndGroups
	}
	return dispatcher.EndGroups
}

// onChannelMessage — handler utama repost
func onChannelMessage(b *ext.Bot, u *ext.Update) error {
	msg := u.EffectiveMessage
	if msg == nil {
		return dispatcher.ContinueGroups
	}

	ctx := context.Background()
	channelID := u.EffectiveChat.GetID()

	partner, err := db.GetPartner(channelID)
	if err != nil || partner == nil || partner.Paused {
		return dispatcher.ContinueGroups
	}

	// Ambil teks/caption
	captionText := ""
	if m := msg.Message; m != nil {
		captionText = m.GetMessage()
	}

	// Cek blacklist
	if matched := db.ContainsBlacklisted(captionText); matched != "" {
		if partner.OwnerID != 0 {
			shouldNotify := db.GetNotifSetting(partner.OwnerID, "blacklist_notif", true)
			if shouldNotify {
				sender := b.Sender.To(message.UserID(partner.OwnerID))
				_ = sender.Text(ctx,
					fmt.Sprintf("🚫 **Postingan ditolak — kata terlarang**\n\n📡 **%s**\n⚠️ Kata: `%s`\n\nPostingan tidak diteruskan ke channel utama.", partner.ChannelName, matched),
				)
			}
		}
		db.LogActivity("blacklist_blocked", channelID, bson.M{"word": matched})
		return dispatcher.EndGroups
	}

	postNumber, _ := db.CountPostsByPartner(channelID)
	postNumber++

	// Get bot info
	botName := config.BotName
	botUsername := config.BotUsername

	inviteLink := partner.InviteLink
	if inviteLink == "" {
		inviteLink, _ = getOrCreateInviteLink(ctx, b, channelID)
	}

	cap := utils.BuildCaption(
		captionText,
		partner.ChannelName,
		inviteLink,
		partner.OwnerName,
		partner.OwnerID,
		botName,
		botUsername,
	)

	// Build URL ke post asli
	var originalURL string
	if partner.Username != "" {
		originalURL = fmt.Sprintf("https://t.me/%s/%d", partner.Username, getMsgID(msg))
	} else {
		cidStr := strings.Replace(fmt.Sprintf("%d", channelID), "-100", "", 1)
		originalURL = fmt.Sprintf("https://t.me/c/%s/%d", cidStr, getMsgID(msg))
	}

	// Kirim ke channel utama
	mainChannel := message.ChannelID(uint64(-config.MainChannelID - 1000000000000))
	sender := b.Sender.To(mainChannel)

	var sentMsgID int
	err = repostMessage(ctx, b, sender, msg, cap, originalURL, &sentMsgID)
	if err != nil {
		repostLog.Printf("Gagal repost dari %d msg %d: %v", channelID, getMsgID(msg), err)
		db.LogActivity("repost_fail", channelID, nil)
		return dispatcher.EndGroups
	}

	if sentMsgID > 0 {
		_ = db.SavePost(channelID, getMsgID(msg), sentMsgID)
		_ = db.IncrementPartnerPosts(channelID)
		db.LogActivity("repost_success", channelID, bson.M{"main_msg_id": sentMsgID})

		// Lazy check post lama
		go lazyCheckChannel(b, channelID)

		// Notif ke owner
		if partner.OwnerID != 0 {
			shouldNotify := db.GetNotifSetting(partner.OwnerID, "repost_notif", true)
			if shouldNotify {
				mainIDStr := strings.Replace(fmt.Sprintf("%d", config.MainChannelID), "-100", "", 1)
				ownerSender := b.Sender.To(message.UserID(partner.OwnerID))
				_ = ownerSender.Text(ctx,
					fmt.Sprintf("✅ **Postingan berhasil di-repost!**\n\n📡 **%s**\n📦 Repost ke-%d\n🔗 [Lihat di channel utama](https://t.me/c/%s/%d)",
						partner.ChannelName, postNumber, mainIDStr, sentMsgID),
				)
			}
		}
	}

	return dispatcher.EndGroups
}

func repostMessage(ctx context.Context, b *ext.Bot, sender *message.RequestBuilder, msg *ext.Message, cap, originalURL string, sentID *int) error {
	// Buat tombol "Lihat Post Asli"
	row := markup.Row(markup.URL("🔗 Lihat Post Asli", originalURL))
	replyMarkup := markup.InlineKeyboard(row)

	m := msg.Message
	if m == nil {
		return fmt.Errorf("pesan kosong")
	}

	// Cek tipe media
	media := m.GetMedia()
	switch v := media.(type) {
	case *tg.MessageMediaPhoto:
		// Kirim foto
		photo, ok := v.Photo.(*tg.Photo)
		if !ok {
			return fmt.Errorf("foto tidak valid")
		}
		_, err := sender.Media(ctx, message.Photo(message.UploadedPhoto(
			&tg.InputPhoto{ID: photo.ID, AccessHash: photo.AccessHash, FileReference: photo.FileReference},
		))).Caption(cap).ReplyMarkup(replyMarkup).Send()
		if err != nil {
			return err
		}
	case *tg.MessageMediaDocument:
		// Kirim dokumen/video
		doc, ok := v.Document.(*tg.Document)
		if !ok {
			return fmt.Errorf("dokumen tidak valid")
		}
		_, err := sender.Media(ctx, message.Document(
			&tg.InputDocument{ID: doc.ID, AccessHash: doc.AccessHash, FileReference: doc.FileReference},
		)).Caption(cap).ReplyMarkup(replyMarkup).Send()
		if err != nil {
			return err
		}
	default:
		// Teks biasa
		textContent := m.GetMessage()
		if textContent == "" {
			return nil // Abaikan pesan kosong
		}
		allowText := db.GetBotSettingBool("allow_text_repost", true)
		if !allowText {
			return nil
		}
		_, err := sender.Text(ctx, cap)
		if err != nil {
			return err
		}
	}

	return nil
}

// lazyCheckChannel memeriksa post lama dari channel untuk sync delete
func lazyCheckChannel(b *ext.Bot, channelID int64) {
	ctx := context.Background()
	posts, err := db.GetRecentPostsByPartner(channelID, 10)
	if err != nil || len(posts) == 0 {
		return
	}
	for _, post := range posts {
		// Coba ambil pesan dari channel partner
		// Jika tidak bisa diambil (deleted), hapus repost di channel utama
		exists := checkMessageExists(ctx, b, channelID, post.PartnerMsgID)
		if !exists {
			repostLog.Printf("[lazy_check] Post %d sudah dihapus → hapus repost", post.PartnerMsgID)
			processDeleted(ctx, b, channelID, []int{post.PartnerMsgID})
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// checkMessageExists memeriksa apakah pesan masih ada
func checkMessageExists(ctx context.Context, b *ext.Bot, channelID int64, msgID int) bool {
	// Implementasi: panggil messages.GetHistory atau channels.GetMessages
	// Return false jika error atau pesan tidak ditemukan
	_ = ctx
	_ = b
	_ = channelID
	_ = msgID
	return true // placeholder
}

// processDeleted menghapus repost untuk msg yang sudah dihapus
func processDeleted(ctx context.Context, b *ext.Bot, channelID int64, msgIDs []int) {
	if !db.GetBotSettingBool("auto_delete_repost", true) {
		return
	}
	for _, msgID := range msgIDs {
		post, err := db.GetPost(channelID, msgID)
		if err != nil || post == nil {
			continue
		}
		// Hapus dari channel utama
		mainChannel := message.ChannelID(uint64(-config.MainChannelID - 1000000000000))
		sender := b.Sender.To(mainChannel)
		_, err = sender.Delete().Messages(ctx, post.MainMsgID)
		if err != nil {
			repostLog.Printf("Gagal hapus main_msg_id=%d: %v", post.MainMsgID, err)
		} else {
			repostLog.Printf("✅ main_msg_id=%d dihapus (partner=%d msg=%d)", post.MainMsgID, channelID, msgID)
			db.LogActivity("repost_deleted", channelID, bson.M{"main_msg_id": post.MainMsgID})
		}
		_ = db.DeletePost(channelID, msgID)
	}
}

// getOrCreateInviteLink mendapatkan atau membuat invite link channel
func getOrCreateInviteLink(ctx context.Context, b *ext.Bot, channelID int64) (string, error) {
	partner, err := db.GetPartner(channelID)
	if err == nil && partner != nil && partner.InviteLink != "" {
		return partner.InviteLink, nil
	}

	// Export invite link via API
	rawChannelID := uint64(-channelID - 1000000000000)
	link, err := b.ExportChatInviteLink(ctx, rawChannelID)
	if err != nil {
		return "", err
	}
	if link != "" {
		_ = db.UpsertPartner(channelID, bson.M{"invite_link": link})
	}
	return link, nil
}

// UpdateOwnerNameScheduler menjalankan sync nama owner setiap tengah malam UTC
func UpdateOwnerNameScheduler(ctx context.Context, b *ext.Bot) {
	for {
		now := time.Now().UTC()
		nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
		waitDuration := nextMidnight.Sub(now)
		repostLog.Printf("[owner_name_sync] Sync berikutnya dalam %dj %dm",
			int(waitDuration.Hours()), int(waitDuration.Minutes())%60)

		select {
		case <-ctx.Done():
			return
		case <-time.After(waitDuration):
		}

		updateOwnerNames(ctx, b)
	}
}

func updateOwnerNames(ctx context.Context, b *ext.Bot) {
	partners, err := db.GetAllPartners()
	if err != nil || len(partners) == 0 {
		return
	}
	repostLog.Printf("[owner_name_sync] Sync %d partner(s)...", len(partners))
	updated := 0
	for _, p := range partners {
		if p.OwnerID == 0 {
			time.Sleep(3 * time.Second)
			continue
		}
		// Ambil info user terbaru dari Telegram
		// Ini memerlukan implementasi spesifik gotd/td
		// Placeholder:
		_ = b
		_ = ctx
		time.Sleep(3 * time.Second)
	}
	repostLog.Printf("[owner_name_sync] Selesai — %d nama diperbarui.", updated)
}

// getMsgID mengambil ID pesan dari ext.Message
func getMsgID(msg *ext.Message) int {
	if msg == nil || msg.Message == nil {
		return 0
	}
	return msg.Message.GetID()
}

// sendText helper untuk mengirim teks ke peer
func sendText(ctx context.Context, b *ext.Bot, peer tg.PeerClass, text string) error {
	switch p := peer.(type) {
	case *tg.PeerUser:
		_, err := b.Sender.To(message.UserID(p.UserID)).Text(ctx, text)
		return err
	case *tg.PeerChannel:
		_, err := b.Sender.To(message.ChannelID(uint64(p.ChannelID))).Text(ctx, text)
		return err
	case *tg.PeerChat:
		_, err := b.Sender.To(message.ChatID(p.ChatID)).Text(ctx, text)
		return err
	}
	return nil
}
