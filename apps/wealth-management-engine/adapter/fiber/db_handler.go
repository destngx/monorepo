package fiber

import (
	configAdapter "apps/wealth-management-engine/adapter/config"
	"apps/wealth-management-engine/port"
	"errors"

	"github.com/gofiber/fiber/v2"
)

type DatabaseHandler struct {
	databaseService port.DatabaseService
}

func NewDatabaseHandler(service port.DatabaseService) *DatabaseHandler {
	return &DatabaseHandler{databaseService: service}
}

func (h *DatabaseHandler) GetAccounts(c *fiber.Ctx) error {
	accounts, err := h.databaseService.ReadAccounts()
	if err != nil {
		statusCode := fiber.StatusInternalServerError
		if errors.Is(err, configAdapter.ErrMissingSheetsConfig) {
			statusCode = fiber.StatusServiceUnavailable
		}

		return c.Status(statusCode).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(accounts)
}
