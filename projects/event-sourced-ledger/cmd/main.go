package main

import (
	"fmt"

	"github.com/deepanshurawat/tour-of-go/projects/event-sourced-ledger/internal/eventstore"
	"github.com/deepanshurawat/tour-of-go/projects/event-sourced-ledger/internal/ledger"
	"github.com/deepanshurawat/tour-of-go/projects/event-sourced-ledger/internal/projection"
)

func main() {
	store := eventstore.New()

	// Open accounts
	alice := ledger.NewAccount("alice", store)
	bob := ledger.NewAccount("bob", store)
	alice.Open()
	bob.Open()

	// Deposit
	alice.Deposit(1000)
	bob.Deposit(500)

	// Transfer
	fmt.Println("=== Event-Sourced Ledger ===")
	fmt.Println("\nAlice deposits $1000, Bob deposits $500")
	fmt.Println("Alice transfers $250 to Bob")

	alice.Transfer("bob", 250)

	// Rebuild projection from event stream
	bp := projection.NewBalanceProjection()
	bp.Rebuild(store)

	fmt.Println("\n--- Balance Projection (rebuilt from events) ---")
	for id, bal := range bp.Balances {
		fmt.Printf("  %s: $%d\n", id, bal)
	}

	// Show event log
	fmt.Println("\n--- Full Event Log ---")
	for _, e := range store.AllEvents() {
		fmt.Printf("  #%d [v%d] %s.%s %v\n", e.ID, e.Version, e.AggregateID, e.Type, e.Data)
	}

	// Demonstrate insufficient funds
	fmt.Println("\n--- Attempting overdraft ---")
	if err := bob.Withdraw(2000); err != nil {
		fmt.Printf("  Error: %v\n", err)
	}
}
