package main

import (
	"customer-help-center-backend/internal/config"
	"customer-help-center-backend/internal/database"
	"customer-help-center-backend/internal/models"
	"fmt"
)

func main() {
	cfg := config.LoadConfig()
	database.ConnectDB(cfg.GetDSN())

	var columns []string
	database.DB.Raw("SELECT column_name FROM information_schema.columns WHERE table_name = 'orders'").Scan(&columns)

	fmt.Println("Columns in 'orders' table:")
	for _, col := range columns {
		fmt.Println("-", col)
	}

	var orders []models.Order
	err := database.DB.Find(&orders).Error
	if err != nil {
		fmt.Printf("Error finding orders: %v\n", err)
	} else {
		fmt.Printf("Successfully found %d orders\n", len(orders))
	}
}
