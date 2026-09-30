package events

import (
	"testing"
	"time"
)

func TestInProcessDispatcherPublishesTypedEvents(t *testing.T) {
	dispatcher := NewInProcessDispatcher()
	observed := time.Date(2026, time.September, 30, 0, 0, 0, 0, time.UTC)
	var gotRate ExchangeRateUpdated
	var gotEconomic EconomicObservationsUpdated
	dispatcher.SubscribeExchangeRateUpdated(func(event ExchangeRateUpdated) { gotRate = event })
	dispatcher.SubscribeEconomicObservationsUpdated(func(event EconomicObservationsUpdated) { gotEconomic = event })

	dispatcher.PublishExchangeRateUpdated(ExchangeRateUpdated{CurrencyPair: "USD/IDR", ObservedAt: observed, Source: "test"})
	dispatcher.PublishEconomicObservationsUpdated(EconomicObservationsUpdated{SeriesCode: "TEST", ReferenceDate: observed, Revision: 1})

	if gotRate.CurrencyPair != "USD/IDR" || !gotRate.ObservedAt.Equal(observed) || gotRate.Source != "test" {
		t.Fatalf("unexpected exchange rate event: %+v", gotRate)
	}
	if gotEconomic.SeriesCode != "TEST" || !gotEconomic.ReferenceDate.Equal(observed) || gotEconomic.Revision != 1 {
		t.Fatalf("unexpected economic event: %+v", gotEconomic)
	}
}
