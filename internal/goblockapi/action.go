package goblockapi

import "time"

const NewAction = 0
const ActiveAction = 1
const ProcessedAction = 2
const RejectedAction = 9

type Action struct {
	CreatedAt    time.Time `json:"created_at"`
	UserId       uint      `json:"user_id" gorm:"index;primaryKey;autoIncrement:false"` // Executor ID
	AdvertiserId uint      `json:"advertiser_id"`                                       // Advertiser ID
	CampaignId   uint      `json:"campaign_id"`
	TaskId       uint      `json:"task_id" gorm:"index;primaryKey;autoIncrement:false"`
	Rank         uint      `json:"rank"`                // Executor user Rank
	Provider     string    `json:"provider"`            // eg. twitter
	Account      string    `json:"account"`             // Social Network internal ID
	Username     string    `json:"username"`            // Social Network username
	ViewTime     uint      `json:"view_time"`           // time content has been watched in ms
	Status       uint      `json:"status" gorm:"index"` // Status [0:New, 1:Confirmed, 2:Processed, 9:Rejected]
	DimpReward   float64   `json:"dimp_reward"`
	DactReward   float64   `json:"dact_reward"`
}

type QuestData struct {
	TotalCounter uint    `json:"total_counter"`
	TodayCounter uint    `json:"today_counter"`
	DimpTotal    float64 `json:"dimp_total"`
	DimpToday    float64 `json:"dimp_today"`
	DactTotal    float64 `json:"dact_total"`
	DactToday    float64 `json:"dact_today"`
}
