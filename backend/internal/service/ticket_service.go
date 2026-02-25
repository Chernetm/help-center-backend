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

	SendCustomerMessage(
		ticketID uint,
		customerID uint,
		message string,
		mediaURL string,
		mediaID string,
		mediaType string,
		audioDuration float64,
		tempID int64,
	) (*models.ChatMessage, error)

	DeleteTicket(id uint, customerID uint) error
	CleanUnassignedTickets() error
	DeleteAllTickets() error
}

type ticketService struct {
	repo       repository.TicketRepository
	caseRepo   repository.CaseRepository
	adminRepo  repository.AdminRepository
	ratingRepo repository.RatingRepository
	chatRepo   repository.ChatRepository
	socket     SocketService
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

//////////////////////////////////////////////////////////////
// CREATE TICKET
//////////////////////////////////////////////////////////////

func (s *ticketService) CreateTicket(ticket *models.Ticket) (*models.Ticket, error) {

	if ticket.CustomerID == 0 {
		return nil, errors.New("customer ID required")
	}

	// Prevent duplicate active ticket
	existing, _ := s.repo.FindActiveTicket(ticket.CustomerID, ticket.CaseID)
	if existing != nil {
		return existing, nil
	}

	caseObj, err := s.caseRepo.FindByID(ticket.CaseID)
	if err != nil {
		return nil, errors.New("case not found")
	}

	agent, _ := s.adminRepo.FindAvailableAgent(caseObj.Department)
	if agent == nil {
		return nil, errors.New("no available agent")
	}

	id := uint64(agent.ID)
	ticket.AdminID = &id
	ticket.Status = "assigned"

	if err := s.repo.Create(ticket); err != nil {
		return nil, err
	}

	agent.ActiveTicketQty++
	_ = s.adminRepo.Update(agent)

	createdTicket, err := s.repo.FindByID(ticket.ID, 0)
	if err != nil {
		return ticket, nil
	}

	// ✅ Notify admin (admin will join ticket room from frontend)
	adminRoom := fmt.Sprintf("admin_%d", *createdTicket.AdminID)
	s.socket.Emit(adminRoom, "ticketAssigned", createdTicket)

	return createdTicket, nil
}

//////////////////////////////////////////////////////////////
// SEND CUSTOMER MESSAGE
//////////////////////////////////////////////////////////////

func (s *ticketService) SendCustomerMessage(
	ticketID uint,
	customerID uint,
	message string,
	mediaURL string,
	mediaID string,
	mediaType string,
	audioDuration float64,
	tempID int64,
) (*models.ChatMessage, error) {

	ticket, err := s.repo.FindByID(ticketID, 0)
	if err != nil {
		return nil, errors.New("ticket not found")
	}

	if ticket.Status == "closed" {
		return nil, errors.New("ticket is closed")
	}

	chatMsg := &models.ChatMessage{
		TicketID:      ticket.ID,
		SenderID:      uint64(customerID),
		SenderType:    "customer",
		Message:       message,
		MediaURL:      mediaURL,
		MediaID:       mediaID,
		MediaType:     mediaType,
		AudioDuration: audioDuration,
		TempID:        tempID,
	}

	if err := s.chatRepo.CreateMessage(chatMsg); err != nil {
		return nil, err
	}

	room := fmt.Sprintf("ticket-%d", ticketID)
	s.socket.Emit(room, "newMessage", chatMsg)

	return chatMsg, nil
}

//////////////////////////////////////////////////////////////
// CLOSE TICKET
//////////////////////////////////////////////////////////////

func (s *ticketService) CloseTicket(id uint) (*models.Ticket, error) {

	ticket, err := s.repo.FindByID(id, 0)
	if err != nil {
		return nil, errors.New("ticket not found")
	}

	if ticket.Status == "closed" {
		return nil, errors.New("already closed")
	}

	ticket.Status = "closed"

	if err := s.repo.Update(ticket); err != nil {
		return nil, err
	}

	if ticket.AdminID != nil {
		agent, err := s.adminRepo.FindByID(*ticket.AdminID)
		if err == nil && agent.ActiveTicketQty > 0 {
			agent.ActiveTicketQty--
			_ = s.adminRepo.Update(agent)
		}
	}

	room := fmt.Sprintf("ticket-%d", id)
	s.socket.Emit(room, "ticketClosed", id)

	return ticket, nil
}

//////////////////////////////////////////////////////////////
// GETTERS
//////////////////////////////////////////////////////////////

func (s *ticketService) GetTicketByID(id uint, customerID uint) (*models.Ticket, error) {
	return s.repo.FindByID(id, customerID)
}

func (s *ticketService) GetCustomerTickets(customerID uint) ([]models.Ticket, error) {
	return s.repo.FindAll(map[string]interface{}{
		"customer_id": customerID,
	})
}

func (s *ticketService) GetAgentTickets(adminID uint64) ([]models.Ticket, error) {
	return s.repo.FindAll(map[string]interface{}{
		"admin_id": adminID,
	})
}

func (s *ticketService) GetAllTickets() ([]models.Ticket, error) {
	return s.repo.FindAll(map[string]interface{}{})
}

//////////////////////////////////////////////////////////////
// RATING
//////////////////////////////////////////////////////////////

func (s *ticketService) CreateRating(ticketID uint, customerID uint, score int, comment string) (*models.Rating, error) {

	ticket, err := s.repo.FindByID(ticketID, 0)
	if err != nil {
		return nil, errors.New("ticket not found")
	}

	if ticket.Status != "closed" {
		return nil, errors.New("ticket not closed")
	}

	existing, _ := s.ratingRepo.FindByTicketID(ticketID)
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
	return s.ratingRepo.FindByTicketID(ticketID)
}

//////////////////////////////////////////////////////////////
// DELETE
//////////////////////////////////////////////////////////////

func (s *ticketService) DeleteTicket(id uint, customerID uint) error {

	ticket, err := s.repo.FindByID(id, customerID)
	if err != nil {
		return errors.New("ticket not found")
	}

	if ticket.AdminID != nil && *ticket.AdminID > 0 {
		return errors.New("cannot delete assigned ticket")
	}

	return s.repo.Delete(id)
}

func (s *ticketService) CleanUnassignedTickets() error {
	return s.repo.DeleteUnassignedTickets()
}

func (s *ticketService) DeleteAllTickets() error {
	return s.repo.DeleteAllTickets()
}

// package service

// import (
// 	"customer-help-center-backend/internal/models"
// 	"customer-help-center-backend/internal/repository"
// 	"errors"
// 	"fmt"
// )

// type TicketService interface {
// 	CreateTicket(ticket *models.Ticket) (*models.Ticket, error)
// 	GetTicketByID(id uint, customerID uint) (*models.Ticket, error)
// 	GetCustomerTickets(customerID uint) ([]models.Ticket, error)
// 	GetAgentTickets(agentID uint64) ([]models.Ticket, error)
// 	GetAllTickets() ([]models.Ticket, error)
// 	CloseTicket(id uint) (*models.Ticket, error)
// 	CreateRating(ticketID uint, customerID uint, score int, comment string) (*models.Rating, error)
// 	GetRating(ticketID uint, customerID uint) (*models.Rating, error)

// 	// ✅ UPDATED
// 	SendCustomerMessage(
// 		ticketID uint,
// 		customerID uint,
// 		message string,
// 		mediaURL string,
// 		mediaID string,
// 		mediaType string,
// 		audioDuration float64,
// 		tempID int64,
// 	) (*models.ChatMessage, error)

// 	DeleteTicket(id uint, customerID uint) error
// 	CleanUnassignedTickets() error
// }

// type ticketService struct {
// 	repo       repository.TicketRepository
// 	caseRepo   repository.CaseRepository
// 	adminRepo  repository.AdminRepository
// 	ratingRepo repository.RatingRepository
// 	chatRepo   repository.ChatRepository
// 	socket     SocketService
// }

// func NewTicketService(
// 	repo repository.TicketRepository,
// 	caseRepo repository.CaseRepository,
// 	adminRepo repository.AdminRepository,
// 	ratingRepo repository.RatingRepository,
// 	chatRepo repository.ChatRepository,
// 	socket SocketService,
// ) TicketService {
// 	return &ticketService{
// 		repo:       repo,
// 		caseRepo:   caseRepo,
// 		adminRepo:  adminRepo,
// 		ratingRepo: ratingRepo,
// 		chatRepo:   chatRepo,
// 		socket:     socket,
// 	}
// }

// func (s *ticketService) CreateTicket(ticket *models.Ticket) (*models.Ticket, error) {

// 	if ticket.CustomerID == 0 {
// 		return nil, errors.New("customer ID required")
// 	}

// 	// Prevent duplicate active ticket
// 	existing, _ := s.repo.FindActiveTicket(ticket.CustomerID, ticket.CaseID)
// 	if existing != nil {
// 		return existing, nil
// 	}

// 	caseObj, err := s.caseRepo.FindByID(ticket.CaseID)
// 	if err != nil {
// 		return nil, errors.New("case not found")
// 	}

// 	agent, _ := s.adminRepo.FindAvailableAgent(caseObj.Department)
// 	if agent == nil {
// 		return nil, errors.New("no available agent")
// 	}

// 	id := uint64(agent.ID)
// 	ticket.AdminID = &id
// 	ticket.Status = "assigned"

// 	if err := s.repo.Create(ticket); err != nil {
// 		return nil, err
// 	}

// 	agent.ActiveTicketQty++
// 	_ = s.adminRepo.Update(agent)

// 	createdTicket, err := s.repo.FindByID(ticket.ID, 0)
// 	if err != nil {
// 		return ticket, nil
// 	}

// 	// -------------------------
// 	// 🔥 REALTIME FIX STARTS HERE
// 	// -------------------------

// 	adminRoom := fmt.Sprintf("admin_%d", *createdTicket.AdminID)
// 	ticketRoom := fmt.Sprintf("ticket-%d", createdTicket.ID)

// 	// 1. Notify admin of assignment
// 	s.socket.Emit(adminRoom, "ticketAssigned", createdTicket)

// 	// 2. FORCE admin socket into ticket room
// 	s.socket.Emit(adminRoom, "forceJoinTicket", ticketRoom)

// 	// -------------------------
// 	// 🔥 REALTIME FIX ENDS HERE
// 	// -------------------------

// 	return createdTicket, nil
// }

// //////////////////////////////////////////////////////////////
// // SEND CUSTOMER MESSAGE (FULLY UPDATED)
// //////////////////////////////////////////////////////////////

// func (s *ticketService) SendCustomerMessage(
// 	ticketID uint,
// 	customerID uint,
// 	message string,
// 	mediaURL string,
// 	mediaID string,
// 	mediaType string,
// 	audioDuration float64,
// 	tempID int64,
// ) (*models.ChatMessage, error) {

// 	ticket, err := s.repo.FindByID(ticketID, 0)
// 	if err != nil {
// 		return nil, errors.New("ticket not found")
// 	}

// 	if ticket.Status == "closed" {
// 		return nil, errors.New("ticket is closed")
// 	}

// 	chatMsg := &models.ChatMessage{
// 		TicketID:      ticket.ID,
// 		SenderID:      uint64(customerID),
// 		SenderType:    "customer",
// 		Message:       message,
// 		MediaURL:      mediaURL,
// 		MediaID:       mediaID,
// 		MediaType:     mediaType,
// 		AudioDuration: audioDuration,
// 		IsRead:        false,
// 		TempID:        tempID, // ✅ Echo back tempId
// 	}

// 	if err := s.chatRepo.CreateMessage(chatMsg); err != nil {
// 		return nil, err
// 	}

// 	// 🔥 IMPORTANT — consistent room name
// 	room := fmt.Sprintf("ticket-%d", ticketID)

// 	fmt.Printf("Emitting newMessage to room %s (DB ID: %d, TempID: %d)\n",
// 		room, chatMsg.ID, tempID)

// 	s.socket.Emit(room, "newMessage", chatMsg)

// 	return chatMsg, nil
// }

// //////////////////////////////////////////////////////////////
// // CLOSE TICKET
// //////////////////////////////////////////////////////////////

// func (s *ticketService) CloseTicket(id uint) (*models.Ticket, error) {

// 	ticket, err := s.repo.FindByID(id, 0)
// 	if err != nil {
// 		return nil, errors.New("ticket not found")
// 	}

// 	if ticket.Status == "closed" {
// 		return nil, errors.New("already closed")
// 	}

// 	ticket.Status = "closed"

// 	if err := s.repo.Update(ticket); err != nil {
// 		return nil, err
// 	}

// 	if ticket.AdminID != nil {
// 		agent, err := s.adminRepo.FindByID(*ticket.AdminID)
// 		if err == nil && agent.ActiveTicketQty > 0 {
// 			agent.ActiveTicketQty--
// 			_ = s.adminRepo.Update(agent)
// 		}
// 	}

// 	room := fmt.Sprintf("ticket-%d", id)
// 	s.socket.Emit(room, "ticketClosed", id)

// 	return ticket, nil
// }

// //////////////////////////////////////////////////////////////
// // GETTERS
// //////////////////////////////////////////////////////////////

// func (s *ticketService) GetTicketByID(id uint, customerID uint) (*models.Ticket, error) {
// 	return s.repo.FindByID(id, customerID)
// }

// func (s *ticketService) GetCustomerTickets(customerID uint) ([]models.Ticket, error) {
// 	return s.repo.FindAll(map[string]interface{}{
// 		"customer_id": customerID,
// 	})
// }

// func (s *ticketService) GetAgentTickets(adminID uint64) ([]models.Ticket, error) {
// 	return s.repo.FindAll(map[string]interface{}{
// 		"admin_id": adminID,
// 	})
// }

// func (s *ticketService) GetAllTickets() ([]models.Ticket, error) {
// 	return s.repo.FindAll(map[string]interface{}{})
// }

// //////////////////////////////////////////////////////////////
// // RATING
// //////////////////////////////////////////////////////////////

// func (s *ticketService) CreateRating(ticketID uint, customerID uint, score int, comment string) (*models.Rating, error) {

// 	ticket, err := s.repo.FindByID(ticketID, 0)
// 	if err != nil {
// 		return nil, errors.New("ticket not found")
// 	}

// 	if ticket.Status != "closed" {
// 		return nil, errors.New("ticket not closed")
// 	}

// 	existing, _ := s.ratingRepo.FindByTicketID(ticketID)
// 	if existing != nil {
// 		return nil, errors.New("rating already exists")
// 	}

// 	rating := &models.Rating{
// 		TicketID:   ticketID,
// 		CustomerID: customerID,
// 		Score:      score,
// 		Comment:    comment,
// 	}

// 	if err := s.ratingRepo.Create(rating); err != nil {
// 		return nil, err
// 	}

// 	return rating, nil
// }

// func (s *ticketService) GetRating(ticketID uint, customerID uint) (*models.Rating, error) {
// 	return s.ratingRepo.FindByTicketID(ticketID)
// }

// //////////////////////////////////////////////////////////////
// // DELETE
// //////////////////////////////////////////////////////////////

// func (s *ticketService) DeleteTicket(id uint, customerID uint) error {

// 	ticket, err := s.repo.FindByID(id, customerID)
// 	if err != nil {
// 		return errors.New("ticket not found")
// 	}

// 	if ticket.AdminID != nil && *ticket.AdminID > 0 {
// 		return errors.New("cannot delete assigned ticket")
// 	}

// 	return s.repo.Delete(id)
// }

// func (s *ticketService) CleanUnassignedTickets() error {
// 	return s.repo.DeleteUnassignedTickets()
// }

// package service

// import (
// 	"customer-help-center-backend/internal/models"
// 	"customer-help-center-backend/internal/repository"
// 	"errors"
// 	"fmt"
// )

// type TicketService interface {
// 	CreateTicket(ticket *models.Ticket) (*models.Ticket, error)
// 	GetTicketByID(id uint, customerID uint) (*models.Ticket, error)
// 	GetCustomerTickets(customerID uint) ([]models.Ticket, error)
// 	GetAgentTickets(agentID uint64) ([]models.Ticket, error)
// 	GetAllTickets() ([]models.Ticket, error)
// 	CloseTicket(id uint) (*models.Ticket, error)
// 	CreateRating(ticketID uint, customerID uint, score int, comment string) (*models.Rating, error)
// 	GetRating(ticketID uint, customerID uint) (*models.Rating, error)
// 	SendCustomerMessage(ticketID uint, customerID uint, message string) (*models.ChatMessage, error)
// 	DeleteTicket(id uint, customerID uint) error
// 	CleanUnassignedTickets() error
// }

// type ticketService struct {
// 	repo       repository.TicketRepository
// 	caseRepo   repository.CaseRepository
// 	adminRepo  repository.AdminRepository
// 	ratingRepo repository.RatingRepository
// 	chatRepo   repository.ChatRepository
// 	socket     SocketService
// 	common     repository.AuditRepository
// }

// func NewTicketService(
// 	repo repository.TicketRepository,
// 	caseRepo repository.CaseRepository,
// 	adminRepo repository.AdminRepository,
// 	ratingRepo repository.RatingRepository,
// 	chatRepo repository.ChatRepository,
// 	socket SocketService,
// ) TicketService {
// 	return &ticketService{
// 		repo:       repo,
// 		caseRepo:   caseRepo,
// 		adminRepo:  adminRepo,
// 		ratingRepo: ratingRepo,
// 		chatRepo:   chatRepo,
// 		socket:     socket,
// 	}
// }
// func (s *ticketService) CreateTicket(ticket *models.Ticket) (*models.Ticket, error) {
// 	if ticket.CustomerID == 0 {
// 		return nil, errors.New("customer ID is required")
// 	}

// 	// 1. Check for existing active ticket for this customer and case
// 	existingTicket, _ := s.repo.FindActiveTicket(ticket.CustomerID, ticket.CaseID)
// 	if existingTicket != nil {
// 		// Found an existing open/assigned/waiting ticket, return it
// 		return existingTicket, nil
// 	}

// 	// 2. No existing ticket, proceed to create
// 	caseCase, err := s.caseRepo.FindByID(ticket.CaseID)
// 	if err != nil {
// 		return nil, errors.New("case not found")
// 	}

// 	agent, _ := s.adminRepo.FindAvailableAgent(caseCase.Department)
// 	if agent == nil {
// 		return nil, errors.New("Sorry no ticket generation message")
// 	}

// 	id := uint64(agent.ID)
// 	agentID := &id
// 	status := "assigned"

// 	ticket.AdminID = agentID
// 	ticket.Status = status

// 	if err := s.repo.Create(ticket); err != nil {
// 		return nil, err
// 	}

// 	// Update agent ticket count
// 	agent.ActiveTicketQty++
// 	_ = s.adminRepo.Update(agent)

// 	// 3. Reload ticket to get all associations (Admin, Customer, Case) for response/socket
// 	createdTicket, err := s.repo.FindByID(ticket.ID, 0)
// 	if err != nil {
// 		// Fallback to minimal ticket if reload fails, but ideally shouldn't happen
// 		createdTicket = ticket
// 	}

// 	// 4. Emit socket event if agent Assigned
// 	if createdTicket.AdminID != nil && *createdTicket.AdminID > 0 {
// 		// Emit to admin room
// 		// Room: admin_{adminID}
// 		// Event: ticketAssigned
// 		fmt.Printf("Socket Emitting to room admin_%d event ticketAssigned\n", *createdTicket.AdminID)
// 		s.socket.Emit(fmt.Sprintf("admin_%d", *createdTicket.AdminID), "ticketAssigned", createdTicket)
// 	}

// 	return createdTicket, nil
// }

// func (s *ticketService) GetTicketByID(id uint, customerID uint) (*models.Ticket, error) {
// 	return s.repo.FindByID(id, customerID) // Pass customerID to filter
// }

// func (s *ticketService) GetCustomerTickets(customerID uint) ([]models.Ticket, error) {
// 	return s.repo.FindAll(map[string]interface{}{
// 		"customer_id": customerID,
// 	})
// }

// func (s *ticketService) GetAgentTickets(adminID uint64) ([]models.Ticket, error) {
// 	fmt.Printf("TicketService: GetAgentTickets called with adminID: %d\n", adminID)
// 	return s.repo.FindAll(map[string]interface{}{
// 		"admin_id": adminID,
// 	})
// }

// func (s *ticketService) GetAllTickets() ([]models.Ticket, error) {
// 	return s.repo.FindAll(map[string]interface{}{})
// }

// func (s *ticketService) CloseTicket(id uint) (*models.Ticket, error) {
// 	ticket, err := s.repo.FindByID(id, 0) // No customer filter for admin closure (or self close)
// 	if err != nil {
// 		return nil, errors.New("ticket not found")
// 	}

// 	if ticket.Status == "closed" {
// 		return nil, errors.New("ticket already closed")
// 	}

// 	ticket.Status = "closed"
// 	if err := s.repo.Update(ticket); err != nil {
// 		return nil, err
// 	}

// 	// Decrement Agent Tickets
// 	if ticket.AdminID != nil {
// 		agent, err := s.adminRepo.FindByID(*ticket.AdminID)
// 		if err == nil && agent.ActiveTicketQty > 0 {
// 			agent.ActiveTicketQty--
// 			_ = s.adminRepo.Update(agent)
// 		}
// 	}

// 	// Emit Socket Event
// 	// io.to(`ticket-${ticketId}`).emit("ticketClosed", ticketId);
// 	s.socket.Emit(fmt.Sprintf("ticket-%d", id), "ticketClosed", id)

// 	return ticket, nil
// }

// func (s *ticketService) CreateRating(ticketID uint, customerID uint, score int, comment string) (*models.Rating, error) {
// 	ticket, err := s.repo.FindByID(ticketID, 0) // We check ownership manually below or pass 0
// 	if err != nil {
// 		return nil, errors.New("ticket not found")
// 	}
// 	if ticket.Status != "closed" {
// 		return nil, errors.New("ticket not closed")
// 	}

// 	existing, _ := s.common.FindByTicketID(ticketID)
// 	if existing != nil {
// 		return nil, errors.New("rating already exists")
// 	}

// 	rating := &models.Rating{
// 		TicketID:   ticketID,
// 		CustomerID: customerID,
// 		Score:      score,
// 		Comment:    comment,
// 	}

// 	if err := s.ratingRepo.Create(rating); err != nil {
// 		return nil, err
// 	}
// 	return rating, nil
// }

// func (s *ticketService) GetRating(ticketID uint, customerID uint) (*models.Rating, error) {
// 	// First check if ticket exists and belongs to customer (if customerID provided)
// 	ticket, err := s.repo.FindByID(ticketID, customerID)
// 	if err != nil {
// 		return nil, errors.New("ticket not found") // Will return 404 if filter fails
// 	}
// 	// If ticket found, it means it matches customerID (if filtered)

// 	// Double check ownership irrelevant here if repo already filtered, but consistent with request
// 	if customerID > 0 && ticket.CustomerID != customerID {
// 		return nil, errors.New("unauthorized")
// 	}

// 	return s.ratingRepo.FindByTicketID(ticketID)
// }

// func (s *ticketService) SendCustomerMessage(ticketID uint, customerID uint, message string) (*models.ChatMessage, error) {
// 	ticket, err := s.repo.FindByID(ticketID, 0)
// 	if err != nil {
// 		return nil, errors.New("ticket not found")
// 	}

// 	// Logic check? User code doesn't strictly check status for customer message sending,
// 	// but strictly speaking customer can send message if ticket is open/assigned.
// 	// But let's follow user code:
// 	// const chat = await prisma.chatMessage.create({...})
// 	// io.to(`ticket-${ticketId}`).emit("newMessage", chat);

// 	chatMsg := &models.ChatMessage{
// 		TicketID:   ticket.ID,
// 		SenderID:   uint64(customerID),
// 		SenderType: "customer",
// 		Message:    message,
// 		// UID/Name/Image etc might be needed?
// 	}

// 	// NOTE: ChatMessage model might need conversion or fields.
// 	// In GORM model:
// 	// SenderID uint, SenderType string (polymorphic?)
// 	// In `chat.go` I recall seeing polymorphic or similar.
// 	// Let's assume structure from provided models.

// 	if err := s.chatRepo.CreateMessage(chatMsg); err != nil {
// 		return nil, err
// 	}

// 	// Emit Socket Event
// 	room := fmt.Sprintf("ticket-%d", ticketID)
// 	fmt.Printf("TicketService: Emitting 'newMessage' to room '%s' with msg ID %d\n", room, chatMsg.ID)
// 	s.socket.Emit(room, "newMessage", chatMsg)
// 	// Note: User code uses "ticket-{id}" here but "ticket_{id}" in update.
// 	// User request:
// 	// Create: console.log...
// 	// SendCustomerMessage: io.to(`ticket-${ticketId}`) (dash)
// 	// CloseTicket: io.to(`ticket_${ticketId}`) (underscore)
// 	// I should follow exactly.

// 	return chatMsg, nil
// }

// func (s *ticketService) DeleteTicket(id uint, customerID uint) error {
// 	fmt.Printf("DeleteTicket [Service]: ID=%d, CustomerID=%d\n", id, customerID)
// 	ticket, err := s.repo.FindByID(id, customerID)
// 	if err != nil {
// 		fmt.Printf("DeleteTicket [Service]: Ticket not found (id=%d, customerID=%d): %v\n", id, customerID, err)
// 		return errors.New("ticket not found")
// 	}

// 	fmt.Printf("DeleteTicket [Service]: Found Ticket. Status=%s, AdminID=%v\n", ticket.Status, ticket.AdminID)

// 	// A ticket is considered "unassigned" if:
// 	// 1. AdminID is nul or 0
// 	// 2. OR status is 'waiting' or 'open' (if open means not yet assigned in some contexts, but usually 'waiting' is the key)
// 	isAssigned := false
// 	if ticket.AdminID != nil && *ticket.AdminID > 0 {
// 		isAssigned = true
// 	}

// 	// If status is "assigned", definitely don't delete
// 	if ticket.Status == "assigned" || ticket.Status == "closed" {
// 		// However, we only care about AdminID association as per user request
// 		if isAssigned {
// 			fmt.Printf("DeleteTicket [Service]: Blocked - Ticket is associated with an admin (AdminID=%d)\n", *ticket.AdminID)
// 			return errors.New("cannot delete an assigned ticket")
// 		}
// 	}

// 	fmt.Printf("DeleteTicket [Service]: Proceeding to repository delete ID=%d\n", id)
// 	return s.repo.Delete(id)
// }

// func (s *ticketService) CleanUnassignedTickets() error {
// 	fmt.Println("AutoCleanup [Service]: Starting cleanup of unassigned tickets")
// 	return s.repo.DeleteUnassignedTickets()
// }
