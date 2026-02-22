// package repository

package repository

import (
	"customer-help-center-backend/internal/models"

	"gorm.io/gorm"
)

type OrderRepository interface {
	Create(order *models.Order) error
	FindByID(id string) (*models.Order, error)
	FindByOrderID(orderID string) (*models.Order, error)
	FindByAdminID(adminID uint64) ([]models.Order, error)
	FindAll(filter map[string]interface{}) ([]models.Order, error)
	Update(order *models.Order) error
	Delete(id string) error
}

type orderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) Create(order *models.Order) error {
	return r.db.Create(order).Error
}

func (r *orderRepository) FindByID(id string) (*models.Order, error) {
	var order models.Order

	if err := r.db.First(&order, "id = ?", id).Error; err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *orderRepository) FindByOrderID(orderID string) (*models.Order, error) {
	var order models.Order

	if err := r.db.First(&order, "order_id = ?", orderID).Error; err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *orderRepository) FindByAdminID(adminID uint64) ([]models.Order, error) {
	var orders []models.Order

	if err := r.db.
		Where("admin_id = ?", adminID).
		Order("created_at desc").
		Find(&orders).Error; err != nil {
		return nil, err
	}

	return orders, nil
}

func (r *orderRepository) FindAll(filter map[string]interface{}) ([]models.Order, error) {
	var orders []models.Order
	query := r.db.Model(&models.Order{})

	if val, ok := filter["admin_id"]; ok {
		query = query.Where("admin_id = ?", val)
	}
	if val, ok := filter["status"]; ok {
		query = query.Where("status = ?", val)
	}
	if val, ok := filter["department"]; ok {
		query = query.Where("department = ?", val)
	}
	if val, ok := filter["branch_name"]; ok {
		if val == nil {
			query = query.Where("branch_name IS NULL")
		} else {
			query = query.Where("branch_name = ?", val)
		}
	}

	if err := query.
		Order("created_at desc").
		Find(&orders).Error; err != nil {
		return nil, err
	}

	return orders, nil
}

func (r *orderRepository) Update(order *models.Order) error {
	return r.db.Save(order).Error
}

func (r *orderRepository) Delete(id string) error {
	return r.db.Delete(&models.Order{}, "id = ?", id).Error
}

// import (
// 	"customer-help-center-backend/internal/models"

// 	"gorm.io/gorm"
// )

// type OrderRepository interface {
// 	Create(order *models.Order) error
// 	FindByID(id string) (*models.Order, error)
// 	FindByUserID(userID int) ([]models.Order, error)
// 	FindAll(filter map[string]interface{}) ([]models.Order, error)
// 	Update(order *models.Order) error
// 	Delete(id string) error
// }

// type orderRepository struct {
// 	db *gorm.DB
// }

// func NewOrderRepository(db *gorm.DB) OrderRepository {
// 	return &orderRepository{db: db}
// }

// func (r *orderRepository) Create(order *models.Order) error {
// 	return r.db.Create(order).Error
// }

// func (r *orderRepository) FindByID(id string) (*models.Order, error) {
// 	var order models.Order
// 	// Node: include: { user: true, audits: true }
// 	// We don't have audits model visible yet, but let's preload User (Admin).
// 	if err := r.db.Preload("User").First(&order, "id = ?", id).Error; err != nil {
// 		return nil, err
// 	}
// 	return &order, nil
// }

// func (r *orderRepository) FindByUserID(userID int) ([]models.Order, error) {
// 	var orders []models.Order
// 	// Node: where: { userId }, orderBy: { createdAt: 'desc' }, include: { user: ... }
// 	if err := r.db.Preload("User").
// 		Where("user_id = ?", userID).
// 		Order("created_at desc").
// 		Find(&orders).Error; err != nil {
// 		return nil, err
// 	}
// 	return orders, nil
// }

// func (r *orderRepository) FindAll(filter map[string]interface{}) ([]models.Order, error) {
// 	var orders []models.Order
// 	query := r.db.Model(&models.Order{}).Preload("User")

// 	if val, ok := filter["user_id"]; ok {
// 		query = query.Where("user_id = ?", val)
// 	}
// 	if val, ok := filter["status"]; ok {
// 		query = query.Where("status = ?", val)
// 	}

// 	if err := query.Find(&orders).Error; err != nil {
// 		return nil, err
// 	}
// 	return orders, nil
// }

// func (r *orderRepository) Update(order *models.Order) error {
// 	return r.db.Save(order).Error
// }

// func (r *orderRepository) Delete(id string) error {
// 	return r.db.Delete(&models.Order{}, "id = ?", id).Error
// }
