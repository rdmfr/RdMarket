package middleware

import (
	"fmt"
	"rdmarket-intelligence/backend/internal/config"
	"rdmarket-intelligence/backend/internal/errors"
	"rdmarket-intelligence/backend/internal/models"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

// SetupMiddleware registers the standard middleware pipeline
func SetupMiddleware(app *fiber.App, cfg *config.Config) {
	// 1. Panic recovery
	app.Use(recover.New(recover.Config{
		EnableStackTrace: cfg.AppEnv != "production",
		StackTraceHandler: func(c *fiber.Ctx, e interface{}) {
			// Structured panic log
			reqID := c.Locals("requestid")
			fmt.Printf(`{"level":"error","time":"%s","request_id":"%v","panic":"%v"}`+"\n",
				time.Now().UTC().Format(time.RFC3339), reqID, e)
		},
	}))

	// 2. Request ID
	app.Use(requestid.New(requestid.Config{
		Header:     "X-Request-ID",
		ContextKey: "requestid",
	}))

	// 3. Security headers & Request body size limit
	app.Use(func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderXContentTypeOptions, "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set(fiber.HeaderReferrerPolicy, "strict-origin-when-cross-origin")
		c.Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none';")
		if cfg.AppEnv == "production" {
			c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Enforce max request body size (2MB)
		if len(c.Body()) > 2*1024*1024 {
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(models.ApiResponse[any]{
				Error: &models.ApiError{
					Code:    string(errors.ErrBadRequest),
					Message: "Request body exceeds size limit (2MB)",
				},
			})
		}

		return c.Next()
	})

	// 4. Structured JSON Logging
	app.Use(func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		latency := time.Since(start)

		reqID := c.Locals("requestid")
		status := c.Response().StatusCode()
		method := c.Method()
		path := c.Path()
		ip := c.IP()

		simulated := c.Locals("simulated") == true || cfg.Provider == "mock"

		logJSON := fmt.Sprintf(
			`{"time":"%s","level":"info","request_id":"%v","method":"%s","path":"%s","status":%d,"latency_ms":%.2f,"ip":"%s","simulated":%t}`+"\n",
			time.Now().UTC().Format(time.RFC3339),
			reqID, method, path, status, float64(latency.Microseconds())/1000.0, ip, simulated,
		)
		fmt.Print(logJSON)

		return err
	})

	// 5. CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSAllowedOrigins,
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS,HEAD",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Request-ID, X-CSRF-Token",
		AllowCredentials: true,
	}))

	// 6. Rate Limiting for auth endpoints
	authLimiter := limiter.New(limiter.Config{
		Max:        10,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(models.ApiResponse[any]{
				Error: &models.ApiError{
					Code:    string(errors.ErrRateLimited),
					Message: "Too many authentication attempts. Please try again later.",
				},
			})
		},
	})
	app.Use("/api/v1/auth/login", authLimiter)
}
