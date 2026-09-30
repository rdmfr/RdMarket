package economic

import (
	"strings"
	"testing"
)

func TestLoadCatalogHasValidatedBLSSeries(t *testing.T) {
	series, err := LoadCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if len(series) != 2 {
		t.Fatalf("expected two licensed BLS series, got %d", len(series))
	}
	for _, item := range series {
		if !item.IsActive || item.SourceProvider != "bls_api" || item.PublicationLagDays == 0 {
			t.Errorf("unexpected configured series: %+v", item)
		}
	}
}

func TestDecodeCatalogRejectsDuplicateCodesAndZeroPublicationLag(t *testing.T) {
	duplicate := `[{"code":"X","name":"x","country":"x","category":"growth","unit":"u","frequency":"monthly","seasonal_adjustment":false,"source_provider":"manual","source_series_id":"x","source_url":"https://example.test","license_note":"ok","description":"x.","change_mode":"level","trend_threshold":"0.1","publication_lag_days":1,"is_active":true},{"code":"X","name":"x","country":"x","category":"growth","unit":"u","frequency":"monthly","seasonal_adjustment":false,"source_provider":"manual","source_series_id":"x","source_url":"https://example.test","license_note":"ok","description":"x.","change_mode":"level","trend_threshold":"0.1","publication_lag_days":1,"is_active":true}]`
	if _, err := DecodeCatalog(strings.NewReader(duplicate)); err == nil {
		t.Fatal("expected duplicate code to fail")
	}
	zeroLag := strings.Replace(duplicate[:strings.Index(duplicate, "},{")], `"publication_lag_days":1`, `"publication_lag_days":0`, 1) + "]"
	if _, err := DecodeCatalog(strings.NewReader(zeroLag)); err == nil {
		t.Fatal("expected zero publication lag to fail")
	}
}
