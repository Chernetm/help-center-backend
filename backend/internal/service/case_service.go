package service

import (
	"customer-help-center-backend/internal/models"
	"customer-help-center-backend/internal/repository"
)

type CaseService interface {
	CreateCase(req *models.Case) (*models.Case, error)
	GetAllCases() ([]models.Case, error)
	GetCase(id uint) (*models.Case, error)
	UpdateCase(id uint, req *models.Case) (*models.Case, error)
	DeleteCase(id uint) error
}

type caseService struct {
	repo repository.CaseRepository
}

func NewCaseService(repo repository.CaseRepository) CaseService {
	return &caseService{repo: repo}
}

func (s *caseService) CreateCase(req *models.Case) (*models.Case, error) {
	// Basic validation if needed
	if err := s.repo.Create(req); err != nil {
		return nil, err
	}
	return req, nil
}

func (s *caseService) GetAllCases() ([]models.Case, error) {
	return s.repo.FindAll()
}

func (s *caseService) GetCase(id uint) (*models.Case, error) {
	return s.repo.FindByID(id)
}

func (s *caseService) UpdateCase(id uint, req *models.Case) (*models.Case, error) {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	existing.Name = req.Name
	existing.Description = req.Description
	existing.Department = req.Department

	if err := s.repo.Update(existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *caseService) DeleteCase(id uint) error {
	return s.repo.Delete(id)
}
