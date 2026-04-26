package poller

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"

	"github.com/tour-of-go/tf-drift-detector/internal/diff"
	"github.com/tour-of-go/tf-drift-detector/internal/state"
)

// DriftResult holds the diff outcome for a single resource.
type DriftResult struct {
	Resource state.ManagedResource
	Drifted  bool
	Fields   []diff.DriftField
}

// Engine fans out live polling across all resources concurrently.
type Engine struct {
	Comparators  map[string]Comparator
	IgnoreFields map[string][]string // resource_type → extra ignore fields
	Concurrency  int
}

// Poll fetches live state for all resources and diffs against TF state.
func (e *Engine) Poll(ctx context.Context, cfg aws.Config, resources []state.ManagedResource) ([]DriftResult, error) {
	sem := semaphore.NewWeighted(int64(e.Concurrency))
	g, ctx := errgroup.WithContext(ctx)

	var mu sync.Mutex
	var results []DriftResult

	for _, res := range resources {
		res := res
		comparator, ok := e.Comparators[res.Type]
		if !ok {
			continue
		}

		if err := sem.Acquire(ctx, 1); err != nil {
			return nil, err
		}
		g.Go(func() error {
			defer sem.Release(1)

			live, err := comparator.FetchLive(ctx, cfg, res.ID)
			if err != nil {
				// Resource not found in AWS = drifted (deleted outside TF)
				mu.Lock()
				results = append(results, DriftResult{
					Resource: res,
					Drifted:  true,
					Fields:   []diff.DriftField{{Path: "existence", Expected: "exists", Actual: "not found: " + err.Error()}},
				})
				mu.Unlock()
				return nil
			}

			fields := diff.Diff(res.Type, res.Attributes, live, e.IgnoreFields[res.Type])
			mu.Lock()
			results = append(results, DriftResult{
				Resource: res,
				Drifted:  len(fields) > 0,
				Fields:   fields,
			})
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}
