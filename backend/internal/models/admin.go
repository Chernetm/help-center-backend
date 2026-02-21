package models

import (
	"time"
)

type Admin struct {
	ID              uint64        `gorm:"primaryKey" json:"id"`
	UID             string        `gorm:"size:191;uniqueIndex" json:"uid"`
	Email           string        `gorm:"size:191;uniqueIndex;not null" json:"email"`
	FirstName       string        `gorm:"size:191" json:"firstName"`
	LastName        string        `gorm:"size:191" json:"lastName"`
	Image           string        `json:"image"`
	PhoneNumber     string        `gorm:"size:191" json:"phoneNumber"`
	Address         string        `gorm:"size:255" json:"address"`
	Role            string        `gorm:"size:191" json:"role"`
	Department      string        `gorm:"size:191" json:"department"`
	Status          string        `gorm:"size:50;default:'Active'" json:"status"`
	ActiveTicketQty int           `gorm:"default:0" json:"activeTicketQty"`
	IsOnline        bool          `gorm:"default:false" json:"isOnline"`
	CreatedAt       time.Time     `json:"createdAt"`
	UpdatedAt       time.Time     `json:"updatedAt"`
	WorkSessions    []WorkSession `json:"workSessions,omitempty"`
	Tickets         []Ticket      `gorm:"foreignKey:AdminID" json:"tickets,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// type Admin struct {
// 	ID              uint64 `gorm:"primaryKey"`
// 	UID             string `gorm:"uniqueIndex"`
// 	Email           string `gorm:"uniqueIndex;not null"`
// 	FirstName       string
// 	LastName        string
// 	Phone           string
// 	Address         string
// 	Username        string `gorm:"unique;not null"`
// 	Password        string `gorm:"-"`
// 	Role            string `gorm:"not null"`
// 	Department      string
// 	ActiveTicketQty int       `gorm:"default:0"`
// 	CreatedAt       time.Time `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP"`
// 	IsOnline        bool      `gorm:"default:false"`
// 	Image           string

//	Tickets      []Ticket
//	ChatMessages []ChatMessage `gorm:"polymorphic:Sender;polymorphicValue:admin"`
//
// WorkSessions []WorkSession
//
//		AuditLogs    []AuditLog
//		Orders       []Order
//		MessageReads []MessageReadStatus
//	}
type AgentPerformance struct {
	ID               uint64   `json:"id"`
	Name             string   `json:"name"`
	Department       string   `json:"department"`
	IsOnline         bool     `json:"isOnline"`
	TotalTickets     int      `json:"totalTickets"`
	ClosedTickets    int      `json:"closedTickets"`
	PendingTickets   int      `json:"pendingTickets"`
	AvgRating        *float64 `json:"avgRating"`
	CustomersServed  int      `json:"customersServed"`
	DurationWeekly   int      `json:"durationWeekly"`   // minutes
	DurationMonthly  int      `json:"durationMonthly"`  // minutes
	DurationAnnually int      `json:"durationAnnually"` // minutes
}
