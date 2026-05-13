package main

import (
	"crane-system/app"
	"crane-system/database"
	"log"
)

func InitAll() {
	log.Println("initializing database schema and base data...")

	cfg, err := app.SetupInfrastructure()
	if err != nil {
		log.Fatal("setup infrastructure failed:", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			log.Printf("close database failed: %v", err)
		}
	}()

	if err := database.AutoMigrate(); err != nil {
		log.Fatal("database initialization failed:", err)
	}

	log.Println("database schema and base data initialized")
	log.Println("default admin employee id: admin001")
	log.Printf("default admin password: %s", cfg.Auth.DefaultPassword)
}
