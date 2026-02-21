package models

import (
	"time"
)

type ChatMessage struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	TicketID   uint   `gorm:"not null" json:"ticketId"`
	SenderID   uint64 `json:"senderId"`
	SenderType string `gorm:"not null" json:"senderType"` // "admin" | "customer"

	// Text message (optional if media)
	Message string `json:"message"`

	// Cloudinary media fields
	MediaURL  string `json:"mediaUrl"`  // secure_url from Cloudinary
	MediaID   string `json:"mediaId"`   // public_id from Cloudinary
	MediaType string `json:"mediaType"` // "image" | "audio"

	// Only for audio (seconds)
	AudioDuration float64 `json:"audioDuration"`

	IsRead    bool      `gorm:"default:false" json:"isRead"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`

	TempID int64 `gorm:"-" json:"tempId"` // for optimistic UI
}

//	type ChatMessage struct {
//		ID         uint      `gorm:"primaryKey" json:"id"`
//		TicketID   uint      `gorm:"not null" json:"ticketId"`
//		SenderID   uint      `json:"senderId"`                   // admin or customer ID
//		SenderType string    `gorm:"not null" json:"senderType"` // "admin" | "customer"
//		Message    string    `gorm:"not null" json:"message"`
//		Image      string    `json:"image"`
//		IsRead     bool      `gorm:"default:false" json:"isRead"`
//		CreatedAt  time.Time `gorm:"autoCreateTime" json:"createdAt"`
//		UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
//		TempID     int64     `gorm:"-" json:"tempId"` // Transient field for optimistic UI matching
//	}
type MessageReadStatus struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	MessageID  uint      `gorm:"not null" json:"messageId"`
	AdminID    uint64    `json:"adminId"`
	CustomerID uint      `json:"customerId"`
	ReadAt     time.Time `gorm:"autoCreateTime" json:"readAt"`
}
