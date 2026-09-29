package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"rdmarket-intelligence/backend/internal/config"
	"rdmarket-intelligence/backend/internal/errors"
	"rdmarket-intelligence/backend/internal/models"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

const (
	SessionCookieName = "rdm_session"
	CSRFCookieName    = "rdm_csrf"
	CSRFHeaderName    = "X-CSRF-Token"
)

type AuthHandler struct {
	cfg *config.Config
}

func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{cfg: cfg}
}

func (h *AuthHandler) generateToken(username string) string {
	ts := time.Now().Unix()
	payload := username + ":" + string(rune(ts))
	mac := hmac.New(sha256.New, []byte(h.cfg.SessionSecret))
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))
	return payload + "." + sig
}

func (h *AuthHandler) validateToken(token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return false
	}
	payload, sig := parts[0], parts[1]
	mac := hmac.New(sha256.New, []byte(h.cfg.SessionSecret))
	mac.Write([]byte(payload))
	expectedSig := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sig), []byte(expectedSig))
}

func (h *AuthHandler) generateCSRFToken() string {
	mac := hmac.New(sha256.New, []byte(h.cfg.SessionSecret))
	mac.Write([]byte("csrf-token-salt"))
	return hex.EncodeToString(mac.Sum(nil))
}

// RequireAuth middleware protects state-changing routes and optional read routes
func (h *AuthHandler) RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Read-only endpoints can be public if AUTH_REQUIRE_READ is false
		if !h.cfg.AuthRequireRead && (c.Method() == fiber.MethodGet || c.Method() == fiber.MethodHead || c.Method() == fiber.MethodOptions) {
			return c.Next()
		}

		cookie := c.Cookies(SessionCookieName)
		if cookie == "" || !h.validateToken(cookie) {
			return c.Status(fiber.StatusUnauthorized).JSON(models.ApiResponse[any]{
				Error: &models.ApiError{
					Code:    string(errors.ErrUnauthorized),
					Message: "Authentication required",
				},
			})
		}

		// CSRF verification on mutating state-changing requests (POST, PUT, DELETE, PATCH)
		if c.Method() == fiber.MethodPost || c.Method() == fiber.MethodPut || c.Method() == fiber.MethodDelete || c.Method() == fiber.MethodPatch {
			csrfHeader := c.Get(CSRFHeaderName)
			expectedCSRF := h.generateCSRFToken()
			if csrfHeader == "" || csrfHeader != expectedCSRF {
				return c.Status(fiber.StatusForbidden).JSON(models.ApiResponse[any]{
					Error: &models.ApiError{
						Code:    string(errors.ErrCSRFInvalid),
						Message: "Invalid or missing CSRF token",
					},
				})
			}
		}

		return c.Next()
	}
}

// LoginRequest request body
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterRoutes registers auth endpoints
func (h *AuthHandler) RegisterRoutes(router fiber.Router) {
	auth := router.Group("/auth")

	auth.Post("/login", func(c *fiber.Ctx) error {
		var req LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(models.ApiResponse[any]{
				Error: &models.ApiError{
					Code:    string(errors.ErrBadRequest),
					Message: "Invalid request payload",
				},
			})
		}

		// Verify username
		if req.Username != h.cfg.AdminUsername {
			return c.Status(fiber.StatusUnauthorized).JSON(models.ApiResponse[any]{
				Error: &models.ApiError{
					Code:    string(errors.ErrInvalidCredentials),
					Message: "Invalid credentials",
				},
			})
		}

		// Verify bcrypt password hash
		err := bcrypt.CompareHashAndPassword([]byte(h.cfg.AdminPasswordHash), []byte(req.Password))
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(models.ApiResponse[any]{
				Error: &models.ApiError{
					Code:    string(errors.ErrInvalidCredentials),
					Message: "Invalid credentials",
				},
			})
		}

		// Issue session cookie
		sessionToken := h.generateToken(req.Username)
		csrfToken := h.generateCSRFToken()

		isSecure := h.cfg.AppEnv == "production"
		c.Cookie(&fiber.Cookie{
			Name:     SessionCookieName,
			Value:    sessionToken,
			HTTPOnly: true,
			Secure:   isSecure,
			SameSite: "Lax",
			Path:     "/",
			Expires:  time.Now().Add(24 * time.Hour),
		})

		c.Cookie(&fiber.Cookie{
			Name:     CSRFCookieName,
			Value:    csrfToken,
			HTTPOnly: false,
			Secure:   isSecure,
			SameSite: "Lax",
			Path:     "/",
			Expires:  time.Now().Add(24 * time.Hour),
		})

		return c.JSON(models.ApiResponse[map[string]any]{
			Data: map[string]any{
				"authenticated": true,
				"username":      req.Username,
				"csrf_token":    csrfToken,
			},
			Meta: map[string]any{
				"simulated": h.cfg.Provider == "mock",
			},
		})
	})

	auth.Post("/logout", func(c *fiber.Ctx) error {
		c.Cookie(&fiber.Cookie{
			Name:     SessionCookieName,
			Value:    "",
			HTTPOnly: true,
			Path:     "/",
			Expires:  time.Now().Add(-1 * time.Hour),
		})
		c.Cookie(&fiber.Cookie{
			Name:     CSRFCookieName,
			Value:    "",
			HTTPOnly: false,
			Path:     "/",
			Expires:  time.Now().Add(-1 * time.Hour),
		})
		return c.JSON(models.ApiResponse[map[string]any]{
			Data: map[string]any{
				"authenticated": false,
			},
		})
	})

	auth.Get("/me", func(c *fiber.Ctx) error {
		cookie := c.Cookies(SessionCookieName)
		isAuth := cookie != "" && h.validateToken(cookie)
		username := ""
		if isAuth {
			username = h.cfg.AdminUsername
		}
		return c.JSON(models.ApiResponse[map[string]any]{
			Data: map[string]any{
				"authenticated": isAuth,
				"username":      username,
			},
			Meta: map[string]any{
				"simulated": h.cfg.Provider == "mock",
			},
		})
	})
}
