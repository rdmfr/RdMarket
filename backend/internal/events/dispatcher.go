package events

import (
	"sync"
	"time"
)

type ExchangeRateUpdated struct {
	CurrencyPair string
	ObservedAt   time.Time
	Source       string
}

type EconomicObservationsUpdated struct {
	SeriesCode    string
	ReferenceDate time.Time
	Revision      int
	ReleaseAt     *time.Time
}

type Dispatcher interface {
	SubscribeExchangeRateUpdated(func(ExchangeRateUpdated))
	PublishExchangeRateUpdated(ExchangeRateUpdated)
	SubscribeEconomicObservationsUpdated(func(EconomicObservationsUpdated))
	PublishEconomicObservationsUpdated(EconomicObservationsUpdated)
}

type InProcessDispatcher struct {
	mu                      sync.RWMutex
	exchangeRateSubscribers []func(ExchangeRateUpdated)
	economicDataSubscribers []func(EconomicObservationsUpdated)
}

func NewInProcessDispatcher() *InProcessDispatcher {
	return &InProcessDispatcher{}
}

func (d *InProcessDispatcher) SubscribeExchangeRateUpdated(handler func(ExchangeRateUpdated)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.exchangeRateSubscribers = append(d.exchangeRateSubscribers, handler)
}

func (d *InProcessDispatcher) PublishExchangeRateUpdated(event ExchangeRateUpdated) {
	d.mu.RLock()
	handlers := append([]func(ExchangeRateUpdated){}, d.exchangeRateSubscribers...)
	d.mu.RUnlock()
	for _, handler := range handlers {
		handler(event)
	}
}

func (d *InProcessDispatcher) SubscribeEconomicObservationsUpdated(handler func(EconomicObservationsUpdated)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.economicDataSubscribers = append(d.economicDataSubscribers, handler)
}

func (d *InProcessDispatcher) PublishEconomicObservationsUpdated(event EconomicObservationsUpdated) {
	d.mu.RLock()
	handlers := append([]func(EconomicObservationsUpdated){}, d.economicDataSubscribers...)
	d.mu.RUnlock()
	for _, handler := range handlers {
		handler(event)
	}
}
