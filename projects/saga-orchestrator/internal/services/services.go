package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

// Mock services simulating microservice calls.

func CreateOrder(_ context.Context, data map[string]any) error {
	slog.Info("  → Order created", "order_id", data["order_id"])
	return nil
}

func CancelOrder(_ context.Context, data map[string]any) error {
	slog.Info("  ← Order cancelled", "order_id", data["order_id"])
	return nil
}

func ChargePayment(_ context.Context, data map[string]any) error {
	amount := data["amount"].(float64)
	if amount > 10000 {
		return errors.New("payment declined: amount exceeds limit")
	}
	slog.Info("  → Payment charged", "amount", fmt.Sprintf("$%.2f", amount))
	return nil
}

func RefundPayment(_ context.Context, data map[string]any) error {
	slog.Info("  ← Payment refunded", "amount", data["amount"])
	return nil
}

func ReserveInventory(_ context.Context, data map[string]any) error {
	item := data["item"].(string)
	if item == "out-of-stock-item" {
		return errors.New("inventory: item out of stock")
	}
	slog.Info("  → Inventory reserved", "item", item)
	return nil
}

func ReleaseInventory(_ context.Context, data map[string]any) error {
	slog.Info("  ← Inventory released", "item", data["item"])
	return nil
}

func ScheduleShipping(_ context.Context, data map[string]any) error {
	slog.Info("  → Shipping scheduled", "order_id", data["order_id"])
	return nil
}
