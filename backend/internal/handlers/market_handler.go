package handlers

import (
	"errors"
	"rdmarket-intelligence/backend/internal/models"
	"rdmarket-intelligence/backend/internal/services"
	"time"

	"github.com/gofiber/fiber/v2"
)

type MarketHandler struct {
	service services.MarketService
}

func NewMarketHandler(service services.MarketService) *MarketHandler {
	return &MarketHandler{service: service}
}

func (h *MarketHandler) RegisterRoutes(router fiber.Router) {
	v1 := router.Group("/api/v1")

	// Health check
	v1.Get("/health", h.GetHealth)
	v1.Get("/ready", h.GetReady)

	// Market endpoints
	market := v1.Group("/market")
	market.Get("/usdidr/current", h.GetCurrentRate)
	market.Get("/usdidr/history", h.GetHistory)
	market.Get("/usdidr/statistics", h.GetStatistics)
	market.Get("/usdidr/indicators", h.GetIndicators)
	market.Get("/usdidr/condition", h.GetCondition)

	// Data sources endpoint
	v1.Get("/data-sources", h.GetDataSources)
}

func (h *MarketHandler) GetReady(c *fiber.Ctx) error {
	if err := h.service.Ready(); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(models.NewErrorResponse(
			"DATABASE_UNAVAILABLE",
			"Database is not ready",
			nil,
		))
	}
	return c.Status(fiber.StatusOK).JSON(models.NewSuccessResponse(fiber.Map{
		"status":  "ready",
		"service": "rdmarket-backend",
	}, nil))
}

func (h *MarketHandler) GetHealth(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(models.NewSuccessResponse(fiber.Map{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"service":   "rdmarket-backend",
		"phase":     1,
	}, nil))
}

func (h *MarketHandler) GetCurrentRate(c *fiber.Ctx) error {
	data, err := h.service.GetCurrentRate("USD/IDR")
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(models.NewErrorResponse(
			"DATA_SOURCE_UNAVAILABLE",
			"Market data source is temporarily unavailable",
			nil,
		))
	}
	if data == nil {
		return c.Status(fiber.StatusNotFound).JSON(models.NewErrorResponse(
			"NOT_FOUND",
			"No exchange rate data found for USD/IDR",
			nil,
		))
	}
	return c.Status(fiber.StatusOK).JSON(models.NewSuccessResponse(data, fiber.Map{
		"currency_pair": "USD/IDR",
		"generated_at":  time.Now().UTC(),
	}))
}

func (h *MarketHandler) GetHistory(c *fiber.Ctx) error {
	rangeParam := c.Query("range", "1M")
	startParam := c.Query("start", "")
	endParam := c.Query("end", "")

	data, err := h.service.GetHistory("USD/IDR", rangeParam, startParam, endParam)
	if err != nil {
		if errors.Is(err, services.ErrInvalidRange) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(models.NewErrorResponse(
				"INVALID_RANGE", "The requested market data range is invalid", nil,
			))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(models.NewErrorResponse(
			"HISTORY_FETCH_ERROR",
			"Failed to retrieve historical exchange rate data",
			nil,
		))
	}
	return c.Status(fiber.StatusOK).JSON(models.NewSuccessResponse(data, fiber.Map{
		"range":              rangeParam,
		"total_points":       data.TotalPoints,
		"source_point_count": data.SourcePointCount,
		"resolution":         data.Resolution,
		"aggregation_method": data.AggregationMethod,
		"generated_at":       time.Now().UTC(),
	}))
}

func (h *MarketHandler) GetStatistics(c *fiber.Ctx) error {
	stats, err := h.service.GetStatistics("USD/IDR")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.NewErrorResponse(
			"STATISTICS_ERROR",
			"Failed to compute market statistics",
			nil,
		))
	}
	return c.Status(fiber.StatusOK).JSON(models.NewSuccessResponse(stats, fiber.Map{
		"currency_pair": "USD/IDR",
		"generated_at":  time.Now().UTC(),
	}))
}

func (h *MarketHandler) GetIndicators(c *fiber.Ctx) error {
	rangeParam := c.Query("range", "1M")
	data, err := h.service.GetIndicators("USD/IDR", rangeParam)
	if err != nil {
		if errors.Is(err, services.ErrInvalidRange) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(models.NewErrorResponse(
				"INVALID_RANGE", "The requested market data range is invalid", nil,
			))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(models.NewErrorResponse(
			"INDICATORS_ERROR",
			"Failed to compute statistical indicators",
			nil,
		))
	}
	return c.Status(fiber.StatusOK).JSON(models.NewSuccessResponse(data, fiber.Map{
		"range":        rangeParam,
		"generated_at": time.Now().UTC(),
	}))
}

func (h *MarketHandler) GetCondition(c *fiber.Ctx) error {
	rangeParam := c.Query("range", "1M")
	data, err := h.service.GetCondition("USD/IDR", rangeParam)
	if err != nil {
		if errors.Is(err, services.ErrInvalidRange) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(models.NewErrorResponse(
				"INVALID_RANGE", "The requested market data range is invalid", nil,
			))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(models.NewErrorResponse(
			"CONDITION_ERROR", "Failed to compute market condition", nil,
		))
	}
	return c.Status(fiber.StatusOK).JSON(models.NewSuccessResponse(data, fiber.Map{
		"range": rangeParam, "generated_at": time.Now().UTC(),
	}))
}

func (h *MarketHandler) GetDataSources(c *fiber.Ctx) error {
	sources, err := h.service.GetDataSources()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.NewErrorResponse(
			"DATA_SOURCES_ERROR",
			"Failed to retrieve data source status",
			nil,
		))
	}
	return c.Status(fiber.StatusOK).JSON(models.NewSuccessResponse(sources, fiber.Map{
		"count":        len(sources),
		"generated_at": time.Now().UTC(),
	}))
}
