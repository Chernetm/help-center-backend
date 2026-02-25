package main

import (
	"customer-help-center-backend/internal/config"
	"customer-help-center-backend/internal/database"
	"customer-help-center-backend/internal/models"
	"fmt"
	"log"

	"gorm.io/gorm"
)

func main() {
	// 1. Load Config
	cfg := config.LoadConfig()

	// 2. Initialize Database Connection
	database.ConnectDB(cfg.GetDSN())

	fmt.Println("🔥 BULK DELETE: Starting deletion of all tickets and associated data...")

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Delete in order to respect foreign keys
		tables := []interface{}{
			&models.MessageReadStatus{},
			&models.ChatMessage{},
			&models.Rating{},
			&models.TicketMetrics{},
			&models.Ticket{},
		}

		for _, table := range tables {
			if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(table).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		log.Fatalf("🔥 BULK DELETE ERROR: %v", err)
	}

	fmt.Println("✅ BULK DELETE: Successfully cleared all ticket data.")
}
