package goblockapi

import "time"

// Tx is a Structure designed to keep the data of internal financial operations
type Tx struct {
	CreatedAt time.Time `json:"created_at"`
	Txid      uint      `json:"txid" gorm:"primaryKey;autoIncrement:true"` // Inner transaction ID
	UserId    uint      `json:"user_id"`                                   // ID of user whose balance is affected by this tx
	AuthorId  uint      `json:"author_id"`                                 // Tx initiator user ID or Admin ID
	Type      string    `json:"type"`                                      // Type: "b":bonus, "r":referral, "s":spent, "e":reward, "y":sync
	Address   string    `json:"address"`
	Status    uint      `json:"status"` // Status [0: New, 1: Accepted, 9: Rejected]
	Amount    float64   `json:"amount"` // Amount
	Token     string    `json:"token"`  // Token: "dimp", "dact"
	Message   string    `json:"message"`
}
