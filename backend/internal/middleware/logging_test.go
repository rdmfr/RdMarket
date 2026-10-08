package middleware

import (
	"strings"
	"testing"
)

func TestRedactSecrets(t *testing.T) {
	input := "password=super-secret token=metrics-token token=forecast-token database_url=postgres://user:super-secret@localhost/db"
	got := RedactSecrets(input)
	for _, forbidden := range []string{"super-secret", "metrics-token", "forecast-token"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("secret still present after redaction: %q", forbidden)
		}
	}
	if !strings.Contains(got, "REDACTED") {
		t.Fatal("expected redacted markers in output")
	}
}
