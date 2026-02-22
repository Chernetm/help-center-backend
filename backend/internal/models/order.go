package models

import "time"

type Order struct {
	ID uint `gorm:"primaryKey;autoIncrement"` // auto-increment PK

	OrderID       string `gorm:"uniqueIndex;not null" json:"order_id"` // business order number
	Status        string `json:"status"`
	Urgency       string `json:"urgency"` // Added Urgency field
	Department    string `json:"department"`
	Description   string `json:"description"`
	AdminID       uint64 `json:"admin_id"`
	EstimatedTime *time.Time
	BranchName    *string

	CreatedAt time.Time
	UpdatedAt time.Time
}
