package repository

import (
	"customer-help-center-backend/internal/models"

	"gorm.io/gorm"
)

// --- Audit Log ---
type AuditRepository interface {
	Create(log *models.AuditLog) error
	FindByTicketID(ticketID uint) ([]models.AuditLog, error)
}
type auditRepository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) AuditRepository {
	return &auditRepository{db: db}
}
func (r *auditRepository) Create(log *models.AuditLog) error {
	return r.db.Create(log).Error
}
func (r *auditRepository) FindByTicketID(ticketID uint) ([]models.AuditLog, error) {
	var logs []models.AuditLog
	err := r.db.Where("ticket_id = ?", ticketID).Order("timestamp desc").Find(&logs).Error
	return logs, err
}

// --- Rating ---

// --- Work Session ---
