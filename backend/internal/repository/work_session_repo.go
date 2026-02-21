package repository

import (
	"customer-help-center-backend/internal/models"
	"time"

	"gorm.io/gorm"
)

type WorkSessionRepository interface {
	Create(session *models.WorkSession) error
	EndActiveSession(adminID uint64) error
	EndSessionByID(sessionID uint) error
	StartSession(session *models.WorkSession) error
	EndSession(sessionID uint, duration int) error
}

type workSessionRepository struct {
	db *gorm.DB
}

func NewWorkSessionRepository(db *gorm.DB) WorkSessionRepository {
	return &workSessionRepository{db: db}
}

func (r *workSessionRepository) Create(session *models.WorkSession) error {
	return r.db.Create(session).Error
}

func (r *workSessionRepository) EndActiveSession(adminID uint64) error {
	var session models.WorkSession
	// Find the most recent session for this admin that hasn't ended
	// We order by started_at desc to get the latest one just in case
	err := r.db.Where("admin_id = ? AND ended_at IS NULL", adminID).
		Order("started_at desc").
		First(&session).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil // No active session to end, which is fine
		}
		return err
	}

	now := time.Now()
	duration := int(now.Sub(session.StartedAt).Minutes())

	session.EndedAt = &now
	session.Duration = &duration

	return r.db.Save(&session).Error
}

func (r *workSessionRepository) StartSession(session *models.WorkSession) error {
	return r.db.Create(session).Error
}

func (r *workSessionRepository) EndSession(sessionID uint, duration int) error {
	return r.db.Model(&models.WorkSession{}).Where("id = ?", sessionID).
		Updates(map[string]interface{}{"ended_at": gorm.Expr("NOW()"), "duration": duration}).Error
}

func (r *workSessionRepository) EndSessionByID(sessionID uint) error {
	var session models.WorkSession
	if err := r.db.First(&session, sessionID).Error; err != nil {
		return err
	}

	now := time.Now()
	duration := int(now.Sub(session.StartedAt).Minutes())

	session.EndedAt = &now
	session.Duration = &duration

	return r.db.Save(&session).Error
}
