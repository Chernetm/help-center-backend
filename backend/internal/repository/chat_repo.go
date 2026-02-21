package repository

import (
	"customer-help-center-backend/internal/models"

	"gorm.io/gorm"
)

type ChatRepository interface {
	CreateMessage(message *models.ChatMessage) error
	GetMessagesByTicketID(ticketID uint) ([]models.ChatMessage, error)
	MarkMessagesAsRead(messageIDs []uint, adminID uint64) error
	MarkTicketMessagesAsRead(ticketID uint, readerRole string) error
}

type chatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) ChatRepository {
	return &chatRepository{db: db}
}

func (r *chatRepository) CreateMessage(message *models.ChatMessage) error {
	return r.db.Create(message).Error
}

func (r *chatRepository) GetMessagesByTicketID(ticketID uint) ([]models.ChatMessage, error) {
	var messages []models.ChatMessage
	// Preload sender info if needed, or read receipts
	if err := r.db.Where("ticket_id = ?", ticketID).Order("created_at asc").Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *chatRepository) MarkMessagesAsRead(messageIDs []uint, adminID uint64) error {
	// Logic to batch insert into MessageReadStatus or update IsRead flag
	// For simplicity, we just update IsRead on the message itself for now,
	// but the schema suggests a MessageReadStatus table for individual admin tracking.
	// Let's implement the IsRead flag update for the message as a simplified approach first
	// or properly insert ReadStatus.

	// Transaction
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.ChatMessage{}).Where("id IN ?", messageIDs).Update("is_read", true).Error; err != nil {
			return err
		}
		// Also create read status entries
		for _, mid := range messageIDs {
			readStatus := models.MessageReadStatus{
				MessageID: mid,
				AdminID:   adminID,
			}
			if err := tx.Create(&readStatus).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *chatRepository) MarkTicketMessagesAsRead(ticketID uint, readerRole string) error {
	var targetSenderType string
	if readerRole == "admin" {
		targetSenderType = "customer"
	} else if readerRole == "customer" {
		targetSenderType = "agent"
	} else {
		return nil // Unknown role, do nothing or error
	}

	return r.db.Model(&models.ChatMessage{}).
		Where("ticket_id = ? AND sender_type = ? AND is_read = ?", ticketID, targetSenderType, false).
		Update("is_read", true).Error
}
