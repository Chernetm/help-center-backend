package service

import (
	"customer-help-center-backend/internal/models"
	"customer-help-center-backend/internal/repository"
	"errors"
)

type RatingService interface {
	CreateRating(ticketID uint, customerID uint, score int, comment string) (*models.Rating, error)
	GetRating(ticketID uint, customerID uint) (*models.Rating, error)
}

type ratingService struct {
	repo       repository.RatingRepository
	ticketRepo repository.TicketRepository
}

func NewRatingService(repo repository.RatingRepository, ticketRepo repository.TicketRepository) RatingService {
	return &ratingService{repo: repo, ticketRepo: ticketRepo}
}

func (s *ratingService) CreateRating(ticketID uint, customerID uint, score int, comment string) (*models.Rating, error) {
	ticket, err := s.ticketRepo.FindByID(ticketID, 0)
	if err != nil {
		return nil, errors.New("ticket not found") // Maps to 404 in handler ideally
	}

	// Validate ownership
	if ticket.CustomerID != customerID {
		return nil, errors.New("unauthorized: ticket does not belong to customer")
	}

	if ticket.Status != "closed" {
		return nil, errors.New("ticket not closed")
	}

	existing, _ := s.repo.FindByTicketID(ticketID)
	if existing != nil {
		return nil, errors.New("rating already exists")
	}

	rating := &models.Rating{
		TicketID:   ticketID,
		CustomerID: ticket.CustomerID, // Use ticket's customer ID
		Score:      score,
		Comment:    comment,
	}

	if err := s.repo.Create(rating); err != nil {
		return nil, err
	}
	return rating, nil
}

func (s *ratingService) GetRating(ticketID uint, customerID uint) (*models.Rating, error) {
	// Check ticket exists first
	ticket, err := s.ticketRepo.FindByID(ticketID, 0)
	if err != nil {
		return nil, errors.New("ticket not found")
	}

	if customerID > 0 && ticket.CustomerID != customerID {
		return nil, errors.New("unauthorized: ticket does not belong to customer")
	}

	return s.repo.FindByTicketID(ticketID)
}
