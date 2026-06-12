package handlers

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/celestix/gotgproto/dispatcher"
	"github.com/celestix/gotgproto/dispatcher/handlers"
	"github.com/celestix/gotgproto/dispatcher/handlers/filters"
	"github.com/celestix/gotgproto/ext"
	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/markup"
	"fessbot/internal/config"
	"fessbot/internal/db"
	"fessbot/internal/utils"
	"go.mongodb.org/mongo-driver/bson"
)

var ownerLog = log.New(log.Writer(), "[owner] ", log.LstdFlags)

// RegisterOwnerHandlers mendaftarkan semua handler owner
func RegisterOwnerHandlers(d dispatcher.Dispatcher) {
	d.AddHandlerToGroup(handlers.NewMessage(
		filters.Message.Private,
		handleOwnerText,
	), 2)
	d.AddHandlerToGroup(handlers.NewCallbackQuery(
		filters.CallbackQuery.Prefix("owner_"),
		handleOwnerCallback,
	), 2)
}

func isOwner(userID int64) bool {
	return userID == config.OwnerID
}

func handleOwnerText(b *ext.Bot, u *ext.Update) error {
	msg := u.EffectiveMessage
	if msg == nil || !isOwner(u.EffectiveUser.GetID()) {
		return dispatcher.ContinueGroups
	}

	ctx := context.Background()
	userID := u.EffectiveUser.GetID()
	sender := b.Sender.To(message.UserID(userID))
	text := ""
	if m := msg.Message; m != nil {
		text = m.GetMessage()
	}

	switch text {
	case "📊 Dashboard":
		return showDashboard(ctx, sender, b)
	case "📋 Partner":
		return showPartnerList(ctx, sender, b, 0)
	case "📣 Broadcast":
		return showBroadcastInfo(ctx, sender)
	case "🔧 Tools":
		return showTools(ctx, sender)
	case "📝 Aktivitas":
		return showActivity(ctx, sender)
	case "⚙️ Pengaturan":
		return showSettings(ctx, sender)
	case "🚫 Blacklist":
		return showBlacklist(ctx, sender)
	case "🔧 Maintenance":
		return showMaintenance(ctx, sender)
	case "🏠 Menu Utama":
		return showMainMenu(ctx, sender)
	case "🟢 Nonaktifkan Maintenance":
		_ = db.SetMaintenance(false, "")
		_ = sender.Text(ctx, "✅ Maintenance dinonaktifkan.")
		return dispatcher.EndGroups
	case "🔴 Aktifkan Maintenance":
		_ = sender.Text(ctx, "Kirimkan alasan maintenance (atau ketik `-` untuk tidak ada alasan):")
		// State machine sederhana bisa ditambahkan untuk menangkap input berikutnya
		return dispatcher.EndGroups
	}
	return dispatcher.ContinueGroups
}

func handleOwnerCallback(b *ext.Bot, u *ext.Update) error {
	cb := u.CallbackQuery
	if cb == nil || !isOwner(cb.From.GetID()) {
		return dispatcher.EndGroups
	}

	ctx := context.Background()
	data := string(cb.Data)
	_ = b.AnswerCallbackQuery(ctx, cb.GetQueryID(), "", false, "", 0)

	sender := b.Sender.To(message.UserID(cb.From.GetID()))

	if strings.HasPrefix(data, "owner_partner_page_") {
		var page int
		fmt.Sscanf(strings.TrimPrefix(data, "owner_partner_page_"), "%d", &page)
		return showPartnerList(ctx, sender, b, page)
	}
	if strings.HasPrefix(data, "owner_partner_detail_") {
		var channelID int64
		fmt.Sscanf(strings.TrimPrefix(data, "owner_partner_detail_"), "%d", &channelID)
		return showPartnerDetail(ctx, sender, channelID)
	}
	if strings.HasPrefix(data, "owner_partner_pause_") {
		var channelID int64
		fmt.Sscanf(strings.TrimPrefix(data, "owner_partner_pause_"), "%d", &channelID)
		return togglePartnerPause(ctx, sender, channelID, true)
	}
	if strings.HasPrefix(data, "owner_partner_resume_") {
		var channelID int64
		fmt.Sscanf(strings.TrimPrefix(data, "owner_partner_resume_"), "%d", &channelID)
		return togglePartnerPause(ctx, sender, channelID, false)
	}
	if strings.HasPrefix(data, "owner_partner_remove_") {
		var channelID int64
		fmt.Sscanf(strings.TrimPrefix(data, "owner_partner_remove_"), "%d", &channelID)
		return removePartner(ctx, sender, channelID)
	}

	return dispatcher.EndGroups
}

func showDashboard(ctx context.Context, sender *message.RequestBuilder, b *ext.Bot) error {
	totalP, _ := db.CountPartners()
	activeP, _ := db.GetActivePartners()
	totalU, _ := db.CountUsers()
	activeU, _ := db.CountActiveUsers()
	today, _ := db.GetPostsToday()
	week, _ := db.GetPostsThisWeek()
	month, _ := db.GetPostsThisMonth()
	topPartners, _ := db.GetTopPartners(3)

	topStr := ""
	for i, p := range topPartners {
		topStr += fmt.Sprintf("  %d. %s — %d post\n", i+1, p.ChannelName, p.TotalPosts)
	}

	maintActive, maintReason := db.GetMaintenance()
	maintStr := "🟢 Normal"
	if maintActive {
		maintStr = fmt.Sprintf("🔴 Maintenance (%s)", maintReason)
	}

	text := fmt.Sprintf(
		"📊 <b>Dashboard FessBot</b>\n<code>%s</code>\n\n"+
			"👥 <b>Users</b>\n  Total: %d | Aktif: %d\n\n"+
			"📡 <b>Partner Channel</b>\n  Total: %d | Aktif: %d\n\n"+
			"📦 <b>Repost</b>\n  Hari ini: %d | Minggu ini: %d | Bulan ini: %d\n\n"+
			"🏆 <b>Top Channel</b>\n%s\n"+
			"⚙️ <b>Status</b>: %s",
		"────────────────────────────",
		totalU, activeU,
		totalP, len(activeP),
		today, week, month,
		topStr, maintStr,
	)
	_, err := sender.Text(ctx, text)
	return err
}

func showPartnerList(ctx context.Context, sender *message.RequestBuilder, b *ext.Bot, page int) error {
	allPartners, err := db.GetAllPartners()
	if err != nil {
		_, err = sender.Text(ctx, "❌ Gagal memuat daftar partner.")
		return err
	}

	chunk, totalPages := utils.Paginate(allPartners, page, 8)

	text := fmt.Sprintf("📋 <b>Partner Channel</b> — halaman %d/%d\n<code>%s</code>\n", page+1, totalPages, "────────────────")
	for _, p := range chunk {
		icon := "▶️"
		if p.Paused {
			icon = "⏸"
		}
		text += fmt.Sprintf("\n%s <b>%s</b> | %d post", icon, p.ChannelName, p.TotalPosts)
	}

	// Build inline keyboard
	var rows []markup.Row
	for _, p := range chunk {
		rows = append(rows, markup.Row(
			markup.CallbackButton(
				fmt.Sprintf("📋 %s", trimStr(p.ChannelName, 30)),
				[]byte(fmt.Sprintf("owner_partner_detail_%d", p.ID)),
			),
		))
	}

	// Nav buttons
	var nav []markup.Button
	if page > 0 {
		nav = append(nav, markup.CallbackButton("◀️", []byte(fmt.Sprintf("owner_partner_page_%d", page-1))))
	}
	nav = append(nav, markup.CallbackButton(fmt.Sprintf("%d/%d", page+1, totalPages), []byte("noop")))
	if page < totalPages-1 {
		nav = append(nav, markup.CallbackButton("▶️", []byte(fmt.Sprintf("owner_partner_page_%d", page+1))))
	}
	if len(nav) > 0 {
		rows = append(rows, markup.Row(nav...))
	}

	_, err = sender.Row(rows...).Text(ctx, text)
	return err
}

func showPartnerDetail(ctx context.Context, sender *message.RequestBuilder, channelID int64) error {
	p, err := db.GetPartner(channelID)
	if err != nil || p == nil {
		_, err = sender.Text(ctx, "❌ Channel tidak ditemukan.")
		return err
	}

	postCount, _ := db.CountPostsByPartner(channelID)
	status := "▶️ Aktif"
	if p.Paused {
		status = "⏸ Paused"
	}

	text := fmt.Sprintf(
		"📡 <b>%s</b>\n<code>%s</code>\n\n"+
			"🆔 ID: <code>%d</code>\n"+
			"👤 Owner: %s\n"+
			"📦 Total repost: %d\n"+
			"📌 Status: %s\n"+
			"📅 Terdaftar: %s",
		p.ChannelName, "────────────────────",
		p.ID,
		p.OwnerName,
		postCount,
		status,
		p.AddedAt.Format("02 Jan 2006 15:04 UTC"),
	)

	var actionBtn markup.Button
	if p.Paused {
		actionBtn = markup.CallbackButton("▶️ Aktifkan", []byte(fmt.Sprintf("owner_partner_resume_%d", channelID)))
	} else {
		actionBtn = markup.CallbackButton("⏸ Pause", []byte(fmt.Sprintf("owner_partner_pause_%d", channelID)))
	}

	_, err = sender.Row(
		markup.Row(actionBtn),
		markup.Row(markup.CallbackButton("🗑 Hapus Channel", []byte(fmt.Sprintf("owner_partner_remove_%d", channelID)))),
		markup.Row(markup.CallbackButton("« Kembali", []byte("owner_partner_page_0"))),
	).Text(ctx, text)
	return err
}

func togglePartnerPause(ctx context.Context, sender *message.RequestBuilder, channelID int64, paused bool) error {
	action := "dijeda"
	if !paused {
		action = "diaktifkan kembali"
	}
	_ = db.UpsertPartner(channelID, bson.M{"paused": paused, "reason": ""})
	_, err := sender.Text(ctx, fmt.Sprintf("✅ Channel %s.", action))
	return err
}

func removePartner(ctx context.Context, sender *message.RequestBuilder, channelID int64) error {
	_ = db.RemovePartner(channelID)
	db.LogActivity("partner_removed", channelID, nil)
	_, err := sender.Text(ctx, "🗑 Channel dihapus dari daftar partner.")
	return err
}

func showBroadcastInfo(ctx context.Context, sender *message.RequestBuilder) error {
	_, err := sender.Text(ctx,
		"📣 <b>Broadcast</b>\n\nGunakan command:\n"+
			"`/broadcast all <pesan>` — kirim ke semua user\n"+
			"`/broadcast owners <pesan>` — kirim ke semua owner channel\n\n"+
			"Contoh:\n<code>/broadcast all Halo semua!</code>",
	)
	return err
}

func showTools(ctx context.Context, sender *message.RequestBuilder) error {
	_, err := sender.Row(
		markup.Row(markup.CallbackButton("🚫 Blacklist", []byte("owner_show_blacklist"))),
		markup.Row(markup.CallbackButton("🔧 Maintenance", []byte("owner_show_maintenance"))),
	).Text(ctx, "🔧 <b>Tools</b>\n\nPilih fitur:")
	return err
}

func showBlacklist(ctx context.Context, sender *message.RequestBuilder) error {
	words, _ := db.GetBlacklist()
	text := "🚫 <b>Daftar Blacklist</b>\n\n"
	if len(words) == 0 {
		text += "_Belum ada kata yang diblacklist._"
	} else {
		for i, w := range words {
			text += fmt.Sprintf("%d. <code>%s</code>\n", i+1, w)
		}
	}
	text += "\n\nGunakan:\n`/addbl <kata>` — tambah kata\n`/rmbl <kata>` — hapus kata"
	_, err := sender.Text(ctx, text)
	return err
}

func showMaintenance(ctx context.Context, sender *message.RequestBuilder) error {
	active, reason := db.GetMaintenance()
	status := "🟢 Normal"
	if active {
		status = fmt.Sprintf("🔴 Maintenance: %s", reason)
	}
	_, err := sender.Row(
		markup.Row(markup.CallbackButton("🔴 Aktifkan Maintenance", []byte("owner_maintenance_on"))),
		markup.Row(markup.CallbackButton("🟢 Nonaktifkan Maintenance", []byte("owner_maintenance_off"))),
	).Text(ctx, fmt.Sprintf("🔧 <b>Maintenance Mode</b>\n\nStatus: %s", status))
	return err
}

func showActivity(ctx context.Context, sender *message.RequestBuilder) error {
	activities, _ := db.GetRecentActivity(10, 0)
	text := "📝 <b>Aktivitas Terbaru</b>\n\n"
	for _, a := range activities {
		event := fmt.Sprintf("%v", a["event"])
		ts := ""
		if t, ok := a["ts"].(interface{ Format(string) string }); ok {
			ts = t.Format("02/01 15:04")
		}
		text += fmt.Sprintf("• <code>%s</code> — %s\n", event, ts)
	}
	if len(activities) == 0 {
		text += "_Belum ada aktivitas._"
	}
	_, err := sender.Text(ctx, text)
	return err
}

func showSettings(ctx context.Context, sender *message.RequestBuilder) error {
	autoDelete := db.GetBotSettingBool("auto_delete_repost", true)
	allowText := db.GetBotSettingBool("allow_text_repost", true)

	autoDeleteStr := "✅ Aktif"
	if !autoDelete {
		autoDeleteStr = "❌ Nonaktif"
	}
	allowTextStr := "✅ Aktif"
	if !allowText {
		allowTextStr = "❌ Nonaktif"
	}

	_, err := sender.Row(
		markup.Row(markup.CallbackButton(fmt.Sprintf("Auto Delete Repost: %s", autoDeleteStr), []byte("owner_toggle_auto_delete"))),
		markup.Row(markup.CallbackButton(fmt.Sprintf("Repost Teks: %s", allowTextStr), []byte("owner_toggle_text_repost"))),
	).Text(ctx, "⚙️ <b>Pengaturan Bot</b>")
	return err
}

func showMainMenu(ctx context.Context, sender *message.RequestBuilder) error {
	_, err := sender.Text(ctx, "🏠 Kembali ke menu utama.")
	return err
}

// trimStr memotong string jika terlalu panjang
func trimStr(s string, max int) string {
	if len([]rune(s)) > max {
		return string([]rune(s)[:max]) + "…"
	}
	return s
}

// ownerLog dipakai untuk menghindari unused variable
var _ = ownerLog
