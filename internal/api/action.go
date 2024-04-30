package api

import (
	"test/internal/goblockapi"
	_ "time/tzdata"
)

type ActionParams struct {
	Account  string `json:"account"`                              // Social Network internal ID
	Username string `json:"username" validate:"required,max=150"` // Social Network username
	Id       uint   `json:"id" validate:"required"`               // Task ID
	Status   uint   `json:"status"`                               // Status [0:New, 1:Confirmed, 2:Processed, 9:Rejected]
	ViewTime uint   `json:"view_time"`                            // Time content has been watched in ms
}

const (
	MessageTargetNotification = "notify"
	MessageTargetAlert        = "alert"

	MessageStyleSuccess = "success"
	MessageStyleWarning = "warning"
	MessageStyleError   = "error"
	MessageStyleInfo    = "info"

	MessageTypeCustom                = "custom"
	MessageTypeQuestCompletedDefault = "quest_completed_default"
	MessageTypeQuestRejectedDefault  = "quest_rejected_default"
)

type WsResponseData struct {
	Target        string               `json:"target"` // Websocket message type: 'notify', 'alert', 'sync'
	User          goblockapi.UserData  `json:"user"`
	ReferralStats goblockapi.RefData   `json:"referral_stats"`
	Data          NotificationData     `json:"data"`
	Config        goblockapi.AppConfig `json:"app_config"`
}

type NotificationData struct {
	Id      int     `json:"id"`
	Style   string  `json:"style"`   // Target component style: 'success', 'warning', 'error', 'info'; mostly used with "type": "custom"
	Type    string  `json:"type"`    // Notification type: 'custom', 'quest_completed_default', 'quest_completed_follow', 'quest_completed_view', 'quest_completed_comment', 'quest_completed_quote'
	Message string  `json:"message"` // AI comment
	Url     string  `json:"url"`
	TaskId  uint    `json:"task_id"`
	Dimp    float64 `json:"dimp"`   // Reward, Transaction, etc. $DIMP amount
	Rating  float64 `json:"rating"` // [0;1] AI estimation of user contribution
}
