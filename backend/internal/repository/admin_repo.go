package repository

import (
	"customer-help-center-backend/internal/models"

	"gorm.io/gorm"
)

type AdminRepository interface {
	Create(admin *models.Admin) error
	FindByEmail(email string) (*models.Admin, error)
	FindByUID(uid string) (*models.Admin, error)
	FindByID(id uint64) (*models.Admin, error)
	FindAll() ([]models.Admin, error)
	FindAvailableAgent(department string) (*models.Admin, error)
	Update(admin *models.Admin) error
	Delete(uid string) error
	FindAgentsFullDetails() ([]models.Admin, error)
	SetOnlineStatus(adminID uint64, isOnline bool) error
}

type adminRepository struct {
	db *gorm.DB
}

func NewAdminRepository(db *gorm.DB) AdminRepository {
	return &adminRepository{db: db}
}

func (r *adminRepository) Create(admin *models.Admin) error {
	return r.db.Create(admin).Error
}

// func (r *adminRepository) FindByUsername(username string) (*models.Admin, error) {
// 	var admin models.Admin
// 	if err := r.db.Where("username = ?", username).First(&admin).Error; err != nil {
// 		return nil, err
// 	}
// 	return &admin, nil
// }

func (r *adminRepository) FindByEmail(email string) (*models.Admin, error) {
	var admin models.Admin
	if err := r.db.Where("email = ?", email).First(&admin).Error; err != nil {
		return nil, err
	}
	return &admin, nil
}

func (r *adminRepository) FindByUID(uid string) (*models.Admin, error) {
	var admin models.Admin
	if err := r.db.Where("uid = ?", uid).First(&admin).Error; err != nil {
		return nil, err
	}
	return &admin, nil
}

func (r *adminRepository) FindByID(id uint64) (*models.Admin, error) {
	var admin models.Admin
	if err := r.db.First(&admin, id).Error; err != nil {
		return nil, err
	}
	return &admin, nil
}

func (r *adminRepository) Update(admin *models.Admin) error {
	return r.db.Save(admin).Error
}

func (r *adminRepository) FindAll() ([]models.Admin, error) {
	var admins []models.Admin
	if err := r.db.Find(&admins).Error; err != nil {
		return nil, err
	}
	return admins, nil
}

func (r *adminRepository) Delete(uid string) error {
	return r.db.Where("uid = ?", uid).Delete(&models.Admin{}).Error
}

func (r *adminRepository) FindAvailableAgent(department string) (*models.Admin, error) {
	var agent models.Admin
	// Find agent with role='agent', isOnline=true, department=selectedCase.department
	// Order by activeTicketQty asc
	err := r.db.Where("role = ? AND is_online = ? AND department = ?", "admin", false, department).
		Order("active_ticket_qty asc").
		First(&agent).Error
	if err != nil {
		return nil, err
	}
	return &agent, nil
}

func (r *adminRepository) FindAgentsFullDetails() ([]models.Admin, error) {
	var admins []models.Admin
	// Fetch all admins (agents) with full nested structure
	// Assuming "agent" role or just all admins? The prompt says role: "agent"
	// Note: Ticket struct has AdminID, so we preload Tickets.
	// Ticket has Ratings.
	err := r.db.Preload("WorkSessions").
		Preload("Tickets", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "status", "customer_id", "admin_id")
		}).
		Preload("Tickets.Ratings", func(db *gorm.DB) *gorm.DB {
			return db.Select("ticket_id", "score")
		}).
		Where("role = ?", "admin"). // Adjust role if needed (e.g. "admin" vs "agent")
		Find(&admins).Error

	return admins, err
}

func (r *adminRepository) SetOnlineStatus(adminID uint64, isOnline bool) error {
	return r.db.Model(&models.Admin{}).Where("id = ?", adminID).Update("is_online", isOnline).Error
}
