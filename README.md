# FessBot v2 — Go Edition

Port dari Python (Pyrogram) ke Go menggunakan **gotd/td** + **gotgproto** + **MongoDB**.

## Struktur Proyek

```
fessbot-go/
├── cmd/fessbot/
│   └── main.go              # Entry point
├── internal/
│   ├── config/
│   │   └── config.go        # Load ENV variables
│   ├── db/
│   │   ├── mongo.go         # Koneksi MongoDB & collections
│   │   ├── models.go        # Struct model data
│   │   └── helpers.go       # Fungsi CRUD database
│   ├── handlers/
│   │   ├── repost.go        # Handler repost utama
│   │   ├── start.go         # Handler /start
│   │   └── owner.go         # Panel owner (dashboard, partner, dsb)
│   └── utils/
│       └── utils.go         # Utility functions
├── .env.example
├── go.mod
└── README.md
```

## Fitur

- ✅ Auto-repost foto, video, dan teks dari channel partner ke channel utama
- ✅ Caption otomatis dengan info owner, tanggal, jam, dan link post asli
- ✅ Blacklist kata — tolak postingan dengan kata terlarang
- ✅ FloodWait handling otomatis
- ✅ Auto-hapus repost jika post asli dihapus
- ✅ Notifikasi ke owner (bisa dimatikan)
- ✅ Maintenance mode
- ✅ Pagination untuk daftar partner
- ✅ Dashboard statistik (owner)
- ✅ Scheduler sync nama owner harian (jam 00:00 UTC)
- ✅ Penyimpanan session di SQLite (atau bisa diubah ke MongoDB)

## Setup

### 1. Prasyarat

- Go 1.22+
- MongoDB (Atlas atau self-hosted)

### 2. Clone & Install

```bash
git clone <repo>
cd fessbot-go
go mod tidy
```

### 3. Konfigurasi

```bash
cp .env.example .env
# Edit .env dengan nilai yang sesuai
```

### 4. Jalankan

```bash
go run ./cmd/fessbot
```

Atau build binary:

```bash
go build -o fessbot ./cmd/fessbot
./fessbot
```

## Dependencies

| Package | Fungsi |
|---|---|
| `github.com/gotd/td` | MTProto client Telegram (low-level) |
| `github.com/celestix/gotgproto` | Framework bot di atas gotd/td |
| `go.mongodb.org/mongo-driver` | Driver MongoDB |
| `github.com/joho/godotenv` | Load file .env |

## Perbedaan dari Versi Python

| Aspek | Python (Pyrogram) | Go (gotd/td) |
|---|---|---|
| Session | File `.session` / MongoDB | SQLite (default) / bisa custom |
| Concurrency | asyncio | goroutine native |
| Performance | ~50MB RAM | ~10–15MB RAM |
| Build | Interpreter | Compiled binary |
| Deployment | `python main.py` | `./fessbot` (binary tunggal) |

## Catatan Penting

Beberapa fungsi memerlukan implementasi lengkap dengan API gotd/td:
- `checkUserMembership` — gunakan `channels.GetParticipant`
- `checkMessageExists` — gunakan `channels.GetMessages`
- `getOrCreateInviteLink` — gunakan `messages.ExportChatInviteLink`
- `repostMessage` — sesuaikan dengan tipe media yang tersedia

Lihat dokumentasi [gotd/td](https://github.com/gotd/td) untuk referensi API lengkap.
