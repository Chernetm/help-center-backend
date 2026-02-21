package repository

import (
	"customer-help-center-backend/internal/models"

	"gorm.io/gorm"
)

type RatingRepository interface {
	Create(rating *models.Rating) error
	FindByTicketID(ticketID uint) (*models.Rating, error)
	GetAverageScore(adminID uint) (float64, error)
}

type ratingRepository struct {
	db *gorm.DB
}

func NewRatingRepository(db *gorm.DB) RatingRepository {
	return &ratingRepository{db: db}
}

func (r *ratingRepository) Create(rating *models.Rating) error {
	return r.db.Create(rating).Error
}

func (r *ratingRepository) FindByTicketID(ticketID uint) (*models.Rating, error) {
	var rating models.Rating
	if err := r.db.Where("ticket_id = ?", ticketID).First(&rating).Error; err != nil {
		return nil, err
	}
	return &rating, nil
}

func (r *ratingRepository) GetAverageScore(adminID uint) (float64, error) {
	// Complex query involving joins through Ticket -> Admin
	var avgScore float64
	err := r.db.Model(&models.Rating{}).
		Joins("JOIN tickets ON tickets.id = ratings.ticket_id").
		Where("tickets.admin_id = ?", adminID).
		Select("AVG(score)").
		Scan(&avgScore).Error
	return avgScore, err
}
