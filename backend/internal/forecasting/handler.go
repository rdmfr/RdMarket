package forecasting

import (
	"errors"
	"rdmarket-intelligence/backend/internal/middleware"
	"rdmarket-intelligence/backend/internal/models"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

type Handler struct {
	service *Service
	auth    *middleware.AuthHandler
}

func NewHandler(service *Service, auth *middleware.AuthHandler) *Handler {
	return &Handler{service: service, auth: auth}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	group := router.Group("/forecast/usdidr")
	group.Get("/models", h.getModels)
	group.Get("/latest", h.getLatest)
	group.Get("/runs/:id", h.getRun)
	group.Get("/jobs", h.getJobs)
	group.Get("/jobs/:id", h.getJob)
	group.Post("/jobs", limiter.New(limiter.Config{
		Max:        10,
		Expiration: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(models.NewErrorResponse(
				"RATE_LIMITED", "Too many forecast job requests. Please try again later.", nil,
			))
		},
	}), h.auth.RequireAuth(), h.createJob)
	group.Get("/backtests", h.getBacktests)
	group.Get("/backtests/:id", h.getBacktest)
	group.Get("/backtests/:id/predictions", h.getPredictions)
	group.Get("/leaderboard", h.getLeaderboard)
}

func (h *Handler) getModels(c *fiber.Ctx) error {
	catalog, err := h.service.Models(c.Context())
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(models.NewSuccessResponse(catalog, fiber.Map{"count": len(catalog)}))
}

func (h *Handler) getLatest(c *fiber.Ctx) error {
	model := c.Query("model", "naive")
	horizon, err := strconv.Atoi(c.Query("horizon", "1"))
	if err != nil {
		return writeError(c, ErrInvalidRequest)
	}
	result, err := h.service.Latest(model, horizon)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(models.NewSuccessResponse(result, fiber.Map{"generated_at": time.Now().UTC()}))
}

func (h *Handler) getRun(c *fiber.Ctx) error {
	result, err := h.service.Run(c.Params("id"))
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(models.NewSuccessResponse(result, fiber.Map{"generated_at": time.Now().UTC()}))
}

func (h *Handler) getJobs(c *fiber.Ctx) error {
	page := parsePage(c.Query("page"), 1, 1000000)
	limit := parsePage(c.Query("limit"), 20, 100)
	jobs, err := h.service.Jobs(limit, (page-1)*limit)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(models.NewSuccessResponse(jobs, fiber.Map{"page": page, "limit": limit}))
}

func (h *Handler) getJob(c *fiber.Ctx) error {
	job, err := h.service.Job(c.Params("id"))
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(models.NewSuccessResponse(job, nil))
}

func (h *Handler) createJob(c *fiber.Ctx) error {
	var request JobRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.NewErrorResponse(
			"BAD_REQUEST", "Invalid forecast job payload", nil,
		))
	}
	job, err := h.service.CreateJob(c.Context(), request, nil)
	if err != nil {
		return writeError(c, err)
	}
	return c.Status(fiber.StatusAccepted).JSON(models.NewSuccessResponse(job, nil))
}

func (h *Handler) getBacktests(c *fiber.Ctx) error {
	page := parsePage(c.Query("page"), 1, 1000000)
	limit := parsePage(c.Query("limit"), 20, 100)
	runs, err := h.service.Backtests(limit, (page-1)*limit)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(models.NewSuccessResponse(runs, fiber.Map{"page": page, "limit": limit}))
}

func (h *Handler) getBacktest(c *fiber.Ctx) error {
	result, err := h.service.Backtest(c.Params("id"))
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(models.NewSuccessResponse(result, nil))
}

func (h *Handler) getPredictions(c *fiber.Ctx) error {
	page := parsePage(c.Query("page"), 1, 1000000)
	limit := parsePage(c.Query("limit"), 100, 500)
	predictions, err := h.service.Predictions(c.Params("id"), limit, (page-1)*limit)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(models.NewSuccessResponse(predictions, fiber.Map{"page": page, "limit": limit}))
}

func (h *Handler) getLeaderboard(c *fiber.Ctx) error {
	horizon, err := strconv.Atoi(c.Query("horizon", "1"))
	if err != nil {
		return writeError(c, ErrInvalidRequest)
	}
	result, err := h.service.Leaderboard(horizon)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(models.NewSuccessResponse(result, fiber.Map{"generated_at": time.Now().UTC()}))
}

func writeError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalidRequest):
		return c.Status(fiber.StatusUnprocessableEntity).JSON(models.NewErrorResponse(
			"INVALID_FORECAST_REQUEST", err.Error(), nil,
		))
	case errors.Is(err, ErrDuplicateJob):
		return c.Status(fiber.StatusConflict).JSON(models.NewErrorResponse(
			"FORECAST_JOB_CONFLICT", err.Error(), nil,
		))
	case errors.Is(err, ErrNotFound):
		return c.Status(fiber.StatusNotFound).JSON(models.NewErrorResponse(
			"NOT_FOUND", "Forecast result was not found", nil,
		))
	case errors.Is(err, ErrServiceUnavailable):
		return c.Status(fiber.StatusServiceUnavailable).JSON(models.NewErrorResponse(
			"FORECAST_SERVICE_UNAVAILABLE", "Forecasting is temporarily unavailable", nil,
		))
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(models.NewErrorResponse(
			"FORECAST_ERROR", "Unable to retrieve forecasting data", nil,
		))
	}
}
