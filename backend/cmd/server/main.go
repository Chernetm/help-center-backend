package main

import (
	"log"
	"time"

	"customer-help-center-backend/internal/config"
	"customer-help-center-backend/internal/database"
	"customer-help-center-backend/internal/handlers"
	"customer-help-center-backend/internal/models"
	"customer-help-center-backend/internal/repository"
	"customer-help-center-backend/internal/routes"
	"customer-help-center-backend/internal/service"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Load Config
	cfg := config.LoadConfig()

	// 2. Initialize Database Connection
	database.ConnectDB(cfg.GetDSN())

	firebaseService, err := service.NewFirebaseService(cfg)
	if err != nil {
		log.Fatal("Failed to init firebase service: ", err)
	}

	// 3. Auto Migrate
	err = database.DB.AutoMigrate(
		&models.Admin{},
		&models.Customer{},
		&models.Case{},
		&models.Ticket{},
		&models.TicketMetrics{},
		&models.Order{},
		&models.ChatMessage{},
		&models.MessageReadStatus{},
		&models.WorkSession{},
		&models.AuditLog{},
		&models.Rating{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database: ", err)
	}

	// 4. Setup Dependencies
	adminRepo := repository.NewAdminRepository(database.DB)
	customerRepo := repository.NewCustomerRepository(database.DB)
	ticketRepo := repository.NewTicketRepository(database.DB)
	caseRepo := repository.NewCaseRepository(database.DB)
	orderRepo := repository.NewOrderRepository(database.DB)
	chatRepo := repository.NewChatRepository(database.DB)
	auditRepo := repository.NewAuditRepository(database.DB)
	ratingRepo := repository.NewRatingRepository(database.DB)
	sessionRepo := repository.NewWorkSessionRepository(database.DB)

	adminService := service.NewAdminService(adminRepo, sessionRepo, firebaseService, cfg)
	customerService := service.NewCustomerService(customerRepo, firebaseService, cfg)
	socketService := service.NewSocketService(adminService, customerService)
	ratingService := service.NewRatingService(ratingRepo, ticketRepo)
	ticketService := service.NewTicketService(ticketRepo, caseRepo, adminRepo, ratingRepo, chatRepo, socketService)
	caseService := service.NewCaseService(caseRepo)
	orderService := service.NewOrderService(orderRepo)
	chatService := service.NewChatService(chatRepo, ticketRepo, socketService)
	commonService := service.NewCommonService(auditRepo, ratingRepo, sessionRepo)

	// ✅ UNCOMMENT TO DELETE ALL TICKETS, CHATS, RATINGS, ETC.
	// log.Println("🔥 BULK DELETE: Starting deletion of all tickets and associated data...")
	// if err := ticketService.DeleteAllTickets(); err != nil {
	// 	log.Printf("🔥 BULK DELETE ERROR: %v", err)
	// } else {
	// 	log.Println("🔥 BULK DELETE: Successfully cleared all ticket data.")
	// }

	// Handlers
	adminHandler := handlers.NewAdminHandler(adminService)
	customerHandler := handlers.NewCustomerHandler(customerService)
	ticketHandler := handlers.NewTicketHandler(ticketService, ratingService)
	caseHandler := handlers.NewCaseHandler(caseService)
	orderHandler := handlers.NewOrderHandler(orderService)
	chatHandler := handlers.NewChatHandler(chatService)
	commonHandler := handlers.NewCommonHandler(commonService)
	externalHandler := handlers.NewExternalHandler()

	// 5. Setup Router
	r := gin.Default()

	// ⭐ GLOBAL CORS MIDDLEWARE ⭐
	//https://selamcustomersupport.vercel.app
	//http://localhost:5173
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "https://selamcustomersupport.vercel.app") // Adjust to your frontend URL
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// WebSocket
	r.GET("/ws", func(c *gin.Context) {
		socketService.HandleConnections(c.Writer, c.Request)
	})

	// Routes
	routes.SetupRoutes(
		r,
		ticketHandler,
		caseHandler,
		adminHandler,
		customerHandler,
		orderHandler,
		chatHandler,
		commonHandler,
		externalHandler,
		adminService,
		customerService,
	)

	// 6. Start Background Workers
	go func() {
		log.Println("AutoCleanup Worker: Started")
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		// Run once at startup
		if err := ticketService.CleanUnassignedTickets(); err != nil {
			log.Printf("AutoCleanup Error: %v", err)
		}

		for range ticker.C {
			if err := ticketService.CleanUnassignedTickets(); err != nil {
				log.Printf("AutoCleanup Error: %v", err)
			}
		}
	}()

	log.Printf("Server starting on :%s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
