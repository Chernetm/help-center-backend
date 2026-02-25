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
	Delete(id uint) error
	DeleteUnassignedTickets() error
	DeleteAllTickets() error
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

	// Default ordering - Most recent update first (Telegram style)
	query = query.Order("updated_at desc")

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
func (r *ticketRepository) Delete(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Delete associations first to avoid foreign key errors
		if err := tx.Where("ticket_id = ?", id).Delete(&models.ChatMessage{}).Error; err != nil {
			return err
		}
		if err := tx.Where("ticket_id = ?", id).Delete(&models.Rating{}).Error; err != nil {
			return err
		}
		if err := tx.Where("ticket_id = ?", id).Delete(&models.TicketMetrics{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ?", id).Delete(&models.Ticket{}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *ticketRepository) DeleteUnassignedTickets() error {
	var tickets []models.Ticket
	// A ticket is unassigned if AdminID is NULL
	if err := r.db.Where("admin_id IS NULL").Find(&tickets).Error; err != nil {
		return err
	}
	// if err := r.db.Unscoped().Where("1 = 1").Delete(&models.Ticket{}).Error; err != nil {
	// 	return err
	// }
	for _, t := range tickets {
		fmt.Printf("AutoCleanup: Deleting unassigned ticket ID=%d\n", t.ID)
		if err := r.Delete(t.ID); err != nil {
			fmt.Printf("AutoCleanup: Error deleting ticket %d: %v\n", t.ID, err)
		}
	}
	return nil
}
func (r *ticketRepository) DeleteAllTickets() error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Delete associations first
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.MessageReadStatus{}).Error; err != nil {
			return err
		}
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.ChatMessage{}).Error; err != nil {
			return err
		}
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Rating{}).Error; err != nil {
			return err
		}
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.TicketMetrics{}).Error; err != nil {
			return err
		}
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.Ticket{}).Error; err != nil {
			return err
		}

		// Also reset agent ticket counts? User didn't ask but it's good practice.
		// However, it's safer to just do what's asked.
		return nil
	})
}
