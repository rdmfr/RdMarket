package domain

import (
	"fmt"
	"math"
	"rdmarket-intelligence/backend/internal/models"
	"time"
)

type ValidationConfig struct {
	MinPlausibleRate float64 // e.g. 5000.0 for USD/IDR
	MaxPlausibleRate float64 // e.g. 30000.0 for USD/IDR
	MaxJumpPercent   float64 // e.g. 10.0 (10% single-step jump)
	MaxStalenessDays int     // e.g. 14 days without update
}

var DefaultValidationConfig = ValidationConfig{
	MinPlausibleRate: 5000.0,
	MaxPlausibleRate: 35000.0,
	MaxJumpPercent:   10.0,
	MaxStalenessDays: 14,
}

type ValidationResult struct {
	QualityStatus string  // "ok", "suspect", "rejected"
	QualityReason *string // explanatory reason if not ok
	IsValid       bool    // true if not rejected
}

// ValidateObservation performs pure domain rule-based checks on an exchange rate observation
func ValidateObservation(
	current *models.ExchangeRate,
	previous *models.ExchangeRate,
	now time.Time,
	cfg ValidationConfig,
) ValidationResult {
	if current == nil {
		reason := "nil observation"
		return ValidationResult{QualityStatus: "rejected", QualityReason: &reason, IsValid: false}
	}

	// 1. Sanity bounds: Non-positive or outside plausible limits
	if current.Rate <= 0 {
		reason := fmt.Sprintf("rate %.4f is non-positive", current.Rate)
		return ValidationResult{QualityStatus: "rejected", QualityReason: &reason, IsValid: false}
	}

	if current.Rate < cfg.MinPlausibleRate {
		reason := fmt.Sprintf("rate %.4f below minimum plausible bound (%.2f)", current.Rate, cfg.MinPlausibleRate)
		return ValidationResult{QualityStatus: "suspect", QualityReason: &reason, IsValid: true}
	}

	if current.Rate > cfg.MaxPlausibleRate {
		reason := fmt.Sprintf("rate %.4f above maximum plausible bound (%.2f)", current.Rate, cfg.MaxPlausibleRate)
		return ValidationResult{QualityStatus: "suspect", QualityReason: &reason, IsValid: true}
	}

	// 2. Future timestamp check
	if current.Timestamp.After(now.Add(1 * time.Hour)) {
		reason := fmt.Sprintf("observation timestamp %s is in the future", current.Timestamp.Format(time.RFC3339))
		return ValidationResult{QualityStatus: "rejected", QualityReason: &reason, IsValid: false}
	}

	// 3. Staleness check
	if now.Sub(current.Timestamp) > time.Duration(cfg.MaxStalenessDays)*24*time.Hour {
		reason := fmt.Sprintf("observation is stale (> %d days old)", cfg.MaxStalenessDays)
		return ValidationResult{QualityStatus: "suspect", QualityReason: &reason, IsValid: true}
	}

	// 4. Comparison against previous observation (if available)
	if previous != nil && previous.Rate > 0 {
		// Out of order check
		if current.Timestamp.Before(previous.Timestamp) {
			reason := fmt.Sprintf("observation timestamp %s is before previous observation %s",
				current.Timestamp.Format(time.RFC3339), previous.Timestamp.Format(time.RFC3339))
			return ValidationResult{QualityStatus: "suspect", QualityReason: &reason, IsValid: true}
		}

		// Duplicate timestamp check
		if current.Timestamp.Equal(previous.Timestamp) {
			reason := fmt.Sprintf("duplicate timestamp %s", current.Timestamp.Format(time.RFC3339))
			return ValidationResult{QualityStatus: "suspect", QualityReason: &reason, IsValid: true}
		}

		// Jump detection
		jumpPct := math.Abs((current.Rate-previous.Rate)/previous.Rate) * 100.0
		if jumpPct > cfg.MaxJumpPercent {
			reason := fmt.Sprintf("rate jump of %.2f%% exceeds threshold of %.2f%% (prev: %.4f, cur: %.4f)",
				jumpPct, cfg.MaxJumpPercent, previous.Rate, current.Rate)
			return ValidationResult{QualityStatus: "suspect", QualityReason: &reason, IsValid: true}
		}
	}

	return ValidationResult{
		QualityStatus: "ok",
		QualityReason: nil,
		IsValid:       true,
	}
}
