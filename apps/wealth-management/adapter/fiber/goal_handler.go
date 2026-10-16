package fiber

import (
	"apps/wealth-management/domain"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
)

// Goal Plan writes (D31): setup, create, edit, delete and reorder.

func (h *DatabaseHandler) SetupGoalPlan(c *fiber.Ctx) error {
	result, err := h.databaseService.SetupGoalPlan()
	if err != nil {
		return h.fail(c, err)
	}
	return c.Status(http.StatusCreated).JSON(result)
}

func (h *DatabaseHandler) CreateGoal(c *fiber.Ctx) error {
	var body domain.GoalInput
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	goal, err := h.databaseService.CreateGoal(body)
	if err != nil {
		return h.fail(c, err)
	}
	return c.Status(http.StatusCreated).JSON(goal)
}

func (h *DatabaseHandler) UpdateGoal(c *fiber.Ctx) error {
	var body domain.GoalUpdateInput
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	if body.ETag == "" {
		body.ETag = utils.CopyString(c.Get(fiber.HeaderIfMatch))
	}
	body.ETag = strings.Trim(body.ETag, `"`)
	return h.respond(c)(h.databaseService.UpdateGoal(pathParam(c, "id"), body))
}

func (h *DatabaseHandler) DeleteGoal(c *fiber.Ctx) error {
	etag := c.Get(fiber.HeaderIfMatch)
	if etag == "" {
		etag = c.Query("etag")
	}
	if err := h.databaseService.DeleteGoal(pathParam(c, "id"), strings.Trim(utils.CopyString(etag), `"`)); err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{"success": true})
}

func (h *DatabaseHandler) ReorderGoals(c *fiber.Ctx) error {
	var body domain.GoalReorderInput
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	return h.respond(c)(h.databaseService.ReorderGoals(body))
}
