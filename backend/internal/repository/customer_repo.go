package repository

import (
	"customer-help-center-backend/internal/models"

	"gorm.io/gorm"
)

type CustomerRepository interface {
	Create(customer *models.Customer) error
	FindByEmail(email string) (*models.Customer, error)
	FindByUID(uid string) (*models.Customer, error)
	FindByID(id uint) (*models.Customer, error)
	FindAll() ([]models.Customer, error)
	Update(customer *models.Customer) error
	UpdateStatus(id uint, status string) error
	UpdateRole(id uint, role string) error
	Delete(id uint) error
}

type customerRepository struct {
	db *gorm.DB
}

func NewCustomerRepository(db *gorm.DB) CustomerRepository {
	return &customerRepository{db: db}
}

func (r *customerRepository) Create(customer *models.Customer) error {
	return r.db.Create(customer).Error
}

func (r *customerRepository) FindByEmail(email string) (*models.Customer, error) {
	var customer models.Customer
	if err := r.db.Where("email = ?", email).First(&customer).Error; err != nil {
		return nil, err
	}
	return &customer, nil
}

func (r *customerRepository) FindByUID(uid string) (*models.Customer, error) {
	var customer models.Customer
	if err := r.db.Where("uid = ?", uid).First(&customer).Error; err != nil {
		return nil, err
	}
	return &customer, nil
}

func (r *customerRepository) FindByID(id uint) (*models.Customer, error) {
	var customer models.Customer
	if err := r.db.First(&customer, id).Error; err != nil {
		return nil, err
	}
	return &customer, nil
}

func (r *customerRepository) FindAll() ([]models.Customer, error) {
	var customers []models.Customer
	// Use a subquery to calculate ticket count for each customer
	if err := r.db.Select("customers.*, (SELECT count(*) FROM tickets WHERE tickets.customer_id = customers.id) as ticket_count").Find(&customers).Error; err != nil {
		return nil, err
	}
	return customers, nil
}

func (r *customerRepository) Update(customer *models.Customer) error {
	return r.db.Save(customer).Error
}

func (r *customerRepository) UpdateStatus(id uint, status string) error {
	return r.db.Model(&models.Customer{}).Where("id = ?", id).Update("status", status).Error
}

func (r *customerRepository) UpdateRole(id uint, role string) error {
	return r.db.Model(&models.Customer{}).Where("id = ?", id).Update("role", role).Error
}

func (r *customerRepository) Delete(id uint) error {
	return r.db.Delete(&models.Customer{}, id).Error
}
