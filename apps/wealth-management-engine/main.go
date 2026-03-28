package main

import (
	fiberAdapter "apps/wealth-management-engine/adapter/fiber"
	"apps/wealth-management-engine/service"
	"github.com/gofiber/fiber/v2"
	"log"
)

func main() {
	app := fiber.New()

	// Initialize services
	healthSvc := service.NewHealthService()

	// Initialize handlers
	healthHandler := fiberAdapter.NewHealthHandler(healthSvc)

	// Routes
	app.Get("/api/health", healthHandler.HealthCheck)

	log.Println("Starting server on :8080")
	log.Fatal(app.Listen(":8080"))
}
