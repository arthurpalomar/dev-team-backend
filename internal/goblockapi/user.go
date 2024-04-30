package goblockapi

import (
	"gorm.io/gorm"
	"time"
)

type User struct {
	Id           uint           `json:"id" gorm:"primarykey"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	Address      string         `gorm:"index;not null" json:"address"`
	Hash         string         `gorm:"index;not null" json:"hash"`
	GoogleId     string         `gorm:"index;not null" json:"google_id"`
	DiscordId    string         `gorm:"index;not null" json:"discord_id"`
	TwitterId    string         `gorm:"index" json:"twitter_id"`
	RedditId     string         `gorm:"index;not null" json:"reddit_id"`
	Email        string         `json:"email"`
	Group        uint           `json:"group"`
	WithdrawMin  float64        `json:"withdraw_min"`
	WithdrawMax  float64        `json:"withdraw_max"`
	Accounts     uint           `gorm:"not null;default:1" json:"accounts"`
	Campaigns    uint           `json:"campaigns"`
	Discount     uint           `json:"discount"`
	Rank         uint           `json:"rank"`
	Exp          uint           `json:"exp"`
	Actions      uint           `json:"actions"`
	Upline       uint           `json:"upline"`
	RefUrl       string         `json:"ref_slug"`
	RefCounter   uint           `json:"ref_counter"`
	Utm          string         `json:"utm"`
	Ip           string         `json:"ip"`
	CountryCode  string         `json:"country_code"`
	Referer      string         `json:"referer"`
	Locale       string         `json:"locale"`
	DactEarned   float64        `json:"dact_earned"`
	DimpBuffer   float64        `json:"dimp_buffer"`
	DimpRewards  float64        `json:"dimp_rewards"`
	DimpEarned   float64        `json:"dimp_earned"`
	DimpSpent    float64        `json:"dimp_spent"`
	GoogleName   string         `json:"google_name"`
	GoogleEmail  string         `json:"google_email"`
	DiscordName  string         `json:"discord_name"`
	DiscordEmail string         `json:"discord_email"`
	TwitterName  string         `json:"twitter_name"`
	TwitterEmail string         `json:"twitter_email"`
	RedditName   string         `json:"reddit_name"`
}

type UserData struct {
	ID         uint    `json:"id"`
	Balance    float64 `json:"dimp"`    // up-to-date user $DIMP balance, on Platform
	Rewards    float64 `json:"rewards"` // up-to-date user $DIMP balance in Rewards buffer
	Dact       float64 `json:"dact"`    // up-to-date user $DACT balance, on Platform
	DimpEarned float64 `json:"dimp_earned"`
	DimpSpent  float64 `json:"dimp_spent"`
	Address    string  `json:"address"`
	Hash       string  `json:"hash"`
	RefUrl     string  `json:"ref_slug"`
	Actions    uint    `json:"quests_completed"`
}
