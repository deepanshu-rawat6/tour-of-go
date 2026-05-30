package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/deepanshurawat/tour-of-go/projects/saga-orchestrator/internal/saga"
	"github.com/deepanshurawat/tour-of-go/projects/saga-orchestrator/internal/services"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	steps := []saga.Step{
		{Name: "CreateOrder", Action: services.CreateOrder, Compensate: services.CancelOrder},
		{Name: "ChargePayment", Action: services.ChargePayment, Compensate: services.RefundPayment},
		{Name: "ReserveInventory", Action: services.ReserveInventory, Compensate: services.ReleaseInventory},
		{Name: "ScheduleShipping", Action: services.ScheduleShipping, Compensate: nil},
	}

	orch := saga.NewOrchestrator(steps)
	ctx := context.Background()

	// Scenario 1: Successful saga
	fmt.Println("========== Saga 1: Happy Path ==========")
	data := map[string]any{"order_id": "ORD-001", "amount": 99.99, "item": "widget"}
	if err := orch.Execute(ctx, "saga-001", data); err != nil {
		fmt.Printf("FAILED: %v\n", err)
	}

	// Scenario 2: Payment fails → compensate order
	fmt.Println("\n========== Saga 2: Payment Fails ==========")
	data2 := map[string]any{"order_id": "ORD-002", "amount": 50000.0, "item": "expensive-thing"}
	if err := orch.Execute(ctx, "saga-002", data2); err != nil {
		fmt.Printf("FAILED: %v\n", err)
	}

	// Scenario 3: Inventory fails → compensate payment + order
	fmt.Println("\n========== Saga 3: Inventory Fails ==========")
	data3 := map[string]any{"order_id": "ORD-003", "amount": 25.0, "item": "out-of-stock-item"}
	if err := orch.Execute(ctx, "saga-003", data3); err != nil {
		fmt.Printf("FAILED: %v\n", err)
	}
}
