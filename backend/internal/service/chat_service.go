package service

import (
	"customer-help-center-backend/internal/models"
	"customer-help-center-backend/internal/repository"
	"errors"
	"fmt"
	"log"
)

type ChatService interface {
	SendMessage(message *models.ChatMessage) error

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

	SendAgentMessage(
		ticketID uint,
		agentID uint64,
		message string,
		mediaURL string,
		mediaID string,
		mediaType string,
		audioDuration float64,
		tempID int64,
	) (*models.ChatMessage, error)

	GetTicketHistory(ticketID uint) ([]models.ChatMessage, error)
	MarkRead(messageIDs []uint, adminID uint64) error
	MarkTicketAsRead(ticketID uint, readerRole string) error
}

type chatService struct {
	repo       repository.ChatRepository
	ticketRepo repository.TicketRepository
	socket     SocketService
}

func NewChatService(repo repository.ChatRepository, ticketRepo repository.TicketRepository, socket SocketService) ChatService {
	return &chatService{
		repo:       repo,
		ticketRepo: ticketRepo,
		socket:     socket,
	}
}

func (s *chatService) SendMessage(message *models.ChatMessage) error {
	return s.repo.CreateMessage(message)
}

func (s *chatService) SendCustomerMessage(
	ticketID uint,
	customerID uint,
	message string,
	mediaURL string,
	mediaID string,
	mediaType string,
	audioDuration float64,
	tempID int64,
) (*models.ChatMessage, error) {

	ticket, err := s.ticketRepo.FindByID(ticketID, 0)
	if err != nil {
		return nil, errors.New("ticket not found")
	}

	if ticket.CustomerID != customerID {
		return nil, errors.New("unauthorized: ticket does not belong to customer")
	}

	if ticket.Status == "closed" {
		return nil, errors.New("chat not allowed: ticket is closed")
	}

	// Must contain either text or media
	if message == "" && mediaURL == "" {
		return nil, errors.New("message or media is required")
	}

	chatMsg := &models.ChatMessage{
		TicketID:      ticketID,
		SenderID:      uint64(customerID),
		SenderType:    "customer",
		Message:       message,
		MediaURL:      mediaURL,
		MediaID:       mediaID,
		MediaType:     mediaType,     // "image" | "audio"
		AudioDuration: audioDuration, // only for audio
		TempID:        tempID,
	}

	if err := s.repo.CreateMessage(chatMsg); err != nil {
		return nil, err
	}

	s.socket.Emit(fmt.Sprintf("ticket-%d", ticketID), "newMessage", chatMsg)
	return chatMsg, nil
}

func (s *chatService) SendAgentMessage(
	ticketID uint,
	agentID uint64,
	message string,
	mediaURL string,
	mediaID string,
	mediaType string,
	audioDuration float64,
	tempID int64,
) (*models.ChatMessage, error) {

	ticket, err := s.ticketRepo.FindByID(ticketID, 0)
	if err != nil {
		return nil, errors.New("ticket not found")
	}

	if ticket.Status == "closed" {
		return nil, errors.New("chat not allowed: ticket is closed")
	}

	if ticket.Status != "assigned" {
		return nil, errors.New("chat not allowed: ticket not assigned")
	}

	if message == "" && mediaURL == "" {
		return nil, errors.New("message or media is required")
	}

	chatMsg := &models.ChatMessage{
		TicketID:      ticketID,
		SenderID:      agentID,
		SenderType:    "agent",
		Message:       message,
		MediaURL:      mediaURL,
		MediaID:       mediaID,
		MediaType:     mediaType,
		AudioDuration: audioDuration,
		TempID:        tempID,
	}

	if err := s.repo.CreateMessage(chatMsg); err != nil {
		return nil, err
	}

	s.socket.Emit(fmt.Sprintf("ticket-%d", ticketID), "newMessage", chatMsg)
	return chatMsg, nil
}

func (s *chatService) GetTicketHistory(ticketID uint) ([]models.ChatMessage, error) {
	return s.repo.GetMessagesByTicketID(ticketID)
}

func (s *chatService) MarkRead(messageIDs []uint, adminID uint64) error {
	return s.repo.MarkMessagesAsRead(messageIDs, adminID)
}

func (s *chatService) MarkTicketAsRead(ticketID uint, readerRole string) error {
	if err := s.repo.MarkTicketMessagesAsRead(ticketID, readerRole); err != nil {
		return err
	}

	room := fmt.Sprintf("ticket-%d", ticketID)
	log.Printf("ChatService: Emitting 'messagesRead' to room '%s' by %s", room, readerRole)

	s.socket.Emit(room, "messagesRead", map[string]interface{}{
		"ticketId": ticketID,
		"readBy":   readerRole,
	})

	return nil
}

// package service

// import (
// 	"customer-help-center-backend/internal/models"
// 	"customer-help-center-backend/internal/repository"
// 	"errors"
// 	"fmt"
// 	"log"
// )

// type ChatService interface {
// 	SendMessage(message *models.ChatMessage) error
// 	SendCustomerMessage(ticketID uint, customerID uint, message string, tempID int64) (*models.ChatMessage, error)
// 	SendAgentMessage(ticketID uint, agentID uint, message string, tempID int64) (*models.ChatMessage, error)
// 	GetTicketHistory(ticketID uint) ([]models.ChatMessage, error)
// 	MarkRead(messageIDs []uint, adminID uint) error
// 	MarkTicketAsRead(ticketID uint, readerRole string) error
// }

// type chatService struct {
// 	repo       repository.ChatRepository
// 	ticketRepo repository.TicketRepository
// 	socket     SocketService
// }

// func NewChatService(repo repository.ChatRepository, ticketRepo repository.TicketRepository, socket SocketService) ChatService {
// 	return &chatService{
// 		repo:       repo,
// 		ticketRepo: ticketRepo,
// 		socket:     socket,
// 	}
// }

// func (s *chatService) SendMessage(message *models.ChatMessage) error {
// 	return s.repo.CreateMessage(message)
// }

// func (s *chatService) SendCustomerMessage(ticketID uint, customerID uint, message string, tempID int64) (*models.ChatMessage, error) {
// 	// Pass 0 for customerID to keep manual ownership check below for explicit "unauthorized" error
// 	ticket, err := s.ticketRepo.FindByID(ticketID, 0)
// 	if err != nil {
// 		return nil, errors.New("ticket not found")
// 	}

// 	if ticket.CustomerID != customerID {
// 		return nil, errors.New("unauthorized: ticket does not belong to customer")
// 	}

// 	if ticket.Status == "closed" {
// 		return nil, errors.New("chat not allowed: ticket is closed")
// 	}

// 	chatMsg := &models.ChatMessage{
// 		TicketID:   ticketID,
// 		SenderID:   customerID,
// 		SenderType: "customer",
// 		Message:    message,
// 		TempID:     tempID,
// 	}

// 	if err := s.repo.CreateMessage(chatMsg); err != nil {
// 		return nil, err
// 	}

// 	s.socket.Emit(fmt.Sprintf("ticket-%d", ticketID), "newMessage", chatMsg)
// 	return chatMsg, nil
// }

// func (s *chatService) SendAgentMessage(ticketID uint, agentID uint, message string, tempID int64) (*models.ChatMessage, error) {
// 	ticket, err := s.ticketRepo.FindByID(ticketID, 0)
// 	if err != nil {
// 		return nil, errors.New("ticket not found")
// 	}

// 	if ticket.Status == "closed" {
// 		return nil, errors.New("chat not allowed: ticket is closed")
// 	}

// 	if ticket.Status != "assigned" {
// 		return nil, errors.New("chat not allowed: ticket not assigned")
// 	}

// 	chatMsg := &models.ChatMessage{
// 		TicketID:   ticketID,
// 		SenderID:   agentID,
// 		SenderType: "agent",
// 		Message:    message,
// 		TempID:     tempID,
// 	}

// 	if err := s.repo.CreateMessage(chatMsg); err != nil {
// 		return nil, err
// 	}

// 	s.socket.Emit(fmt.Sprintf("ticket-%d", ticketID), "newMessage", chatMsg)
// 	return chatMsg, nil
// }

// func (s *chatService) GetTicketHistory(ticketID uint) ([]models.ChatMessage, error) {
// 	return s.repo.GetMessagesByTicketID(ticketID)
// }

// func (s *chatService) MarkRead(messageIDs []uint, adminID uint) error {
// 	return s.repo.MarkMessagesAsRead(messageIDs, adminID)
// }

// func (s *chatService) MarkTicketAsRead(ticketID uint, readerRole string) error {
// 	if err := s.repo.MarkTicketMessagesAsRead(ticketID, readerRole); err != nil {
// 		return err
// 	}
// 	// Notify via socket that messages were read
// 	room := fmt.Sprintf("ticket-%d", ticketID)
// 	log.Printf("ChatService: Emitting 'messagesRead' to room '%s' by %s", room, readerRole)
// 	s.socket.Emit(room, "messagesRead", map[string]interface{}{
// 		"ticketId": ticketID,
// 		"readBy":   readerRole,
// 	})
// 	return nil
// }
