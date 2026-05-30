package ledger

import (
	"errors"

	"github.com/deepanshurawat/tour-of-go/projects/event-sourced-ledger/internal/eventstore"
)

type Account struct {
	ID      string
	Balance int64
	Version int
	store   *eventstore.Store
}

func NewAccount(id string, store *eventstore.Store) *Account {
	return &Account{ID: id, store: store}
}

// Hydrate rebuilds account state from events.
func (a *Account) Hydrate() {
	for _, e := range a.store.Load(a.ID) {
		a.apply(e)
	}
}

func (a *Account) apply(e eventstore.Event) {
	switch e.Type {
	case eventstore.AccountOpened:
		// account exists
	case eventstore.MoneyDeposited:
		a.Balance += int64(e.Data["amount"].(float64))
	case eventstore.MoneyWithdrawn:
		a.Balance -= int64(e.Data["amount"].(float64))
	case eventstore.MoneyTransferred:
		a.Balance -= int64(e.Data["amount"].(float64))
	}
	a.Version = e.Version
}

func (a *Account) Open() error {
	err := a.store.Append(a.ID, a.Version, []eventstore.Event{
		{Type: eventstore.AccountOpened, Data: map[string]any{"account_id": a.ID}},
	})
	if err == nil {
		a.Version++
	}
	return err
}

func (a *Account) Deposit(amount int64) error {
	if amount <= 0 {
		return errors.New("amount must be positive")
	}
	a.Hydrate()
	err := a.store.Append(a.ID, a.Version, []eventstore.Event{
		{Type: eventstore.MoneyDeposited, Data: map[string]any{"amount": float64(amount)}},
	})
	if err == nil {
		a.Balance += amount
		a.Version++
	}
	return err
}

func (a *Account) Withdraw(amount int64) error {
	if amount <= 0 {
		return errors.New("amount must be positive")
	}
	a.Hydrate()
	if a.Balance < amount {
		return errors.New("insufficient funds")
	}
	return a.store.Append(a.ID, a.Version, []eventstore.Event{
		{Type: eventstore.MoneyWithdrawn, Data: map[string]any{"amount": float64(amount)}},
	})
}

func (a *Account) Transfer(toID string, amount int64) error {
	if amount <= 0 {
		return errors.New("amount must be positive")
	}
	a.Hydrate()
	if a.Balance < amount {
		return errors.New("insufficient funds")
	}
	// Debit source
	err := a.store.Append(a.ID, a.Version, []eventstore.Event{
		{Type: eventstore.MoneyTransferred, Data: map[string]any{"amount": float64(amount), "to": toID}},
	})
	if err != nil {
		return err
	}
	// Credit destination
	to := NewAccount(toID, a.store)
	to.Hydrate()
	return a.store.Append(toID, to.Version, []eventstore.Event{
		{Type: eventstore.MoneyDeposited, Data: map[string]any{"amount": float64(amount), "from": a.ID}},
	})
}
