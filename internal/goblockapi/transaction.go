package goblockapi

import "time"

type Transaction struct {
	CreatedAt time.Time `json:"created_at"`
	Txid      string    `json:"txid" gorm:"primaryKey;autoIncrement:false"` // Blockchain transaction id
	UserId    uint      `json:"user_id"`                                    // Executor ID
	AuthorId  uint      `json:"author_id"`                                  // Referrer or Admin ID
	Type      string    `json:"type"`                                       // Type: "in", "out"
	Address   string    `json:"address"`
	Status    uint      `json:"status"` // Status [0: New, 1: Accepted, 9: Rejected]
	Amount    float64   `json:"amount"` // Amount
	Token     string    `json:"token"`  // Token: "dimp", "dact"
	Message   string    `json:"message"`
}
