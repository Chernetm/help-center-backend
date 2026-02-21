package handlers

import (
	"customer-help-center-backend/internal/models"
	"customer-help-center-backend/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	service service.ChatService
}

func NewChatHandler(s service.ChatService) *ChatHandler {
	return &ChatHandler{service: s}
}

func (h *ChatHandler) SendMessage(c *gin.Context) {
	// Generic fallback
	var msg models.ChatMessage
	if err := c.ShouldBindJSON(&msg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if err := h.service.SendMessage(&msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, msg)
}

func (h *ChatHandler) SendCustomerMessage(c *gin.Context) {
	var req struct {
		Message       string  `json:"message"`
		MediaURL      string  `json:"mediaUrl"`
		MediaID       string  `json:"mediaId"`
		MediaType     string  `json:"mediaType"`     // "image" | "audio"
		AudioDuration float64 `json:"audioDuration"` // only for audio
		TicketID      uint    `json:"ticketId"`
		TempID        int64   `json:"tempId"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	customerIDCtx, exists := c.Get("customer_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	customerID, ok := customerIDCtx.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid customer ID type"})
		return
	}

	chat, err := h.service.SendCustomerMessage(
		req.TicketID,
		customerID,
		req.Message,
		req.MediaURL,
		req.MediaID,
		req.MediaType,
		req.AudioDuration,
		req.TempID,
	)
	if err != nil {
		switch err.Error() {
		case "ticket not found":
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case "unauthorized: ticket does not belong to customer":
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		case "chat not allowed: ticket is closed":
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot send message to a closed ticket"})
		case "message or media is required":
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, chat)
}

func (h *ChatHandler) SendAgentMessage(c *gin.Context) {
	var req struct {
		Message       string  `json:"message"`
		MediaURL      string  `json:"mediaUrl"`
		MediaID       string  `json:"mediaId"`
		MediaType     string  `json:"mediaType"`
		AudioDuration float64 `json:"audioDuration"`
		TicketID      uint    `json:"ticketId"`
		TempID        int64   `json:"tempId"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	adminIDCtx, exists := c.Get("admin_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	adminID, ok := adminIDCtx.(uint64)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid admin ID type"})
		return
	}

	chat, err := h.service.SendAgentMessage(
		req.TicketID,
		adminID,
		req.Message,
		req.MediaURL,
		req.MediaID,
		req.MediaType,
		req.AudioDuration,
		req.TempID,
	)
	if err != nil {
		switch err.Error() {
		case "ticket not found":
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case "chat not allowed: ticket not assigned":
			c.JSON(http.StatusForbidden, gin.H{"message": "Chat not allowed"})
		case "chat not allowed: ticket is closed":
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot send message to a closed ticket"})
		case "message or media is required":
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, chat)
}

func (h *ChatHandler) GetHistory(c *gin.Context) {
	ticketIDStr := c.Param("ticketId")
	ticketID, err := strconv.Atoi(ticketIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Ticket ID"})
		return
	}

	msgs, err := h.service.GetTicketHistory(uint(ticketID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, msgs)
}

func (h *ChatHandler) MarkMessagesAsRead(c *gin.Context) {
	ticketIDStr := c.Param("id")
	ticketID, err := strconv.Atoi(ticketIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Ticket ID"})
		return
	}

	var role string
	if _, exists := c.Get("admin_id"); exists {
		role = "admin"
	} else if _, exists := c.Get("customer_id"); exists {
		role = "customer"
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.service.MarkTicketAsRead(uint(ticketID), role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Messages marked as read"})
}

// package handlers

// import (
// 	"customer-help-center-backend/internal/models"
// 	"customer-help-center-backend/internal/service"
// 	"net/http"
// 	"strconv"

// 	"github.com/gin-gonic/gin"
// )

// type ChatHandler struct {
// 	service service.ChatService
// }

// func NewChatHandler(s service.ChatService) *ChatHandler {
// 	return &ChatHandler{service: s}
// }

// func (h *ChatHandler) SendMessage(c *gin.Context) {
// 	// Deprecated or keep generic?
// 	// Let's keep it if old frontend uses it, but the request focuses on specific flows.
// 	var msg models.ChatMessage
// 	if err := c.ShouldBindJSON(&msg); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
// 		return
// 	}
// 	if err := h.service.SendMessage(&msg); err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}
// 	c.JSON(http.StatusCreated, msg)
// }

// func (h *ChatHandler) SendCustomerMessage(c *gin.Context) {
// 	var req struct {
// 		Message  string `json:"message"`
// 		TicketID uint   `json:"ticketId"`
// 		TempID   int64  `json:"tempId"`
// 	}
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
// 		return
// 	}

// 	// Get Customer ID from context (secure)
// 	customerIDCtx, exists := c.Get("customer_id")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
// 		return
// 	}
// 	customerID, ok := customerIDCtx.(uint)
// 	if !ok {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid customer ID type"})
// 		return
// 	}

// 	// Call service to send message
// 	chat, err := h.service.SendCustomerMessage(req.TicketID, customerID, req.Message, req.TempID)
// 	if err != nil {
// 		if err.Error() == "ticket not found" {
// 			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
// 			return
// 		}
// 		if err.Error() == "unauthorized: ticket does not belong to customer" {
// 			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
// 			return
// 		}
// 		if err.Error() == "chat not allowed: ticket is closed" {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot send message to a closed ticket"})
// 			return
// 		}
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	c.JSON(http.StatusOK, chat) // User code expects just the chat object
// }

// func (h *ChatHandler) SendAgentMessage(c *gin.Context) {
// 	var req struct {
// 		Message  string `json:"message"`
// 		TicketID uint   `json:"ticketId"`
// 		TempID   int64  `json:"tempId"`
// 	}

// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
// 		return
// 	}

// 	// Get Agent ID from context (secure)
// 	adminIDCtx, exists := c.Get("admin_id")
// 	if !exists {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
// 		return
// 	}
// 	adminID, ok := adminIDCtx.(uint)
// 	if !ok {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid admin ID type"})
// 		return
// 	}

// 	chat, err := h.service.SendAgentMessage(req.TicketID, adminID, req.Message, req.TempID)
// 	if err != nil {
// 		if err.Error() == "ticket not found" {
// 			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
// 			return
// 		}
// 		if err.Error() == "chat not allowed: ticket not assigned" {
// 			c.JSON(http.StatusForbidden, gin.H{"message": "Chat not allowed"})
// 			return
// 		}
// 		if err.Error() == "chat not allowed: ticket is closed" {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot send message to a closed ticket"})
// 			return
// 		}
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}
// 	c.JSON(http.StatusOK, chat)
// }

// func (h *ChatHandler) GetHistory(c *gin.Context) {
// 	ticketIDStr := c.Param("ticketId")
// 	ticketID, err := strconv.Atoi(ticketIDStr)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Ticket ID"})
// 		return
// 	}
// 	msgs, err := h.service.GetTicketHistory(uint(ticketID))
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}
// 	c.JSON(http.StatusOK, msgs)
// }

// func (h *ChatHandler) MarkMessagesAsRead(c *gin.Context) {
// 	ticketIDStr := c.Param("id")
// 	ticketID, err := strconv.Atoi(ticketIDStr)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Ticket ID"})
// 		return
// 	}

// 	var role string
// 	if _, exists := c.Get("admin_id"); exists {
// 		role = "admin"
// 	} else if _, exists := c.Get("customer_id"); exists {
// 		role = "customer"
// 	} else {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
// 		return
// 	}

// 	if err := h.service.MarkTicketAsRead(uint(ticketID), role); err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{"message": "Messages marked as read"})
// }
