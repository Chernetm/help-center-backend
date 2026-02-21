package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort              string
	FirebaseCredentialsPath string
	FirebaseCredentialsJSON string
	FirebaseAPIKey          string

	DBUser    string
	DBPass    string
	DBHost    string
	DBName    string
	DBPort    string
	DBSSLMode string

	DatabaseURL string // Allow full connection string override
}

func LoadConfig() *Config {
	// Load .env file if exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment")
	}

	cfg := &Config{
		ServerPort:              os.Getenv("PORT"),
		FirebaseCredentialsJSON: os.Getenv("FIREBASE_CREDENTIALS_JSON"),
		FirebaseAPIKey:          os.Getenv("FIREBASE_API_KEY"),

		DBUser:      os.Getenv("DB_USER"),
		DBPass:      os.Getenv("DB_PASS"),
		DBHost:      os.Getenv("DB_HOST"),
		DBName:      os.Getenv("DB_NAME"),
		DBPort:      os.Getenv("DB_PORT"),
		DBSSLMode:   os.Getenv("DB_SSLMODE"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}

	if cfg.ServerPort == "" {
		cfg.ServerPort = "8080"
	}

	if cfg.FirebaseCredentialsJSON == "" {
		log.Fatal("FIREBASE_CREDENTIALS_JSON is set")
	}

	if cfg.FirebaseAPIKey == "" {
		log.Fatal("FIREBASE_API_KEY is not set")
	}

	// If DatabaseURL is not provided, we need the individual DB vars
	if cfg.DatabaseURL == "" {
		if cfg.DBUser == "" || cfg.DBPass == "" || cfg.DBHost == "" || cfg.DBName == "" {
			log.Fatal("Database environment variables or DATABASE_URL are not set")
		}
	}

	return cfg
}

// ⭐ DSN builder for PostgreSQL
func (c *Config) GetDSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}

	sslMode := c.DBSSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	port := c.DBPort
	if port == "" {
		port = "5432"
	}

	// PostgreSQL DSN format: host=localhost user=gorm password=gorm dbname=gorm port=9920 sslmode=disable TimeZone=Asia/Shanghai
	return "host=" + c.DBHost +
		" user=" + c.DBUser +
		" password=" + c.DBPass +
		" dbname=" + c.DBName +

		" sslmode=" + sslMode
}
