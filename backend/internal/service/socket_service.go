package service

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ================= INTERFACE =================

type SocketService interface {
	Emit(room, event string, payload interface{})
	Broadcast(event string, payload interface{})
	HandleConnections(w http.ResponseWriter, r *http.Request)
}

// ================= MODELS =================

type Client struct {
	Hub     *socketService
	Conn    *websocket.Conn
	Send    chan []byte
	Rooms   map[string]bool
	IsAdmin bool
	AdminID uint64
	UserID  uint
	Token   string
}

type SocketEvent struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
	Room    string      `json:"room,omitempty"`
}

// ================= WEBSOCKET =================

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ================= SERVICE =================

type socketService struct {
	adminService    AdminService
	customerService CustomerService

	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	rooms      map[string]map[*Client]bool

	mu sync.RWMutex
}

func NewSocketService(adminService AdminService, customerService CustomerService) SocketService {
	s := &socketService{
		adminService:    adminService,
		customerService: customerService,
		clients:         make(map[*Client]bool),
		register:        make(chan *Client, 1024),
		unregister:      make(chan *Client, 1024),
		rooms:           make(map[string]map[*Client]bool),
	}
	go s.run()
	return s
}

// ================= HUB LOOP =================

func (s *socketService) run() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("WS: CRITICAL Hub panic recovered: %v. Restarting loop...", r)
			go s.run()
		}
	}()

	for {
		select {
		case client := <-s.register:
			s.mu.Lock()
			s.clients[client] = true
			s.mu.Unlock()
			log.Printf("WS: Client registered (Total: %d)", len(s.clients))

		case client := <-s.unregister:
			s.mu.Lock()
			if _, ok := s.clients[client]; ok {
				s.removeClient(client)
			}
			s.mu.Unlock()
			log.Printf("WS: Client unregistered (Remaining: %d)", len(s.clients))
		}
	}
}

func (s *socketService) removeClient(client *Client) {
	delete(s.clients, client)

	for room := range client.Rooms {
		if roomClients, ok := s.rooms[room]; ok {
			delete(roomClients, client)
			if len(roomClients) == 0 {
				delete(s.rooms, room)
			}
		}
	}

	close(client.Send)
	client.Conn.Close()

	if client.IsAdmin && client.AdminID != 0 {
		// Only end session/set offline if no other connections exist for this admin
		stillConnected := false
		for other := range s.clients {
			if other.IsAdmin && other.AdminID == client.AdminID {
				stillConnected = true
				break
			}
		}

		if !stillConnected {
			log.Printf("WS: Admin %d fully disconnected. Ending session asynchronously.", client.AdminID)
			adminID := client.AdminID
			go func() {
				_ = s.adminService.EndWorkSession(adminID)
				_ = s.adminService.SetOnlineStatus(adminID, false)

				s.Broadcast("adminStatusChanged", map[string]interface{}{
					"adminId":  adminID,
					"isOnline": false,
				})
			}()
		} else {
			log.Printf("WS: Admin %d disconnected one session, but still has others active.", client.AdminID)
		}
	}
}

// ================= CONNECTION =================

func (s *socketService) HandleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WS upgrade error:", err)
		return
	}
	log.Printf("WS: Connection upgraded for %s", r.RemoteAddr)

	tokenCookie, _ := r.Cookie("adminToken")
	var tokenValue string
	if tokenCookie != nil {
		tokenValue = tokenCookie.Value
	}

	client := &Client{
		Hub:   s,
		Conn:  ws,
		Send:  make(chan []byte, 256),
		Rooms: make(map[string]bool),
		Token: tokenValue,
	}

	s.register <- client

	go client.writePump()
	go client.readPump()
}

// ================= EMIT =================

func (s *socketService) Emit(room, event string, payload interface{}) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	targets, ok := s.rooms[room]
	if !ok || len(targets) == 0 {
		return
	}

	msg := SocketEvent{
		Type:    event,
		Payload: payload,
		Room:    room,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Println("WS marshal error:", err)
		return
	}

	for client := range targets {
		select {
		case client.Send <- data:
		default:
			// Queue unregister instead of calling removeClient directly to avoid race
			log.Printf("WS: Unresponsiveness detected for client in room %s, unregistering...", room)
			select {
			case s.unregister <- client:
			default:
				log.Println("WS: Unregister channel full, dropping client via goroutine")
				go s.removeClient(client)
			}
		}
	}
	log.Printf("WS: Emitted %s to %d targets in room %s", event, len(targets), room)
}

func (s *socketService) Broadcast(event string, payload interface{}) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msg := SocketEvent{
		Type:    event,
		Payload: payload,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	for client := range s.clients {
		select {
		case client.Send <- data:
		default:
		}
	}
}

// ================= CLIENT =================

func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512000)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var event SocketEvent
		if err := json.Unmarshal(message, &event); err != nil {
			log.Println("WS invalid JSON:", err)
			continue
		}

		c.handleEvent(event)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(50 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ================= EVENTS =================

func (c *Client) handleEvent(event SocketEvent) {
	switch event.Type {

	case "joinTicket":
		var ticketIDStr string
		switch v := event.Payload.(type) {
		case string:
			ticketIDStr = v
		case float64:
			ticketIDStr = fmt.Sprintf("%.0f", v)
		default:
			ticketIDStr = fmt.Sprintf("%v", v)
		}
		room := "ticket-" + ticketIDStr
		c.joinRoom(room)

		ack := SocketEvent{Type: "ticketJoined", Payload: room}
		if b, err := json.Marshal(ack); err == nil {
			c.Send <- b
		}

	case "adminLogin":
		c.handleAdminLogin(event.Payload)

	case "customerLogin":
		c.handleCustomerLogin(event.Payload)

	default:
		log.Printf("WS: Received unknown event '%s' from client", event.Type)
	}
}

func (c *Client) handleAdminLogin(payload interface{}) {
	token, ok := payload.(string)
	if !ok || token == "" {
		token = c.Token
	}
	if token == "" {
		return
	}

	verifiedToken, err := c.Hub.adminService.VerifyIDToken(token)
	if err != nil {
		log.Printf("WS: Admin token verification failed: %v", err)
		return
	}

	profile, err := c.Hub.adminService.GetProfileByUID(verifiedToken.UID)
	if err != nil {
		log.Printf("WS: Admin profile not found for UID %s", verifiedToken.UID)
		return
	}

	_, _ = c.Hub.adminService.StartWorkSession(profile.ID)
	_ = c.Hub.adminService.SetOnlineStatus(profile.ID, true)

	c.IsAdmin = true
	c.AdminID = profile.ID

	adminRoom := fmt.Sprintf("admin_%d", profile.ID)
	c.joinRoom(adminRoom)

	log.Printf("WS: Admin %d logged in", profile.ID)
	c.Hub.Broadcast("adminStatusChanged", map[string]interface{}{
		"adminId":  profile.ID,
		"isOnline": true,
	})
}

func (c *Client) handleCustomerLogin(payload interface{}) {
	token, ok := payload.(string)
	if !ok || token == "" {
		return
	}

	verifiedToken, err := c.Hub.adminService.VerifyIDToken(token)
	if err != nil {
		log.Printf("WS: Customer token verification failed: %v", err)
		return
	}

	profile, err := c.Hub.customerService.GetCustomerByUID(verifiedToken.UID)
	if err != nil {
		log.Printf("WS: Customer profile not found for UID %s", verifiedToken.UID)
		return
	}

	c.IsAdmin = false
	c.UserID = profile.ID

	customerRoom := fmt.Sprintf("customer_%d", profile.ID)
	c.joinRoom(customerRoom)
	log.Printf("WS: Customer %d logged in", profile.ID)
}

// ================= ROOM =================

func (c *Client) joinRoom(room string) {
	c.Hub.mu.Lock()
	defer c.Hub.mu.Unlock()

	if _, ok := c.Hub.rooms[room]; !ok {
		c.Hub.rooms[room] = make(map[*Client]bool)
	}

	c.Hub.rooms[room][c] = true
	c.Rooms[room] = true

	log.Println("WS: Joined room:", room)
}

// package service

// import (
// 	"encoding/json"
// 	"fmt"
// 	"log"
// 	"net/http"
// 	"sync"
// 	"time"

// 	"github.com/gorilla/websocket"
// )

// // SocketService defines the interface for our websocket service
// type SocketService interface {
// 	Emit(room, event string, args ...interface{})
// 	Broadcast(event string, payload interface{})
// 	HandleConnections(w http.ResponseWriter, r *http.Request)
// }

// // Client represents a single websocket connection
// type Client struct {
// 	Hub     *socketService
// 	Conn    *websocket.Conn
// 	Send    chan []byte
// 	Rooms   map[string]bool // Set of rooms this client is in
// 	IsAdmin bool
// 	AdminID uint64
// 	UserID  uint
// 	Token   string
// }

// // SocketEvent matches the Socket.IO event structure for compatibility with our frontend expectations
// type SocketEvent struct {
// 	Type    string      `json:"type"` // "joinTicket", "newMessage", "ticketJoined"
// 	Payload interface{} `json:"payload"`
// 	Room    string      `json:"room,omitempty"`
// }

// var upgrader = websocket.Upgrader{
// 	CheckOrigin: func(r *http.Request) bool {
// 		return true // Allow all CORS
// 	},
// }

// type socketService struct {
// 	adminService    AdminService
// 	customerService CustomerService

// 	// Registered clients
// 	clients map[*Client]bool

// 	// Inbound messages from the clients
// 	broadcastChan chan []byte

// 	// Register requests from the clients
// 	register chan *Client

// 	// Unregister requests from clients
// 	unregister chan *Client

// 	// Room management: Room Name -> Set of Clients
// 	rooms map[string]map[*Client]bool

// 	// Mutex for safe map access
// 	mu sync.RWMutex
// }

// func NewSocketService(adminService AdminService, customerService CustomerService) SocketService {
// 	s := &socketService{
// 		adminService:    adminService,
// 		customerService: customerService,
// 		clients:         make(map[*Client]bool),
// 		broadcastChan:   make(chan []byte),
// 		register:        make(chan *Client),
// 		unregister:      make(chan *Client),
// 		rooms:           make(map[string]map[*Client]bool),
// 	}
// 	go s.run()
// 	return s
// }

// func (s *socketService) run() {
// 	for {
// 		select {
// 		case client := <-s.register:
// 			s.mu.Lock()
// 			s.clients[client] = true
// 			s.mu.Unlock()
// 			log.Println("WS: New Client Registered")

// 		case client := <-s.unregister:
// 			s.mu.Lock()
// 			if _, ok := s.clients[client]; ok {
// 				s.removeClient(client)
// 			}
// 			s.mu.Unlock()
// 			log.Println("WS: Client Unregistered")

// 		case message := <-s.broadcastChan:
// 			// Broadcast to all (if needed, but usually we use rooms)
// 			// For this implementation, Emit handles room targeting directly.
// 			// This channel might be unused if we only use Emit.
// 			_ = message
// 		}
// 	}
// }

// func (s *socketService) removeClient(client *Client) {
// 	delete(s.clients, client)
// 	close(client.Send)
// 	client.Conn.Close()

// 	// End Work Session if Admin
// 	if client.IsAdmin && client.AdminID != 0 {
// 		// Run in goroutine to not block? Or sync. It's fast db call.
// 		if err := s.adminService.EndWorkSession(client.AdminID); err != nil {
// 			log.Printf("WS: Failed to end work session for Admin %d: %v", client.AdminID, err)
// 		} else {
// 			log.Printf("WS: Ended work session for Admin %d", client.AdminID)
// 		}

// 		// Update Online Status
// 		if err := s.adminService.SetOnlineStatus(client.AdminID, false); err != nil {
// 			log.Printf("WS: Failed to set offline status for Admin %d: %v", client.AdminID, err)
// 		} else {
// 			log.Printf("WS: Admin %d is now offline", client.AdminID)
// 			s.Broadcast("adminStatusChanged", map[string]interface{}{
// 				"adminId":  client.AdminID,
// 				"isOnline": false,
// 			})
// 		}
// 	}

// 	// Remove from all rooms
// 	for room := range client.Rooms {
// 		if clientsInRoom, ok := s.rooms[room]; ok {
// 			delete(clientsInRoom, client)
// 			if len(clientsInRoom) == 0 {
// 				delete(s.rooms, room)
// 			}
// 		}
// 	}
// }

// func (s *socketService) HandleConnections(w http.ResponseWriter, r *http.Request) {
// 	ws, err := upgrader.Upgrade(w, r, nil)
// 	if err != nil {
// 		log.Fatal(err)
// 		return
// 	}

// 	token, _ := r.Cookie("adminToken")
// 	var tokenValue string
// 	if token != nil {
// 		tokenValue = token.Value
// 	}

// 	client := &Client{
// 		Hub:   s,
// 		Conn:  ws,
// 		Send:  make(chan []byte, 256),
// 		Rooms: make(map[string]bool),
// 		Token: tokenValue,
// 	}

// 	s.register <- client

// 	// Start goroutines for read/write
// 	go client.writePump()
// 	go client.readPump()
// }

// func (s *socketService) Emit(room, event string, data ...interface{}) {
// 	s.mu.RLock()
// 	defer s.mu.RUnlock()

// 	targets, ok := s.rooms[room]
// 	if !ok || len(targets) == 0 {
// 		// No one in room
// 		return
// 	}

// 	// Construct message
// 	payload := data[0]
// 	if len(data) > 1 {
// 		// If multiple args, wrap them or pick first?
// 		// Usually we just send one object payload
// 	}

// 	msg := SocketEvent{
// 		Type:    event,
// 		Payload: payload,
// 		Room:    room,
// 	}

// 	bytes, err := json.Marshal(msg)
// 	if err != nil {
// 		log.Println("WS Error: Marshal:", err)
// 		return
// 	}

// 	for client := range targets {
// 		select {
// 		case client.Send <- bytes:
// 		default:
// 			// If buffer is full, the client is likely unresponsive or disconnected.
// 			// We should unregister it to prevent panics on closed channel later.
// 			log.Printf("WS: Removing unresponsive client from room %s", room)
// 			s.removeClient(client)
// 		}
// 	}
// }

// // Client readPump pumps messages from the websocket connection to the hub.
// func (c *Client) readPump() {
// 	defer func() {
// 		c.Hub.unregister <- c
// 		c.Conn.Close()
// 	}()
// 	c.Conn.SetReadLimit(512000) // 512KB
// 	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
// 	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second)); return nil })

// 	for {
// 		_, message, err := c.Conn.ReadMessage()
// 		if err != nil {
// 			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
// 				log.Printf("WS: error: %v", err)
// 			}
// 			break
// 		}

// 		// Handle incoming JSON messages
// 		var event SocketEvent
// 		if err := json.Unmarshal(message, &event); err != nil {
// 			log.Println("WS: Invalid JSON:", err)
// 			continue
// 		}

// 		c.handleEvent(event)
// 	}
// }

// // Client writePump pumps messages from the hub to the websocket connection.
// func (c *Client) writePump() {
// 	ticker := time.NewTicker(54 * time.Second)
// 	defer func() {
// 		ticker.Stop()
// 		c.Conn.Close()
// 	}()

// 	for {
// 		select {
// 		case message, ok := <-c.Send:
// 			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
// 			if !ok {
// 				// The hub closed the channel.
// 				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
// 				return
// 			}

// 			w, err := c.Conn.NextWriter(websocket.TextMessage)
// 			if err != nil {
// 				return
// 			}
// 			w.Write(message)

// 			if err := w.Close(); err != nil {
// 				return
// 			}
// 		case <-ticker.C:
// 			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
// 			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
// 				return
// 			}
// 		}
// 	}
// }

// func (c *Client) handleEvent(event SocketEvent) {
// 	switch event.Type {
// 	case "joinTicket":
// 		// Payload should be ticket ID
// 		var ticketIDStr string
// 		switch v := event.Payload.(type) {
// 		case string:
// 			ticketIDStr = v
// 		case float64:
// 			ticketIDStr = fmt.Sprintf("%d", int(v))
// 		default:
// 			ticketIDStr = fmt.Sprintf("%v", v)
// 		}

// 		roomName := fmt.Sprintf("ticket-%s", ticketIDStr)
// 		c.joinRoom(roomName)

// 		// Acknowledge
// 		ack := SocketEvent{
// 			Type:    "ticketJoined",
// 			Payload: roomName,
// 		}
// 		if b, err := json.Marshal(ack); err == nil {
// 			c.Send <- b
// 		}

// 	case "forceJoinTicket":
// 		room, ok := event.Payload.(string)
// 		if ok {
// 			c.joinRoom(room)
// 			log.Println("WS: forced join room:", room)
// 		}

// 	case "adminLogin":
// 		// Payload: token string
// 		token, ok := event.Payload.(string)
// 		if !ok || token == "" {
// 			token = c.Token
// 		}

// 		if token == "" {
// 			log.Println("WS: No admin token provided in payload or cookie")
// 			return
// 		}

// 		// Verify Token
// 		verifiedToken, err := c.Hub.adminService.VerifyIDToken(token)
// 		if err != nil {
// 			log.Printf("WS: Failed to verify admin token: %v", err)
// 			return
// 		}

// 		// Get Admin Profile to get ID
// 		profile, err := c.Hub.adminService.GetProfileByUID(verifiedToken.UID)
// 		if err != nil {
// 			log.Printf("WS: Failed to find admin profile: %v", err)
// 			return
// 		}

// 		// Start Work Session
// 		if _, err := c.Hub.adminService.StartWorkSession(profile.ID); err != nil {
// 			log.Printf("WS: Failed to start work session: %v", err)
// 		} else {
// 			log.Printf("WS: Started work session for Admin %d", profile.ID)
// 		}

// 		// Update Online Status
// 		if err := c.Hub.adminService.SetOnlineStatus(profile.ID, true); err != nil {
// 			log.Printf("WS: Failed to set online status for Admin %d: %v", profile.ID, err)
// 		} else {
// 			log.Printf("WS: Admin %d is now online", profile.ID)
// 			c.Hub.Broadcast("adminStatusChanged", map[string]interface{}{
// 				"adminId":  profile.ID,
// 				"isOnline": true,
// 			})
// 		}

// 		// Mark client as admin
// 		c.IsAdmin = true
// 		c.AdminID = profile.ID

// 		// Join Admin specific room
// 		roomName := fmt.Sprintf("admin_%d", profile.ID)
// 		c.joinRoom(roomName)

// 	case "customerLogin":
// 		// Payload: token string
// 		token, ok := event.Payload.(string)
// 		if !ok || token == "" {
// 			log.Println("WS: No customer token provided in payload")
// 			return
// 		}

// 		// Verify Token (using adminService's VerifyIDToken as it's the same firebase auth)
// 		verifiedToken, err := c.Hub.adminService.VerifyIDToken(token)
// 		if err != nil {
// 			log.Printf("WS: Failed to verify customer token: %v", err)
// 			return
// 		}

// 		// Get Customer Profile to get ID
// 		profile, err := c.Hub.customerService.GetCustomerByUID(verifiedToken.UID)
// 		if err != nil {
// 			log.Printf("WS: Failed to find customer profile: %v", err)
// 			return
// 		}

// 		// Mark client as customer
// 		c.IsAdmin = false
// 		c.UserID = profile.ID

// 		// Join personal room
// 		customerRoom := fmt.Sprintf("customer_%d", profile.ID)
// 		c.joinRoom(customerRoom)
// 		log.Printf("WS: Customer %d logged in and joined room %s", profile.ID, customerRoom)

// 	case "join":
// 		// Generic join room (e.g. for admin_ID fallback)
// 		room, ok := event.Payload.(string)
// 		if ok && room != "" {
// 			c.joinRoom(room)
// 		}

// 	default:
// 		log.Println("WS: Unknown event type:", event.Type)
// 	}
// }

// func (c *Client) joinRoom(room string) {
// 	c.Hub.mu.Lock()
// 	defer c.Hub.mu.Unlock()

// 	if _, ok := c.Hub.rooms[room]; !ok {
// 		c.Hub.rooms[room] = make(map[*Client]bool)
// 	}
// 	c.Hub.rooms[room][c] = true
// 	c.Rooms[room] = true
// 	log.Printf("WS: Client joined room: %s", room)
// }

// func (s *socketService) Broadcast(event string, payload interface{}) {
// 	s.mu.RLock()
// 	defer s.mu.RUnlock()

// 	msg := SocketEvent{
// 		Type:    event,
// 		Payload: payload,
// 	}

// 	bytes, err := json.Marshal(msg)
// 	if err != nil {
// 		log.Println("WS Error: Marshal:", err)
// 		return
// 	}

// 	for client := range s.clients {
// 		select {
// 		case client.Send <- bytes:
// 		default:
// 			// Client channel might be full or closed
// 		}
// 	}
// }
