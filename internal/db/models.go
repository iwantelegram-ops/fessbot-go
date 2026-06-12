package db

import "time"

// Partner — channel yang terdaftar di FessBot
type Partner struct {
	ID          int64     `bson:"_id"`
	OwnerID     int64     `bson:"owner_id"`
	OwnerName   string    `bson:"owner_name"`
	ChannelName string    `bson:"channel_name"`
	Username    string    `bson:"username"`
	InviteLink  string    `bson:"invite_link"`
	Paused      bool      `bson:"paused"`
	Reason      string    `bson:"reason"`
	AddedAt     time.Time `bson:"added_at"`
	TotalPosts  int       `bson:"total_posts"`
}

// Post — rekaman repost (partner_msg_id → main_msg_id)
type Post struct {
	ID           string    `bson:"_id"` // "{partner_id}_{partner_msg_id}"
	PartnerID    int64     `bson:"partner_id"`
	PartnerMsgID int       `bson:"partner_msg_id"`
	MainMsgID    int       `bson:"main_msg_id"`
	AddedAt      time.Time `bson:"added_at"`
}

// User — pengguna bot
type User struct {
	ID       int64     `bson:"_id"`
	Joined   bool      `bson:"joined"`
	JoinedAt time.Time `bson:"joined_at,omitempty"`
	LastSeen time.Time `bson:"last_seen"`
	Username string    `bson:"username"`
	Name     string    `bson:"name"`
}

// Blacklist
type Blacklist struct {
	ID    string   `bson:"_id"` // "global"
	Words []string `bson:"words"`
}

// Setting — key-value generik
type Setting struct {
	ID        string      `bson:"_id"`
	Value     interface{} `bson:"value"`
	UpdatedAt time.Time   `bson:"updated_at"`
}

// Maintenance
type Maintenance struct {
	ID        string    `bson:"_id"` // "maintenance"
	Active    bool      `bson:"active"`
	Reason    string    `bson:"reason"`
	UpdatedAt time.Time `bson:"updated_at"`
}

// Activity log
type Activity struct {
	Event     string    `bson:"event"`
	PartnerID int64     `bson:"partner_id,omitempty"`
	TS        time.Time `bson:"ts"`
	Extra     interface{} `bson:"extra,omitempty"`
}

// Broadcast history
type BroadcastRecord struct {
	SenderID int64     `bson:"sender_id"`
	Target   string    `bson:"target"`
	Message  string    `bson:"message"`
	Success  int       `bson:"success"`
	Fail     int       `bson:"fail"`
	SentAt   time.Time `bson:"sent_at"`
}

// NotifSetting — per user
type NotifSetting struct {
	ID            int64 `bson:"_id"`
	RepostNotif   bool  `bson:"repost_notif"`
	BlacklistNotif bool `bson:"blacklist_notif"`
	StatusNotif   bool  `bson:"status_notif"`
}
