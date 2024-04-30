package goblockapi

import "time"

// Ref is a Structure designed to store referral relations
type Ref struct {
	CreatedAt     time.Time `json:"created_at"`
	UserId        uint      `json:"user_id" gorm:"primaryKey;autoIncrement:false"`   // ID of user whose balance is affected by this tx
	AuthorId      uint      `json:"author_id" gorm:"primaryKey;autoIncrement:false"` // Rewarding initiator's user ID or Admin ID
	AuthorUpline  uint      `json:"author_upline"`                                   // How many own referrals your referral has
	GoogleName    string    `json:"google_name"`
	AuthorAddress string    `json:"author_address"`
	AuthorEmail   string    `json:"author_email"`
	Lvl           uint      `json:"lvl"` // Only used for referral tx, referrer level
	Dimp          float64   `json:"dimp"`
	Dact          float64   `json:"dact"`
}

type RefData struct {
	TotalCounter    uint    `json:"total_counter"`
	LlvOneCounter   uint    `json:"lvl_one_counter"`
	LlvTwoCounter   uint    `json:"lvl_two_counter"`
	LlvThreeCounter uint    `json:"lvl_three_counter"`
	DimpTotal       float64 `json:"dimp_total"`
	DimpLvlOne      float64 `json:"dimp_lvl_one"`
	DimpLvlTwo      float64 `json:"dimp_lvl_two"`
	DimpLvlThree    float64 `json:"dimp_lvl_three"`
	DactTotal       float64 `json:"dact_total"`
	DactLvlOne      float64 `json:"dact_lvl_one"`
	DactLvlTwo      float64 `json:"dact_lvl_two"`
	DactLvlThree    float64 `json:"dact_lvl_three"`
}
