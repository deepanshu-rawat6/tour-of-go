package stdout

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/tour-of-go/k8s-event-sink/internal/core"
)

// Alerter implements core.AlerterPort writing JSON to stdout.
type Alerter struct{}

func New() *Alerter { return &Alerter{} }

func (a *Alerter) Notify(_ context.Context, event core.Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, string(data))
	return nil
}
