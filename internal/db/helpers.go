package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var bg = context.Background

// ══════════════════════════════════════════════════════════
//  PARTNERS
// ══════════════════════════════════════════════════════════

func GetPartner(channelID int64) (*Partner, error) {
	var p Partner
	err := Partners.FindOne(bg(), bson.M{"_id": channelID}).Decode(&p)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &p, err
}

func UpsertPartner(channelID int64, data bson.M) error {
	_, err := Partners.UpdateOne(
		bg(),
		bson.M{"_id": channelID},
		bson.M{"$set": data},
		options.Update().SetUpsert(true),
	)
	return err
}

func RemovePartner(channelID int64) error {
	_, err := Partners.DeleteOne(bg(), bson.M{"_id": channelID})
	return err
}

func IncrementPartnerPosts(channelID int64) error {
	_, err := Partners.UpdateOne(bg(), bson.M{"_id": channelID}, bson.M{"$inc": bson.M{"total_posts": 1}})
	return err
}

func GetAllPartners() ([]Partner, error) {
	opts := options.Find().SetSort(bson.M{"total_posts": -1})
	cur, err := Partners.Find(bg(), bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(bg())
	var result []Partner
	return result, cur.All(bg(), &result)
}

func GetActivePartners() ([]Partner, error) {
	cur, err := Partners.Find(bg(), bson.M{"paused": false})
	if err != nil {
		return nil, err
	}
	defer cur.Close(bg())
	var result []Partner
	return result, cur.All(bg(), &result)
}

func GetPartnersByOwner(ownerID int64) ([]Partner, error) {
	cur, err := Partners.Find(bg(), bson.M{"owner_id": ownerID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(bg())
	var result []Partner
	return result, cur.All(bg(), &result)
}

func CountPartners() (int64, error) {
	return Partners.CountDocuments(bg(), bson.M{})
}

func SearchPartners(query string) ([]Partner, error) {
	regex := bson.M{"$regex": query, "$options": "i"}
	filter := bson.M{"$or": []bson.M{
		{"channel_name": regex},
		{"username": regex},
	}}
	cur, err := Partners.Find(bg(), filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(bg())
	var result []Partner
	return result, cur.All(bg(), &result)
}

func GetTopPartners(limit int) ([]Partner, error) {
	opts := options.Find().SetSort(bson.M{"total_posts": -1}).SetLimit(int64(limit))
	cur, err := Partners.Find(bg(), bson.M{"paused": false}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(bg())
	var result []Partner
	return result, cur.All(bg(), &result)
}

// ══════════════════════════════════════════════════════════
//  POSTS
// ══════════════════════════════════════════════════════════

func postID(partnerID int64, msgID int) string {
	return fmt.Sprintf("%d_%d", partnerID, msgID)
}

func SavePost(partnerID int64, partnerMsgID, mainMsgID int) error {
	_, err := Posts.InsertOne(bg(), Post{
		ID:           postID(partnerID, partnerMsgID),
		PartnerID:    partnerID,
		PartnerMsgID: partnerMsgID,
		MainMsgID:    mainMsgID,
		AddedAt:      time.Now().UTC(),
	})
	return err
}

func GetPost(partnerID int64, partnerMsgID int) (*Post, error) {
	var p Post
	err := Posts.FindOne(bg(), bson.M{"_id": postID(partnerID, partnerMsgID)}).Decode(&p)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &p, err
}

func DeletePost(partnerID int64, partnerMsgID int) error {
	_, err := Posts.DeleteOne(bg(), bson.M{"_id": postID(partnerID, partnerMsgID)})
	return err
}

func GetAllTrackedPosts() ([]Post, error) {
	cur, err := Posts.Find(bg(), bson.M{}, options.Find().SetProjection(bson.M{
		"partner_id": 1, "partner_msg_id": 1, "main_msg_id": 1,
	}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(bg())
	var result []Post
	return result, cur.All(bg(), &result)
}

func CountPostsByPartner(partnerID int64) (int64, error) {
	return Posts.CountDocuments(bg(), bson.M{"partner_id": partnerID})
}

func GetPostsToday() (int64, error) {
	start := time.Now().UTC().Truncate(24 * time.Hour)
	return Posts.CountDocuments(bg(), bson.M{"added_at": bson.M{"$gte": start}})
}

func GetPostsThisWeek() (int64, error) {
	start := time.Now().UTC().Add(-7 * 24 * time.Hour)
	return Posts.CountDocuments(bg(), bson.M{"added_at": bson.M{"$gte": start}})
}

func GetPostsThisMonth() (int64, error) {
	start := time.Now().UTC().Add(-30 * 24 * time.Hour)
	return Posts.CountDocuments(bg(), bson.M{"added_at": bson.M{"$gte": start}})
}

func GetRecentPostsByPartner(partnerID int64, limit int) ([]Post, error) {
	opts := options.Find().SetSort(bson.M{"added_at": -1}).SetLimit(int64(limit))
	cur, err := Posts.Find(bg(), bson.M{"partner_id": partnerID}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(bg())
	var result []Post
	return result, cur.All(bg(), &result)
}

// ══════════════════════════════════════════════════════════
//  USERS
// ══════════════════════════════════════════════════════════

func GetUser(userID int64) (*User, error) {
	var u User
	err := Users.FindOne(bg(), bson.M{"_id": userID}).Decode(&u)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &u, err
}

func UpsertUser(userID int64, data bson.M) error {
	_, err := Users.UpdateOne(
		bg(),
		bson.M{"_id": userID},
		bson.M{"$set": data},
		options.Update().SetUpsert(true),
	)
	return err
}

func IsJoined(userID int64) bool {
	u, err := GetUser(userID)
	return err == nil && u != nil && u.Joined
}

func GetAllUserIDs() ([]int64, error) {
	cur, err := Users.Find(bg(), bson.M{}, options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(bg())
	var ids []int64
	for cur.Next(bg()) {
		var u struct {
			ID int64 `bson:"_id"`
		}
		if err := cur.Decode(&u); err == nil {
			ids = append(ids, u.ID)
		}
	}
	return ids, nil
}

func CountUsers() (int64, error) {
	return Users.CountDocuments(bg(), bson.M{})
}

func CountActiveUsers() (int64, error) {
	return Users.CountDocuments(bg(), bson.M{"joined": true})
}

// ══════════════════════════════════════════════════════════
//  BLACKLIST
// ══════════════════════════════════════════════════════════

func GetBlacklist() ([]string, error) {
	var doc Blacklist
	err := BlacklistCol.FindOne(bg(), bson.M{"_id": "global"}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return doc.Words, err
}

func AddBlacklist(word string) error {
	word = strings.ToLower(strings.TrimSpace(word))
	_, err := BlacklistCol.UpdateOne(
		bg(),
		bson.M{"_id": "global"},
		bson.M{"$addToSet": bson.M{"words": word}},
		options.Update().SetUpsert(true),
	)
	return err
}

func RemoveBlacklist(word string) error {
	word = strings.ToLower(strings.TrimSpace(word))
	_, err := BlacklistCol.UpdateOne(
		bg(),
		bson.M{"_id": "global"},
		bson.M{"$pull": bson.M{"words": word}},
	)
	return err
}

func ContainsBlacklisted(text string) string {
	words, _ := GetBlacklist()
	lower := strings.ToLower(text)
	for _, w := range words {
		if strings.Contains(lower, w) {
			return w
		}
	}
	return ""
}

// ══════════════════════════════════════════════════════════
//  MAINTENANCE
// ══════════════════════════════════════════════════════════

func SetMaintenance(active bool, reason string) error {
	_, err := SettingsCol.UpdateOne(
		bg(),
		bson.M{"_id": "maintenance"},
		bson.M{"$set": bson.M{"active": active, "reason": reason, "updated_at": time.Now().UTC()}},
		options.Update().SetUpsert(true),
	)
	return err
}

func GetMaintenance() (bool, string) {
	var doc struct {
		Active bool   `bson:"active"`
		Reason string `bson:"reason"`
	}
	if err := SettingsCol.FindOne(bg(), bson.M{"_id": "maintenance"}).Decode(&doc); err != nil {
		return false, ""
	}
	return doc.Active, doc.Reason
}

// ══════════════════════════════════════════════════════════
//  BOT SETTINGS
// ══════════════════════════════════════════════════════════

func GetBotSetting(key string, def interface{}) interface{} {
	var doc Setting
	if err := SettingsCol.FindOne(bg(), bson.M{"_id": "setting_" + key}).Decode(&doc); err != nil {
		return def
	}
	if doc.Value == nil {
		return def
	}
	return doc.Value
}

func GetBotSettingBool(key string, def bool) bool {
	v := GetBotSetting(key, def)
	if b, ok := v.(bool); ok {
		return b
	}
	return def
}

func SetBotSetting(key string, value interface{}) error {
	_, err := SettingsCol.UpdateOne(
		bg(),
		bson.M{"_id": "setting_" + key},
		bson.M{"$set": bson.M{"value": value, "updated_at": time.Now().UTC()}},
		options.Update().SetUpsert(true),
	)
	return err
}

// ══════════════════════════════════════════════════════════
//  ACTIVITY LOG
// ══════════════════════════════════════════════════════════

func LogActivity(event string, partnerID int64, extra interface{}) {
	doc := bson.M{
		"event":      event,
		"partner_id": partnerID,
		"ts":         time.Now().UTC(),
	}
	if extra != nil {
		doc["extra"] = extra
	}
	_, _ = ActivityCol.InsertOne(bg(), doc)
}

func GetRecentActivity(limit int, partnerID int64) ([]bson.M, error) {
	filter := bson.M{}
	if partnerID != 0 {
		filter["partner_id"] = partnerID
	}
	opts := options.Find().SetSort(bson.M{"ts": -1}).SetLimit(int64(limit))
	cur, err := ActivityCol.Find(bg(), filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(bg())
	var result []bson.M
	return result, cur.All(bg(), &result)
}

// ══════════════════════════════════════════════════════════
//  NOTIFICATION SETTINGS
// ══════════════════════════════════════════════════════════

func GetNotifSetting(userID int64, key string, def bool) bool {
	var doc bson.M
	if err := NotifCol.FindOne(bg(), bson.M{"_id": userID}).Decode(&doc); err != nil {
		return def
	}
	if v, ok := doc[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}

func SetNotifSetting(userID int64, key string, value bool) error {
	_, err := NotifCol.UpdateOne(
		bg(),
		bson.M{"_id": userID},
		bson.M{"$set": bson.M{key: value}},
		options.Update().SetUpsert(true),
	)
	return err
}

// ══════════════════════════════════════════════════════════
//  BROADCAST HISTORY
// ══════════════════════════════════════════════════════════

func SaveBroadcast(senderID int64, target, message string, success, fail int) error {
	if len(message) > 200 {
		message = message[:200]
	}
	_, err := BroadcastCol.InsertOne(bg(), BroadcastRecord{
		SenderID: senderID,
		Target:   target,
		Message:  message,
		Success:  success,
		Fail:     fail,
		SentAt:   time.Now().UTC(),
	})
	return err
}

func GetBroadcastHistory(limit int) ([]BroadcastRecord, error) {
	opts := options.Find().SetSort(bson.M{"sent_at": -1}).SetLimit(int64(limit))
	cur, err := BroadcastCol.Find(bg(), bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(bg())
	var result []BroadcastRecord
	return result, cur.All(bg(), &result)
}
