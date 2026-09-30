package economic

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

var (
	ErrCSVTooLarge = errors.New("economic CSV exceeds the configured size limit")
	decimalPattern = regexp.MustCompile(`^[+-]?[0-9]+(?:\.[0-9]+)?$`)
)

type ImportedObservation struct {
	ReferenceDate    time.Time
	PeriodEnd        *time.Time
	Value            string
	ReleaseTimestamp *time.Time
}

type CSVRowResult struct {
	Line        int
	Observation *ImportedObservation
	Reason      string
}

type CSVImportResult struct {
	Rows     []CSVRowResult
	Accepted int
	Rejected int
}

type CSVImportProvider struct{}

func (CSVImportProvider) GetProviderName() string {
	return "csv_import"
}

func (CSVImportProvider) Parse(reader io.Reader, maxBytes int64) (CSVImportResult, error) {
	if maxBytes <= 0 {
		return CSVImportResult{}, fmt.Errorf("CSV size limit must be positive")
	}
	content, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return CSVImportResult{}, fmt.Errorf("read CSV: %w", err)
	}
	if int64(len(content)) > maxBytes {
		return CSVImportResult{}, ErrCSVTooLarge
	}

	csvReader := csv.NewReader(bytes.NewReader(content))
	csvReader.FieldsPerRecord = -1
	header, err := csvReader.Read()
	if err != nil {
		return CSVImportResult{}, fmt.Errorf("read CSV header: %w", err)
	}
	columns, err := validateCSVHeader(header)
	if err != nil {
		return CSVImportResult{}, err
	}

	result := CSVImportResult{Rows: make([]CSVRowResult, 0)}
	seenDates := make(map[string]struct{})
	line := 1
	for {
		record, readErr := csvReader.Read()
		if errors.Is(readErr, io.EOF) {
			break
		}
		line++
		if readErr != nil {
			result.Rows = append(result.Rows, CSVRowResult{Line: line, Reason: "Malformed CSV row"})
			result.Rejected++
			continue
		}
		observation, reason := parseCSVObservation(record, columns)
		if reason == "" {
			key := observation.ReferenceDate.Format("2006-01-02")
			if _, exists := seenDates[key]; exists {
				reason = "Duplicate reference_date in this file"
			} else {
				seenDates[key] = struct{}{}
			}
		}
		if reason != "" {
			result.Rows = append(result.Rows, CSVRowResult{Line: line, Reason: reason})
			result.Rejected++
			continue
		}
		result.Rows = append(result.Rows, CSVRowResult{Line: line, Observation: &observation})
		result.Accepted++
	}
	return result, nil
}

func validateCSVHeader(header []string) (map[string]int, error) {
	columns := make(map[string]int, len(header))
	allowed := map[string]bool{"reference_date": true, "value": true, "period_end": true, "release_timestamp": true}
	for index, raw := range header {
		name := strings.ToLower(strings.TrimSpace(raw))
		if !allowed[name] {
			return nil, fmt.Errorf("unsupported CSV column %q", raw)
		}
		if _, exists := columns[name]; exists {
			return nil, fmt.Errorf("duplicate CSV column %q", name)
		}
		columns[name] = index
	}
	if _, ok := columns["reference_date"]; !ok {
		return nil, fmt.Errorf("CSV requires reference_date column")
	}
	if _, ok := columns["value"]; !ok {
		return nil, fmt.Errorf("CSV requires value column")
	}
	return columns, nil
}

func parseCSVObservation(record []string, columns map[string]int) (ImportedObservation, string) {
	if len(record) != len(columns) {
		return ImportedObservation{}, "Column count does not match the header"
	}
	referenceDate, err := time.Parse("2006-01-02", strings.TrimSpace(record[columns["reference_date"]]))
	if err != nil {
		return ImportedObservation{}, "reference_date must use YYYY-MM-DD"
	}
	value := strings.TrimSpace(record[columns["value"]])
	if !validDecimal(value) {
		return ImportedObservation{}, "value must be a decimal with at most 14 integer and 10 fractional digits"
	}
	observation := ImportedObservation{ReferenceDate: referenceDate.UTC(), Value: value}
	if index, ok := columns["period_end"]; ok && strings.TrimSpace(record[index]) != "" {
		periodEnd, parseErr := time.Parse("2006-01-02", strings.TrimSpace(record[index]))
		if parseErr != nil || periodEnd.Before(referenceDate) {
			return ImportedObservation{}, "period_end must be a date on or after reference_date"
		}
		periodEnd = periodEnd.UTC()
		observation.PeriodEnd = &periodEnd
	}
	if index, ok := columns["release_timestamp"]; ok && strings.TrimSpace(record[index]) != "" {
		releaseAt, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(record[index]))
		if parseErr != nil || releaseAt.Before(referenceDate) {
			return ImportedObservation{}, "release_timestamp must use RFC3339 or be empty"
		}
		releaseAt = releaseAt.UTC()
		observation.ReleaseTimestamp = &releaseAt
	}
	return observation, ""
}

func validDecimal(value string) bool {
	if !decimalPattern.MatchString(value) {
		return false
	}
	unsigned := strings.TrimPrefix(strings.TrimPrefix(value, "+"), "-")
	parts := strings.SplitN(unsigned, ".", 2)
	integerDigits := len(strings.TrimLeft(parts[0], "0"))
	if integerDigits == 0 {
		integerDigits = 1
	}
	if integerDigits > 14 {
		return false
	}
	return len(parts) == 1 || len(parts[1]) <= 10
}
