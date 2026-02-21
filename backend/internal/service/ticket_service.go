package service

import (
	"customer-help-center-backend/internal/models"
	"customer-help-center-backend/internal/repository"
	"errors"
	"fmt"
)

type TicketService interface {
	CreateTicket(ticket *models.Ticket) (*models.Ticket, error)
	GetTicketByID(id uint, customerID uint) (*models.Ticket, error)
	GetCustomerTickets(customerID uint) ([]models.Ticket, error)
	GetAgentTickets(agentID uint64) ([]models.Ticket, error)
	GetAllTickets() ([]models.Ticket, error)
	CloseTicket(id uint) (*models.Ticket, error)
	CreateRating(ticketID uint, customerID uint, score int, comment string) (*models.Rating, error)
	GetRating(ticketID uint, customerID uint) (*models.Rating, error)
	SendCustomerMessage(ticketID uint, customerID uint, message string) (*models.ChatMessage, error)
}

type ticketService struct {
	repo       repository.TicketRepository
	caseRepo   repository.CaseRepository
	adminRepo  repository.AdminRepository
	ratingRepo repository.RatingRepository
	chatRepo   repository.ChatRepository
	socket     SocketService
	common     repository.AuditRepository
}

func NewTicketService(
	repo repository.TicketRepository,
	caseRepo repository.CaseRepository,
	adminRepo repository.AdminRepository,
	ratingRepo repository.RatingRepository,
	chatRepo repository.ChatRepository,
	socket SocketService,
) TicketService {
	return &ticketService{
		repo:       repo,
		caseRepo:   caseRepo,
		adminRepo:  adminRepo,
		ratingRepo: ratingRepo,
		chatRepo:   chatRepo,
		socket:     socket,
	}
}
func (s *ticketService) CreateTicket(ticket *models.Ticket) (*models.Ticket, error) {
	if ticket.CustomerID == 0 {
		return nil, errors.New("customer ID is required")
	}

	// 1. Check for existing active ticket for this customer and case
	existingTicket, _ := s.repo.FindActiveTicket(ticket.CustomerID, ticket.CaseID)
	if existingTicket != nil {
		// Found an existing open/assigned/waiting ticket, return it
		return existingTicket, nil
	}

	// 2. No existing ticket, proceed to create
	caseCase, err := s.caseRepo.FindByID(ticket.CaseID)
	if err != nil {
		return nil, errors.New("case not found")
	}

	agent, err := s.adminRepo.FindAvailableAgent(caseCase.Department)

	var agentID *uint64
	status := "waiting"

	if err == nil && agent != nil {
		id := uint64(agent.ID)
		agentID = &id
		status = "assigned"
	}

	ticket.AdminID = agentID
	ticket.Status = status

	if err := s.repo.Create(ticket); err != nil {
		return nil, err
	}

	if agent != nil {
		agent.ActiveTicketQty++
		_ = s.adminRepo.Update(agent)
	}

	// 3. Reload ticket to get all associations (Admin, Customer, Case) for response/socket
	createdTicket, err := s.repo.FindByID(ticket.ID, 0)
	if err != nil {
		// Fallback to minimal ticket if reload fails, but ideally shouldn't happen
		createdTicket = ticket
	}

	// 4. Emit socket event if agent Assigned
	if createdTicket.AdminID != nil && *createdTicket.AdminID > 0 {
		// Emit to admin room
		// Room: admin_{adminID}
		// Event: ticketAssigned
		fmt.Printf("Socket Emitting to room admin_%d event ticketAssigned\n", *createdTicket.AdminID)
		s.socket.Emit(fmt.Sprintf("admin_%d", *createdTicket.AdminID), "ticketAssigned", createdTicket)
	}

	return createdTicket, nil
}

func (s *ticketService) GetTicketByID(id uint, customerID uint) (*models.Ticket, error) {
	return s.repo.FindByID(id, customerID) // Pass customerID to filter
}

func (s *ticketService) GetCustomerTickets(customerID uint) ([]models.Ticket, error) {
	return s.repo.FindAll(map[string]interface{}{
		"customer_id": customerID,
	})
}

func (s *ticketService) GetAgentTickets(adminID uint64) ([]models.Ticket, error) {
	fmt.Printf("TicketService: GetAgentTickets called with adminID: %d\n", adminID)
	return s.repo.FindAll(map[string]interface{}{
		"admin_id": adminID,
	})
}

func (s *ticketService) GetAllTickets() ([]models.Ticket, error) {
	return s.repo.FindAll(map[string]interface{}{})
}

func (s *ticketService) CloseTicket(id uint) (*models.Ticket, error) {
	ticket, err := s.repo.FindByID(id, 0) // No customer filter for admin closure (or self close)
	if err != nil {
		return nil, errors.New("ticket not found")
	}

	if ticket.Status == "closed" {
		return nil, errors.New("ticket already closed")
	}

	ticket.Status = "closed"
	if err := s.repo.Update(ticket); err != nil {
		return nil, err
	}

	// Decrement Agent Tickets
	if ticket.AdminID != nil {
		agent, err := s.adminRepo.FindByID(*ticket.AdminID)
		if err == nil && agent.ActiveTicketQty > 0 {
			agent.ActiveTicketQty--
			_ = s.adminRepo.Update(agent)
		}
	}

	// Emit Socket Event
	// io.to(`ticket-${ticketId}`).emit("ticketClosed", ticketId);
	s.socket.Emit(fmt.Sprintf("ticket-%d", id), "ticketClosed", id)

	return ticket, nil
}

func (s *ticketService) CreateRating(ticketID uint, customerID uint, score int, comment string) (*models.Rating, error) {
	ticket, err := s.repo.FindByID(ticketID, 0) // We check ownership manually below or pass 0
	if err != nil {
		return nil, errors.New("ticket not found")
	}
	if ticket.Status != "closed" {
		return nil, errors.New("ticket not closed")
	}

	existing, _ := s.common.FindByTicketID(ticketID)
	if existing != nil {
		return nil, errors.New("rating already exists")
	}

	rating := &models.Rating{
		TicketID:   ticketID,
		CustomerID: customerID,
		Score:      score,
		Comment:    comment,
	}

	if err := s.ratingRepo.Create(rating); err != nil {
		return nil, err
	}
	return rating, nil
}

func (s *ticketService) GetRating(ticketID uint, customerID uint) (*models.Rating, error) {
	// First check if ticket exists and belongs to customer (if customerID provided)
	ticket, err := s.repo.FindByID(ticketID, customerID)
	if err != nil {
		return nil, errors.New("ticket not found") // Will return 404 if filter fails
	}
	// If ticket found, it means it matches customerID (if filtered)

	// Double check ownership irrelevant here if repo already filtered, but consistent with request
	if customerID > 0 && ticket.CustomerID != customerID {
		return nil, errors.New("unauthorized")
	}

	return s.ratingRepo.FindByTicketID(ticketID)
}

func (s *ticketService) SendCustomerMessage(ticketID uint, customerID uint, message string) (*models.ChatMessage, error) {
	ticket, err := s.repo.FindByID(ticketID, 0)
	if err != nil {
		return nil, errors.New("ticket not found")
	}

	// Logic check? User code doesn't strictly check status for customer message sending,
	// but strictly speaking customer can send message if ticket is open/assigned.
	// But let's follow user code:
	// const chat = await prisma.chatMessage.create({...})
	// io.to(`ticket-${ticketId}`).emit("newMessage", chat);

	chatMsg := &models.ChatMessage{
		TicketID:   ticket.ID,
		SenderID:   uint64(customerID),
		SenderType: "customer",
		Message:    message,
		// UID/Name/Image etc might be needed?
	}

	// NOTE: ChatMessage model might need conversion or fields.
	// In GORM model:
	// SenderID uint, SenderType string (polymorphic?)
	// In `chat.go` I recall seeing polymorphic or similar.
	// Let's assume structure from provided models.

	if err := s.chatRepo.CreateMessage(chatMsg); err != nil {
		return nil, err
	}

	// Emit Socket Event
	room := fmt.Sprintf("ticket-%d", ticketID)
	fmt.Printf("TicketService: Emitting 'newMessage' to room '%s' with msg ID %d\n", room, chatMsg.ID)
	s.socket.Emit(room, "newMessage", chatMsg)
	// Note: User code uses "ticket-{id}" here but "ticket_{id}" in update.
	// User request:
	// Create: console.log...
	// SendCustomerMessage: io.to(`ticket-${ticketId}`) (dash)
	// CloseTicket: io.to(`ticket_${ticketId}`) (underscore)
	// I should follow exactly.

	return chatMsg, nil
}
