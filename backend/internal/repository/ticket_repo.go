package repository

import (
	"customer-help-center-backend/internal/models"
	"fmt"

	"gorm.io/gorm"
)

type TicketRepository interface {
	Create(ticket *models.Ticket) error
	FindByID(id uint, customerID uint) (*models.Ticket, error)
	FindAll(filter map[string]interface{}) ([]models.Ticket, error)
	Update(ticket *models.Ticket) error
	FindActiveTicket(customerID uint, caseID uint) (*models.Ticket, error)
}

type ticketRepository struct {
	db *gorm.DB
}

func NewTicketRepository(db *gorm.DB) TicketRepository {
	return &ticketRepository{db: db}
}

func (r *ticketRepository) Create(ticket *models.Ticket) error {
	return r.db.Create(ticket).Error
}

func (r *ticketRepository) FindByID(id uint, customerID uint) (*models.Ticket, error) {
	var ticket models.Ticket
	query := r.db.Preload("Customer").Preload("Case").Preload("Admin").Preload("Chats")

	if customerID > 0 {
		query = query.Where("customer_id = ?", customerID)
	}

	if err := query.First(&ticket, id).Error; err != nil {
		return nil, err
	}
	return &ticket, nil
}
func (r *ticketRepository) FindAll(filter map[string]interface{}) ([]models.Ticket, error) {
	fmt.Printf("TicketRepo: FindAll called with filter: %+v\n", filter)
	var tickets []models.Ticket

	query := r.db.Model(&models.Ticket{}).
		Preload("Customer").
		Preload("Case").
		Preload("Chats", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at asc")
		}).
		Preload("Admin")

	if val, ok := filter["customer_id"]; ok {
		query = query.Where("customer_id = ?", val)
	}

	if val, ok := filter["admin_id"]; ok {
		query = query.Where("admin_id = ?", val)
	}

	if val, ok := filter["status"]; ok {
		query = query.Where("status = ?", val)
	}

	// Default ordering
	query = query.Order("created_at asc")

	if err := query.Find(&tickets).Error; err != nil {
		return nil, err
	}

	return tickets, nil
}

func (r *ticketRepository) Update(ticket *models.Ticket) error {
	return r.db.Save(ticket).Error
}

func (r *ticketRepository) FindActiveTicket(customerID uint, caseID uint) (*models.Ticket, error) {
	var ticket models.Ticket
	// Find ticket for this customer and case that is NOT closed
	err := r.db.Where("customer_id = ? AND case_id = ? AND status != ?", customerID, caseID, "closed").
		Preload("Customer").
		Preload("Case").
		Preload("Admin").
		Preload("Chats").
		First(&ticket).Error

	if err != nil {
		fmt.Printf("FindActiveTicket: No active ticket found for C:%d Case:%d. Error: %v\n", customerID, caseID, err)
		return nil, err
	}
	fmt.Printf("FindActiveTicket: Found existing ticket ID: %d for C:%d Case:%d\n", ticket.ID, customerID, caseID)
	return &ticket, nil
}
