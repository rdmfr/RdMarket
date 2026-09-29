package domain

import (
	"rdmarket-intelligence/backend/internal/models"
	"testing"
	"time"
)

func TestValidateObservation_Valid(t *testing.T) {
	now := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	prev := &models.ExchangeRate{
		Timestamp: now.AddDate(0, 0, -1),
		Rate:      16200.0,
	}
	cur := &models.ExchangeRate{
		Timestamp: now,
		Rate:      16250.0,
	}

	res := ValidateObservation(cur, prev, now, DefaultValidationConfig)
	if !res.IsValid {
		t.Fatalf("expected valid observation, got invalid: %v", res.QualityReason)
	}
	if res.QualityStatus != "ok" {
		t.Errorf("expected quality_status ok, got %s", res.QualityStatus)
	}
	if res.QualityReason != nil {
		t.Errorf("expected nil reason, got %v", *res.QualityReason)
	}
}

func TestValidateObservation_NonPositiveRejected(t *testing.T) {
	now := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	cur := &models.ExchangeRate{
		Timestamp: now,
		Rate:      -100.0,
	}

	res := ValidateObservation(cur, nil, now, DefaultValidationConfig)
	if res.IsValid {
		t.Errorf("negative rate must be rejected")
	}
	if res.QualityStatus != "rejected" {
		t.Errorf("expected quality_status rejected, got %s", res.QualityStatus)
	}
}

func TestValidateObservation_JumpSuspect(t *testing.T) {
	now := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	prev := &models.ExchangeRate{
		Timestamp: now.AddDate(0, 0, -1),
		Rate:      16000.0,
	}
	// 20% jump
	cur := &models.ExchangeRate{
		Timestamp: now,
		Rate:      19500.0,
	}

	res := ValidateObservation(cur, prev, now, DefaultValidationConfig)
	if !res.IsValid {
		t.Errorf("jump should be suspect and stored, not rejected")
	}
	if res.QualityStatus != "suspect" {
		t.Errorf("expected quality_status suspect, got %s", res.QualityStatus)
	}
	if res.QualityReason == nil {
		t.Errorf("expected quality reason for jump")
	}
}

func TestValidateObservation_StaleSuspect(t *testing.T) {
	now := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	staleTime := now.AddDate(0, 0, -20) // 20 days ago
	cur := &models.ExchangeRate{
		Timestamp: staleTime,
		Rate:      16200.0,
	}

	res := ValidateObservation(cur, nil, now, DefaultValidationConfig)
	if res.QualityStatus != "suspect" {
		t.Errorf("expected suspect for stale observation, got %s", res.QualityStatus)
	}
}

func TestValidateObservation_FutureRejected(t *testing.T) {
	now := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	futureTime := now.AddDate(0, 0, 2)
	cur := &models.ExchangeRate{
		Timestamp: futureTime,
		Rate:      16200.0,
	}

	res := ValidateObservation(cur, nil, now, DefaultValidationConfig)
	if res.IsValid {
		t.Errorf("future observation must be rejected")
	}
	if res.QualityStatus != "rejected" {
		t.Errorf("expected rejected, got %s", res.QualityStatus)
	}
}
