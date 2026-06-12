package utils

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gotd/td/telegram/message"
	"github.com/gotd/td/tg"
	"github.com/youruser/fessbot/internal/config"
	"github.com/youruser/fessbot/internal/db"
)

// ProgressBar membuat progress bar visual
func ProgressBar(val, total, width int) string {
	if total == 0 {
		return strings.Repeat("░", width)
	}
	filled := int(float64(val) / float64(total) * float64(width))
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

// Paginate membagi slice menjadi halaman
func Paginate[T any](data []T, page, pageSize int) ([]T, int) {
	total := len(data)
	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}
	start := page * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}
	return data[start:end], totalPages
}

// BuildCaption membuat caption repost sesuai format standar
func BuildCaption(originalCaption, channelTitle, inviteLink, ownerName string, ownerID int64, botName, botUsername string) string {
	now := time.Now().UTC()
	date := now.Format("02 Jan 2006")
	timeStr := now.Format("15:04 UTC")

	var chLink string
	if inviteLink != "" {
		chLink = fmt.Sprintf("[%s](%s)", channelTitle, inviteLink)
	} else {
		chLink = fmt.Sprintf("**%s**", channelTitle)
	}
	ownerLink := fmt.Sprintf("[%s](tg://user?id=%d)", ownerName, ownerID)
	botLink := fmt.Sprintf("[%s](https://t.me/%s?start=start)", botName, botUsername)

	cap := chLink + "\n"
	if originalCaption != "" {
		cap += "\n" + originalCaption + "\n"
	}
	cap += fmt.Sprintf(
		"\n━━━━━━━━━━━━━━━━━━━━\n"+
			"👤  Owner   :  %s\n"+
			"📅  Tanggal :  %s\n"+
			"🕒  Jam     :  %s\n"+
			"━━━━━━━━━━━━━━━━━━━━\n"+
			"🔁  via %s",
		ownerLink, date, timeStr, botLink,
	)
	return strings.TrimSpace(cap)
}

// CheckMembership memeriksa apakah user adalah anggota channel utama
func CheckMembership(ctx context.Context, api *tg.Client, userID int64) bool {
	// Implementasi bergantung pada library Telegram yang digunakan
	// Contoh dengan gotd/td
	channels := tg.NewChannelsAccessor(api)
	_ = channels
	// Implementasi sebenarnya perlu resolve channel ID ke InputChannel terlebih dahulu
	// Ini placeholder — lihat handlers/repost.go untuk contoh lengkap
	return true
}

// FloodSafe menjalankan operasi dengan retry saat FloodWait
func FloodSafe(ctx context.Context, op func() error, maxRetries int) error {
	for i := 0; i < maxRetries; i++ {
		err := op()
		if err == nil {
			return nil
		}
		// Deteksi FloodWait dari pesan error gotd
		errStr := err.Error()
		if strings.Contains(errStr, "FLOOD_WAIT_") {
			// Parse wait time dari error string
			var wait int
			fmt.Sscanf(errStr, "FLOOD_WAIT_%d", &wait)
			if wait == 0 {
				wait = 5
			}
			wait += 2
			if wait > config.FloodSleepThreshold {
				log.Printf("[FloodSafe] FloodWait %ds terlalu lama — lewati", wait)
				return nil
			}
			log.Printf("[FloodSafe] FloodWait %ds (percobaan %d)", wait, i+1)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(wait) * time.Second):
			}
			continue
		}
		return err
	}
	return fmt.Errorf("melebihi batas retry")
}

// SafeSend mengirim pesan dengan penanganan flood wait
func SafeSend(ctx context.Context, sender *message.Sender, op func(*message.Sender) error) error {
	return FloodSafe(ctx, func() error {
		return op(sender)
	}, 5)
}

// BotSetting helper untuk mendapatkan setting dengan type assertion
func GetSettingBool(key string, def bool) bool {
	return db.GetBotSettingBool(key, def)
}
