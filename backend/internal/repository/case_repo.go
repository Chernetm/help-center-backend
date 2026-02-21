package repository

import (
	"customer-help-center-backend/internal/models"

	"gorm.io/gorm"
)

type CaseRepository interface {
	Create(c *models.Case) error
	FindAll() ([]models.Case, error)
	FindByID(id uint) (*models.Case, error)
	Update(c *models.Case) error
	Delete(id uint) error
}

type caseRepository struct {
	db *gorm.DB
}

func NewCaseRepository(db *gorm.DB) CaseRepository {
	return &caseRepository{db: db}
}

func (r *caseRepository) Create(c *models.Case) error {
	return r.db.Create(c).Error
}

func (r *caseRepository) FindAll() ([]models.Case, error) {
	var cases []models.Case
	err := r.db.Find(&cases).Error
	return cases, err
}

func (r *caseRepository) FindByID(id uint) (*models.Case, error) {
	var c models.Case
	err := r.db.First(&c, id).Error
	return &c, err
}

func (r *caseRepository) Update(c *models.Case) error {
	return r.db.Save(c).Error
}

func (r *caseRepository) Delete(id uint) error {
	return r.db.Delete(&models.Case{}, id).Error
}
