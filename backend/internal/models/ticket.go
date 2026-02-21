package models

import (
	"time"
)

type Ticket struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	CustomerID uint      `gorm:"not null" json:"customerId"`
	AdminID    *uint64   `json:"adminId"` // nullable
	CaseID     uint      `gorm:"not null" json:"caseId"`
	Status     string    `gorm:"default:'open'" json:"status"` // open, assigned, waiting, closed
	Priority   string    `gorm:"default:'normal'" json:"priority"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updatedAt"`

	Customer Customer      `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	Chats    []ChatMessage `gorm:"foreignKey:TicketID" json:"chats,omitempty"`
	Admin    *Admin        `gorm:"foreignKey:AdminID" json:"admin,omitempty"`
	Ratings  []Rating      `gorm:"foreignKey:TicketID" json:"ratings,omitempty"`
	Case     Case          `gorm:"foreignKey:CaseID" json:"case,omitempty"`
}

type TicketMetrics struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	TicketID        uint       `gorm:"unique;not null" json:"ticketId"`
	FirstResponseAt *time.Time `json:"firstResponseAt"`
	ResolvedAt      *time.Time `json:"resolvedAt"`
	SlaBreached     bool       `gorm:"default:false" json:"slaBreached"`
}
