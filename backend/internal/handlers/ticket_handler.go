package handlers

import (
	"customer-help-center-backend/internal/models"
	"customer-help-center-backend/internal/service"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type TicketHandler struct {
	service       service.TicketService
	ratingService service.RatingService
}

func NewTicketHandler(service service.TicketService, ratingService service.RatingService) *TicketHandler {
	return &TicketHandler{
		service:       service,
		ratingService: ratingService,
	}
}

func (h *TicketHandler) CreateTicket(c *gin.Context) {
	var ticket models.Ticket
	if err := c.ShouldBindJSON(&ticket); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Get Customer ID from context (set by Middleware)
	customerID, exists := c.Get("customer_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: No customer ID found"})
		return
	}

	// Assign Customer ID to ticket
	idUint, ok := customerID.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Error: Invalid customer ID type"})
		return
	}
	ticket.CustomerID = idUint

	createdTicket, err := h.service.CreateTicket(&ticket)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Map to custom response format (same as GetAllTickets/GetTicket)
	ticketMap := gin.H{
		"id":         createdTicket.ID,
		"customerId": createdTicket.CustomerID,
		"adminId":    createdTicket.AdminID,
		"caseId":     createdTicket.CaseID,
		"status":     createdTicket.Status,
		"priority":   createdTicket.Priority,
		"createdAt":  createdTicket.CreatedAt,
		"updatedAt":  createdTicket.UpdatedAt,
		"customer":   createdTicket.Customer,
		"case":       createdTicket.Case,
		"chats":      createdTicket.Chats,
	}

	if createdTicket.Admin != nil {
		agentName := createdTicket.Admin.FirstName
		if len(createdTicket.Admin.LastName) > 0 {
			agentName = fmt.Sprintf("%s %s.", agentName, strings.ToUpper(string(createdTicket.Admin.LastName[0])))
		}

		ticketMap["agent"] = gin.H{
			"id":       createdTicket.Admin.ID,
			"name":     agentName,
			"isOnline": createdTicket.Admin.IsOnline,
			"image":    createdTicket.Admin.Image,
		}
	} else {
		ticketMap["agent"] = nil
	}

	c.JSON(http.StatusCreated, ticketMap)
}

func (h *TicketHandler) GetTicket(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	// Get Customer ID (optional, e.g. for admin it might be missing or we use admin flow)
	// But if middleware sets it, we use it for filtering.
	var customerID uint
	if val, exists := c.Get("customer_id"); exists {
		if idUint, ok := val.(uint); ok {
			customerID = idUint
		}
	}

	ticket, err := h.service.GetTicketByID(uint(id), customerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	c.JSON(http.StatusOK, ticket)
}

func (h *TicketHandler) GetAllTickets(c *gin.Context) {
	var tickets []models.Ticket
	var err error

	// Check for customer_id in context
	if customerID, exists := c.Get("customer_id"); exists {
		// If called by customer, return only their tickets
		idUint, ok := customerID.(uint)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Error: Invalid customer ID type"})
			return
		}
		tickets, err = h.service.GetCustomerTickets(idUint)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Map to custom response format
	var response []map[string]interface{}
	for _, t := range tickets {
		ticketMap := gin.H{
			"id":         t.ID,
			"customerId": t.CustomerID,
			"adminId":    t.AdminID,
			"caseId":     t.CaseID,
			"status":     t.Status,
			"priority":   t.Priority,
			"createdAt":  t.CreatedAt,
			"updatedAt":  t.UpdatedAt,
			"customer":   t.Customer,
			"case":       t.Case,
			"chats":      t.Chats,
		}

		if t.Admin != nil {
			agentName := t.Admin.FirstName
			if len(t.Admin.LastName) > 0 {
				agentName = fmt.Sprintf("%s %s.", agentName, strings.ToUpper(string(t.Admin.LastName[0])))
			}

			ticketMap["agent"] = gin.H{
				"id":       t.Admin.ID,
				"name":     agentName,
				"isOnline": t.Admin.IsOnline,
				"image":    t.Admin.Image,
			}
		} else {
			ticketMap["agent"] = nil
		}
		response = append(response, ticketMap)
	}

	c.JSON(http.StatusOK, response)
}

func (h *TicketHandler) GetAgentTickets(c *gin.Context) {
	var tickets []models.Ticket
	var err error

	// Check for admin_id in context (same style as customer)
	if adminID, exists := c.Get("admin_id"); exists {
		// If called by agent, return only their tickets
		idUint, ok := adminID.(uint64)
		if !ok {
			fmt.Println("GetAgentTickets Handler: Invalid admin ID type")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Internal Error: Invalid admin ID type",
			})
			return
		}
		fmt.Printf("GetAgentTickets Handler: Calling service with ID: %d\n", idUint)
		tickets, err = h.service.GetAgentTickets(idUint)
		fmt.Printf("GetAgentTickets Handler: Service returned %d tickets, err: %v\n", len(tickets), err)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Map response (Prisma-like)
	response := make([]gin.H, 0, len(tickets))

	for _, t := range tickets {
		ticketMap := gin.H{
			"id":        t.ID,
			"status":    t.Status,
			"priority":  t.Priority,
			"createdAt": t.CreatedAt,
			"updatedAt": t.UpdatedAt,
			"isClosed":  t.Status == "closed",

			"customer": gin.H{
				"id":          t.Customer.ID,
				"name":        t.Customer.Name,
				"phoneNumber": t.Customer.PhoneNumber,
				"email":       t.Customer.Email,
			},

			"chats": t.Chats,
		}

		if t.Admin != nil {
			agentName := t.Admin.FirstName
			if len(t.Admin.LastName) > 0 {
				agentName = fmt.Sprintf(
					"%s %s.",
					agentName,
					strings.ToUpper(string(t.Admin.LastName[0])),
				)
			}

			ticketMap["agent"] = gin.H{
				"id":       t.Admin.ID,
				"name":     agentName,
				"isOnline": t.Admin.IsOnline,
				"image":    t.Admin.Image,
			}
		} else {
			ticketMap["agent"] = nil
		}

		response = append(response, ticketMap)
	}

	c.JSON(http.StatusOK, response)
}

func (h *TicketHandler) CloseTicket(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	ticket, err := h.service.CloseTicket(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Ticket " + idStr + " closed successfully", "ticket": ticket})
}

func (h *TicketHandler) CreateRating(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Ticket ID"})
		return
	}

	var req struct {
		// CustomerID is taken from the ticket, not the body
		Score   int    `json:"score"`
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Get Customer ID from context (set by Middleware)
	customerIDCtx, exists := c.Get("customer_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: No customer ID found"})
		return
	}
	customerID, ok := customerIDCtx.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Error: Invalid customer ID type"})
		return
	}

	rating, err := h.ratingService.CreateRating(uint(id), customerID, req.Score, req.Comment)
	if err != nil {
		if err.Error() == "ticket not closed" || err.Error() == "rating already exists" {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		if err.Error() == "unauthorized: ticket does not belong to customer" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "ticket not found" {
			c.JSON(http.StatusNotFound, gin.H{"message": "Ticket not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Server error"})
		return
	}

	c.JSON(http.StatusCreated, rating)
}

func (h *TicketHandler) GetRating(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Ticket ID"})
		return
	}

	var customerID uint
	if val, exists := c.Get("customer_id"); exists {
		if idUint, ok := val.(uint); ok {
			customerID = idUint
		}
	}

	rating, err := h.ratingService.GetRating(uint(id), customerID)
	if err != nil {
		if err.Error() == "ticket not found" {
			c.JSON(http.StatusNotFound, gin.H{"message": "Ticket not found"})
			return
		}
		if err.Error() == "unauthorized: ticket does not belong to customer" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Server error"})
		return
	}
	if rating == nil {
		// If ticket exists but no rating, user code returns null
		c.JSON(http.StatusOK, nil)
		return
	}

	c.JSON(http.StatusOK, rating)
}
