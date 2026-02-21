package models

import (
	"time"
)

type WorkSession struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	AdminID   uint64     `gorm:"not null" json:"adminId"`
	StartedAt time.Time  `gorm:"autoCreateTime" json:"startedAt"`
	EndedAt   *time.Time `json:"endedAt"`
	Duration  *int       `json:"duration"` // seconds
}
type AuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Action    string    `gorm:"not null" json:"action"`
	AdminID   uint      `gorm:"not null" json:"adminId"`
	TicketID  *uint     `json:"ticketId"`
	OrderID   *string   `json:"orderId"`
	Timestamp time.Time `gorm:"autoCreateTime" json:"timestamp"`
	Hash      string    `gorm:"not null" json:"hash"`
}
