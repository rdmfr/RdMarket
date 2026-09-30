package economic

import (
	"errors"
	"rdmarket-intelligence/backend/internal/middleware"
	"rdmarket-intelligence/backend/internal/models"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type HTTPHandler struct {
	service APIService
}

func NewHTTPHandler(service APIService) *HTTPHandler {
	return &HTTPHandler{service: service}
}

func (h *HTTPHandler) RegisterRoutes(router fiber.Router, auth *middleware.AuthHandler) {
	economic := router.Group("/economic")
	economic.Get("/indicators", auth.RequireAuth(), h.ListIndicators)
	economic.Get("/indicators/:code", auth.RequireAuth(), h.GetIndicator)
	economic.Get("/indicators/:code/history", auth.RequireAuth(), h.GetHistory)
}

func (h *HTTPHandler) ListIndicators(c *fiber.Ctx) error {
	data, err := h.service.ListIndicators(c.UserContext())
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(models.NewErrorResponse("SERVICE_UNAVAILABLE", "Economic indicator data is temporarily unavailable", nil))
	}
	return c.Status(fiber.StatusOK).JSON(models.NewSuccessResponse(data, fiber.Map{"count": len(data), "generated_at": time.Now().UTC()}))
}

func (h *HTTPHandler) GetIndicator(c *fiber.Ctx) error {
	data, err := h.service.GetIndicator(c.UserContext(), c.Params("code"))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(models.NewErrorResponse("NOT_FOUND", "Economic indicator was not found", nil))
	}
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(models.NewErrorResponse("SERVICE_UNAVAILABLE", "Economic indicator data is temporarily unavailable", nil))
	}
	return c.Status(fiber.StatusOK).JSON(models.NewSuccessResponse(data, fiber.Map{"generated_at": time.Now().UTC()}))
}

func (h *HTTPHandler) GetHistory(c *fiber.Ctx) error {
	start, end, err := parseHistoryRange(c)
	if err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(models.NewErrorResponse("UNPROCESSABLE_ENTITY", "The requested economic history range is invalid", nil))
	}
	revisions := false
	if value := c.Query("revisions"); value != "" {
		revisions, err = strconv.ParseBool(value)
		if err != nil {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(models.NewErrorResponse("UNPROCESSABLE_ENTITY", "The revisions parameter must be true or false", nil))
		}
	}
	data, err := h.service.GetHistory(c.UserContext(), c.Params("code"), start, end, revisions)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(models.NewErrorResponse("NOT_FOUND", "Economic indicator was not found", nil))
	}
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(models.NewErrorResponse("SERVICE_UNAVAILABLE", "Economic history is temporarily unavailable", nil))
	}
	return c.Status(fiber.StatusOK).JSON(models.NewSuccessResponse(data, fiber.Map{"count": len(data), "revisions": revisions, "generated_at": time.Now().UTC()}))
}

func parseHistoryRange(c *fiber.Ctx) (time.Time, time.Time, error) {
	startValue, endValue := c.Query("start"), c.Query("end")
	if startValue != "" || endValue != "" {
		if startValue == "" || endValue == "" {
			return time.Time{}, time.Time{}, errors.New("both start and end are required")
		}
		start, err := time.Parse("2006-01-02", startValue)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		end, err := time.Parse("2006-01-02", endValue)
		if err != nil || end.Before(start) {
			return time.Time{}, time.Time{}, errors.New("invalid date interval")
		}
		return start.UTC(), end.UTC().Add(24*time.Hour - time.Nanosecond), nil
	}
	end := time.Now().UTC()
	start := end.AddDate(-1, 0, 0)
	switch c.Query("range", "1Y") {
	case "1M":
		start = end.AddDate(0, -1, 0)
	case "3M":
		start = end.AddDate(0, -3, 0)
	case "1Y":
	case "5Y":
		start = end.AddDate(-5, 0, 0)
	default:
		return time.Time{}, time.Time{}, errors.New("unsupported range")
	}
	return start, end, nil
}
