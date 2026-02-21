package service

import (
	"customer-help-center-backend/internal/models"
	"customer-help-center-backend/internal/repository"
	"errors"
	"time"
)

type OrderService interface {
	CreateOrder(order *models.Order) error
	GetOrder(id string) (*models.Order, error)
	GetOrdersByAdmin(adminID uint64) ([]models.Order, error)
	ListOrders(status, department string) ([]models.Order, error)
	UpdateOrder(id string, status, department string, branchName *string, estTime *time.Time) (*models.Order, error)
	DeleteOrder(id string) error
}

type orderService struct {
	repo repository.OrderRepository
}

func NewOrderService(repo repository.OrderRepository) OrderService {
	return &orderService{repo: repo}
}

func (s *orderService) CreateOrder(order *models.Order) error {

	if order.OrderID == "" {
		return errors.New("business order ID is required")
	}

	// Default status
	if order.Status == "" {
		order.Status = "pending"
	}

	return s.repo.Create(order)
}

func (s *orderService) GetOrder(id string) (*models.Order, error) {
	return s.repo.FindByID(id)
}

func (s *orderService) GetOrdersByAdmin(adminID uint64) ([]models.Order, error) {
	return s.repo.FindByAdminID(adminID)
}

func (s *orderService) ListOrders(status, department string) ([]models.Order, error) {
	filter := make(map[string]interface{})

	if status != "" {
		filter["status"] = status
	}

	if department != "" {
		filter["department"] = department
	}

	return s.repo.FindAll(filter)
}

func (s *orderService) UpdateOrder(
	id string,
	status string,
	department string,
	branchName *string,
	estTime *time.Time,
) (*models.Order, error) {

	order, err := s.repo.FindByID(id)
	if err != nil {
		return nil, errors.New("order not found")
	}

	// Update simple fields if provided
	if status != "" {
		order.Status = status
	}

	if department != "" {
		order.Department = department
	}

	// Nullable field handling
	if branchName != nil {
		order.BranchName = branchName
	}

	if estTime != nil {
		order.EstimatedTime = estTime
	}

	if err := s.repo.Update(order); err != nil {
		return nil, err
	}

	return order, nil
}

func (s *orderService) DeleteOrder(id string) error {
	_, err := s.repo.FindByID(id)
	if err != nil {
		return errors.New("order not found")
	}

	return s.repo.Delete(id)
}

// package service

// import (
// 	"customer-help-center-backend/internal/models"
// 	"customer-help-center-backend/internal/repository"
// 	"errors"
// 	"time"
// )

// type OrderService interface {
// 	CreateOrder(order *models.Order) error
// 	GetOrder(id string) (*models.Order, error)
// 	GetOrdersByUser(userID int) ([]models.Order, error)
// 	ListOrders(status string) ([]models.Order, error)
// 	UpdateOrder(id string, status, department, urgency, currentStage, branchName string, estFinish *time.Time) (*models.Order, error)
// 	DeleteOrder(id string) error
// }

// type orderService struct {
// 	repo repository.OrderRepository
// }

// func NewOrderService(repo repository.OrderRepository) OrderService {
// 	return &orderService{repo: repo}
// }

// func (s *orderService) CreateOrder(order *models.Order) error {
// 	if order.ID == "" {
// 		return errors.New("order ID is required")
// 	}
// 	// Check if already exists handled by DB unique constraint usually, or:
// 	existing, _ := s.repo.FindByID(order.ID)
// 	if existing != nil {
// 		return errors.New("Order ID already exists") // Mapped to P2002/409 logic
// 	}

// 	// Defaults handled in struct tags but good to ensure
// 	if order.Status == "" {
// 		order.Status = "pending"
// 	}
// 	if order.Urgency == "" {
// 		order.Urgency = "normal"
// 	}

// 	return s.repo.Create(order)
// }

// func (s *orderService) GetOrder(id string) (*models.Order, error) {
// 	return s.repo.FindByID(id)
// }

// func (s *orderService) GetOrdersByUser(userID int) ([]models.Order, error) {
// 	return s.repo.FindByUserID(userID)
// }

// func (s *orderService) ListOrders(status string) ([]models.Order, error) {
// 	// For general listing if needed
// 	filter := make(map[string]interface{})
// 	if status != "" {
// 		filter["status"] = status
// 	}
// 	return s.repo.FindAll(filter)
// }

// func (s *orderService) UpdateOrder(id string, status, department, urgency, currentStage, branchName string, estFinish *time.Time) (*models.Order, error) {
// 	order, err := s.repo.FindByID(id)
// 	if err != nil {
// 		return nil, errors.New("Order not found")
// 	}

// 	// Update basic fields if provided
// 	if department != "" {
// 		order.Department = department
// 	}
// 	if urgency != "" {
// 		order.Urgency = urgency
// 	}

// 	// State Machine & Workflow Logic
// 	// Initial default is Digital.
// 	if order.CurrentStage == "" {
// 		order.CurrentStage = "Digital"
// 	}

// 	// Allow status update (e.g. Cancelled) regardless of stage, or specific per stage.
// 	// Requirement: Each class can say Cancelled.
// 	if status == "Cancelled" {
// 		order.Status = "Cancelled"
// 	} else if status != "" {
// 		// Validating Status based on Stage
// 		switch order.CurrentStage {
// 		case "Digital":
// 			if status == "Processing" {
// 				order.Status = "Processing"
// 			} else if status == "Sent to Printing" {
// 				order.Status = "Sent to Printing" // Transition trigger?
// 				// Move to next stage
// 				order.CurrentStage = "Printing"
// 				// Reset status for new stage or keep "Sent to Printing"?
// 				// Usually "Sent to Printing" implies it IS in Printing queue.
// 				// Let's set status to "Pending Printing" or just keep the transition status until Printing class picks it up.
// 				// Prompt says: "next printing class can say , on going printing"
// 			}
// 		case "Printing":
// 			if status == "On Going Printing" {
// 				order.Status = "On Going Printing"
// 			} else if status == "Sent to Distribution" {
// 				order.Status = "Sent to Distribution"
// 				order.CurrentStage = "Distribution"
// 			}
// 		case "Distribution":
// 			if status == "On Distribution" {
// 				order.Status = "On Distribution"
// 			} else if status == "Distributed" {
// 				order.Status = "Distributed"
// 				order.CurrentStage = "Completed" // Or stay in Distribution? "finally disturbtion class can say on distrubution , and distrubuted"
// 				if branchName != "" {
// 					order.BranchName = branchName
// 				}
// 			}
// 		}
// 	}

// 	// Allow updating estimates for the current stage
// 	if estFinish != nil {
// 		switch order.CurrentStage {
// 		case "Digital":
// 			order.DigitalEstFinish = estFinish
// 		case "Printing":
// 			order.PrintingEstFinish = estFinish
// 		case "Distribution":
// 			order.DistributionEstFinish = estFinish
// 		}
// 	}

// 	// If manually correcting stage (admin override?), generally we stick to flow.

// 	if err := s.repo.Update(order); err != nil {
// 		return nil, err
// 	}
// 	return order, nil
// }

// func (s *orderService) DeleteOrder(id string) error {
// 	_, err := s.repo.FindByID(id)
// 	if err != nil {
// 		return errors.New("Order not found")
// 	}
// 	return s.repo.Delete(id)
// }
