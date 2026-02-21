package service

import (
	"customer-help-center-backend/internal/models"
	"customer-help-center-backend/internal/repository"
)

type CommonService interface {
	// Audit
	LogAction(log *models.AuditLog) error
	GetTicketLogs(ticketID uint) ([]models.AuditLog, error)
	// Rating
	SubmitRating(rating *models.Rating) error
	// Session
	StartSession(session *models.WorkSession) error
	EndSession(sessionID uint, duration int) error
}

type commonService struct {
	auditRepo   repository.AuditRepository
	ratingRepo  repository.RatingRepository
	sessionRepo repository.WorkSessionRepository
}

func NewCommonService(
	aRepo repository.AuditRepository,
	rRepo repository.RatingRepository,
	sRepo repository.WorkSessionRepository,
) CommonService {
	return &commonService{
		auditRepo:   aRepo,
		ratingRepo:  rRepo,
		sessionRepo: sRepo,
	}
}

func (s *commonService) LogAction(log *models.AuditLog) error {
	return s.auditRepo.Create(log)
}

func (s *commonService) GetTicketLogs(ticketID uint) ([]models.AuditLog, error) {
	return s.auditRepo.FindByTicketID(ticketID)
}

func (s *commonService) SubmitRating(rating *models.Rating) error {
	// Could validate that ticket is resolved first
	return s.ratingRepo.Create(rating)
}

func (s *commonService) StartSession(session *models.WorkSession) error {
	return s.sessionRepo.StartSession(session)
}

func (s *commonService) EndSession(sessionID uint, duration int) error {
	return s.sessionRepo.EndSession(sessionID, duration)
}
