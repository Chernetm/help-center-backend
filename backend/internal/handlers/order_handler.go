package handlers

import (
	"customer-help-center-backend/internal/models"
	"customer-help-center-backend/internal/service"
	"net/http"

	// "strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	service service.OrderService
}

func NewOrderHandler(s service.OrderService) *OrderHandler {
	return &OrderHandler{service: s}
}

// =====================
// CREATE ORDER
// =====================
func (h *OrderHandler) CreateOrder(c *gin.Context) {

	var req struct {
		OrderID       string  `json:"order_id"`
		Status        string  `json:"status"`
		Urgency       string  `json:"urgency"`
		Description   string  `json:"description"`
		EstimatedTime *string `json:"estimated_time"` // ISO string
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	if req.OrderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "order_id is required"})
		return
	}
	// Extract admin info from middleware
	adminIDVal, exists := c.Get("admin_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "admin not found in context"})
		return
	}

	// Assert the correct type from middleware
	adminID, ok := adminIDVal.(uint64) // keep as uint
	if !ok || adminID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid admin"})
		return
	}

	// Get department safely
	departmentVal, exists := c.Get("department")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "department not found in context"})
		return
	}

	department, ok := departmentVal.(string)

	var estTime *time.Time
	if req.EstimatedTime != nil {
		parsed, err := time.Parse(time.RFC3339, *req.EstimatedTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid estimated_time format"})
			return
		}
		estTime = &parsed
	}

	order := models.Order{
		OrderID:       req.OrderID,
		AdminID:       adminID,
		Department:    department,
		Status:        req.Status,
		Urgency:       req.Urgency,
		Description:   req.Description,
		EstimatedTime: estTime,
	}

	if err := h.service.CreateOrder(&order); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "order created successfully",
		"order":   order,
	})
}

// =====================
// GET ORDER BY ID
// =====================
func (h *OrderHandler) GetOrderById(c *gin.Context) {
	id := c.Param("id")

	order, err := h.service.GetOrder(id)
	if err != nil || order == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "order not found"})
		return
	}

	c.JSON(http.StatusOK, order)
}

// =====================
// TRACK ORDER (PUBLIC)
// =====================
func (h *OrderHandler) TrackOrder(c *gin.Context) {
	orderID := c.Param("id")

	order, err := h.service.GetOrderByBusinessID(orderID)
	if err != nil || order == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "order not found"})
		return
	}

	// Returning sanitized order (no internal IDs needed for tracking usually, but we keep structure)
	c.JSON(http.StatusOK, order)
}

// =====================
// GET ORDERS BY ADMIN
// =====================
func (h *OrderHandler) GetOrdersByAdmin(c *gin.Context) {
	// Extract admin info from middleware
	adminIDVal, exists := c.Get("admin_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "admin not found in context"})
		return
	}

	// Assert the correct type from middleware
	adminID, ok := adminIDVal.(uint64) // keep as uint
	if !ok || adminID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid admin"})
		return
	}

	orders, err := h.service.GetOrdersByAdmin(adminID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to fetch orders"})
		return
	}

	c.JSON(http.StatusOK, orders)
}

// =====================
// LIST ORDERS (FILTER)
// =====================
func (h *OrderHandler) ListOrders(c *gin.Context) {
	status := c.Query("status")
	department := c.Query("department")

	orders, err := h.service.ListOrders(status, department)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, orders)
}

// =====================
// UPDATE ORDER
// =====================
func (h *OrderHandler) UpdateOrder(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Status        string  `json:"status"`
		Department    string  `json:"department"`
		Urgency       string  `json:"urgency"`
		Description   string  `json:"description"`
		BranchName    *string `json:"branch_name"`
		EstimatedTime *string `json:"estimated_time"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}

	var estTime *time.Time
	if req.EstimatedTime != nil {
		parsed, err := time.Parse(time.RFC3339, *req.EstimatedTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid estimated_time format"})
			return
		}
		estTime = &parsed
	}

	// ⭐️ Get department from middleware if not provided or to enforce it
	deptVal, exists := c.Get("department")
	if exists {
		if dept, ok := deptVal.(string); ok && dept != "" {
			req.Department = dept // Use department from middleware
		}
	}

	updatedOrder, err := h.service.UpdateOrder(
		id,
		req.Status,
		req.Department,
		req.Urgency,
		req.Description,
		req.BranchName,
		estTime,
	)

	if err != nil {
		if err.Error() == "order not found" {
			c.JSON(http.StatusNotFound, gin.H{"message": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to update order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "order updated successfully",
		"order":   updatedOrder,
	})
}

// =====================
// DELETE ORDER
// =====================
func (h *OrderHandler) DeleteOrder(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.DeleteOrder(id); err != nil {
		if err.Error() == "order not found" {
			c.JSON(http.StatusNotFound, gin.H{"message": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to delete order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "order deleted successfully"})
}

// package handlers

// import (
// 	"customer-help-center-backend/internal/models"
// 	"customer-help-center-backend/internal/service"
// 	"net/http"
// 	"strconv"

// 	"github.com/gin-gonic/gin"
// )

// type OrderHandler struct {
// 	service service.OrderService
// }

// func NewOrderHandler(s service.OrderService) *OrderHandler {
// 	return &OrderHandler{service: s}
// }

// func (h *OrderHandler) CreateOrder(c *gin.Context) {
// 	var req struct {
// 		ID      string `json:"id"`
// 		Status  string `json:"status"`
// 		Urgency string `json:"urgency"`
// 	}
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
// 		return
// 	}

// 	if req.ID == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"message": "Order ID is required"})
// 		return
// 	}

// 	// Extract Admin Details from Context
// 	adminIDVal, _ := c.Get("admin_id") // set as uint64 in middleware
// 	departmentVal, _ := c.Get("department")

// 	adminID := int(adminIDVal.(uint64))
// 	department := departmentVal.(string)

// 	order := models.Order{
// 		ID:         req.ID,
// 		AdminID:    adminID,
// 		Status:     req.Status,
// 		Department: department,
// 		Urgency:    req.Urgency,
// 	}

// 	if err := h.service.CreateOrder(&order); err != nil {
// 		if err.Error() == "Order ID already exists" {
// 			c.JSON(http.StatusConflict, gin.H{"message": "Order ID already exists"})
// 			return
// 		}
// 		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to create order"})
// 		return
// 	}

// 	c.JSON(http.StatusCreated, gin.H{
// 		"message": "Order created successfully",
// 		"order":   order,
// 	})
// }
// func (h *OrderHandler) GetOrdersByUser(c *gin.Context) {
// 	userIdStr := c.Param("userId")
// 	userId, err := strconv.Atoi(userIdStr)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid User ID"})
// 		return
// 	}

// 	orders, err := h.service.GetOrdersByUser(userId)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to fetch orders"})
// 		return
// 	}
// 	c.JSON(http.StatusOK, orders)
// }

// func (h *OrderHandler) GetOrderById(c *gin.Context) {
// 	id := c.Param("id")
// 	order, err := h.service.GetOrder(id)
// 	if err != nil || order == nil {
// 		c.JSON(http.StatusNotFound, gin.H{"message": "Order not found"})
// 		return
// 	}
// 	c.JSON(http.StatusOK, order)
// }

// func (h *OrderHandler) UpdateOrder(c *gin.Context) {
// 	id := c.Param("id")
// 	var req struct {
// 		Status     string `json:"status"`
// 		Department string `json:"department"`
// 		Urgency    string `json:"urgency"`
// 	}
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
// 		return
// 	}

// 	updatedOrder, err := h.service.UpdateOrder(id, req.Status, req.Department, req.Urgency)
// 	if err != nil {
// 		if err.Error() == "Order not found" {
// 			c.JSON(http.StatusNotFound, gin.H{"message": "Order not found"})
// 			return
// 		}
// 		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update order"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"message": "Order updated successfully",
// 		"order":   updatedOrder,
// 	})
// }

// func (h *OrderHandler) DeleteOrder(c *gin.Context) {
// 	id := c.Param("id")
// 	if err := h.service.DeleteOrder(id); err != nil {
// 		if err.Error() == "Order not found" {
// 			c.JSON(http.StatusNotFound, gin.H{"message": "Order not found"})
// 			return
// 		}
// 		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete order"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{"message": "Order deleted successfully"})
// }

// func (h *OrderHandler) ListOrders(c *gin.Context) {
// 	// Query params?
// 	status := c.Query("status")
// 	orders, err := h.service.ListOrders(status)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}
// 	c.JSON(http.StatusOK, orders)
// }
