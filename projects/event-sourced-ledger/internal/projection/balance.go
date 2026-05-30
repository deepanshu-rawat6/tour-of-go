package projection

import (
	"github.com/deepanshurawat/tour-of-go/projects/event-sourced-ledger/internal/eventstore"
)

// BalanceProjection is a read model showing current balances.
type BalanceProjection struct {
	Balances map[string]int64
}

func NewBalanceProjection() *BalanceProjection {
	return &BalanceProjection{Balances: make(map[string]int64)}
}

// Rebuild replays all events to build current state.
func (p *BalanceProjection) Rebuild(store *eventstore.Store) {
	p.Balances = make(map[string]int64)
	for _, e := range store.AllEvents() {
		switch e.Type {
		case eventstore.AccountOpened:
			p.Balances[e.AggregateID] = 0
		case eventstore.MoneyDeposited:
			p.Balances[e.AggregateID] += int64(e.Data["amount"].(float64))
		case eventstore.MoneyWithdrawn:
			p.Balances[e.AggregateID] -= int64(e.Data["amount"].(float64))
		case eventstore.MoneyTransferred:
			p.Balances[e.AggregateID] -= int64(e.Data["amount"].(float64))
		}
	}
}
