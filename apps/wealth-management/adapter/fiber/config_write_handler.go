package fiber

import (
	"apps/wealth-management/domain"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// Writes for inputs that were sheet-only before AI form filling (D46): fund
// trades, crypto holdings, cards, cashback programmes and merchant MCCs.

func (h *DatabaseHandler) RecordFundTrade(c *fiber.Ctx) error {
	var body domain.FundTradeInput
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	result, err := h.databaseService.RecordFundTrade(body)
	if err != nil {
		return h.fail(c, err)
	}
	return c.Status(http.StatusCreated).JSON(result)
}

func (h *DatabaseHandler) SetCryptoHoldings(c *fiber.Ctx) error {
	var body domain.CryptoHoldingsInput
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	results, err := h.databaseService.SetCryptoHoldings(body)
	if err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{"holdings": results})
}

func (h *DatabaseHandler) CreateCard(c *fiber.Ctx) error {
	var body domain.CardInput
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	card, err := h.databaseService.CreateCard(body)
	if err != nil {
		return h.fail(c, err)
	}
	return c.Status(http.StatusCreated).JSON(card)
}

func (h *DatabaseHandler) UpdateCard(c *fiber.Ctx) error {
	var body domain.CardInput
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	return h.respond(c)(h.databaseService.UpdateCard(pathParam(c, "name"), body))
}

func (h *DatabaseHandler) CreateCardProgram(c *fiber.Ctx) error {
	var body domain.ProgramInput
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	program, err := h.databaseService.CreateCardProgram(body)
	if err != nil {
		return h.fail(c, err)
	}
	return c.Status(http.StatusCreated).JSON(program)
}

func (h *DatabaseHandler) SaveMerchant(c *fiber.Ctx) error {
	var body domain.MerchantInput
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	return h.respond(c)(h.databaseService.SaveMerchant(body))
}
