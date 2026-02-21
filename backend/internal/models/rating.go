package models

import (
	"time"
)

type Rating struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	TicketID   uint      `gorm:"not null" json:"ticketId"`
	CustomerID uint      `gorm:"not null" json:"customerId"`
	Score      int       `gorm:"not null" json:"score"`
	Comment    string    `json:"comment"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"createdAt"`
}
