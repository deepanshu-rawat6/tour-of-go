package core

import (
	"context"
	"fmt"
	"os"
)

// MultiAlerter fans out to all configured alerters.
type MultiAlerter struct {
	alerters []AlerterPort
}

func NewMultiAlerter(alerters ...AlerterPort) *MultiAlerter {
	return &MultiAlerter{alerters: alerters}
}

func (m *MultiAlerter) Notify(ctx context.Context, event Event) error {
	for _, a := range m.alerters {
		if err := a.Notify(ctx, event); err != nil {
			fmt.Fprintf(os.Stderr, "alerter error: %v\n", err)
		}
	}
	return nil
}
