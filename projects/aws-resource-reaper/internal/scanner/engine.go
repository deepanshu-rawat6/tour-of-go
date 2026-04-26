package scanner

import (
	"context"
	"sync"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"

	"github.com/aws/aws-sdk-go-v2/aws"
	appconfig "github.com/tour-of-go/aws-resource-reaper/internal/config"
)

// AuthProvider resolves an aws.Config for a given account.
type AuthProvider interface {
	ForAccount(ctx context.Context, account appconfig.AccountConfig) (aws.Config, error)
}

// Engine fans out scanning across all account × region pairs.
type Engine struct {
	Auth     AuthProvider
	Scanners []Scanner
}

// Scan runs all scanners across every account × region pair concurrently,
// bounded by cfg.Concurrency. The first error cancels all in-flight work.
func (e *Engine) Scan(ctx context.Context, cfg *appconfig.Config) ([]Resource, error) {
	sem := semaphore.NewWeighted(int64(cfg.Concurrency))
	g, ctx := errgroup.WithContext(ctx)

	var mu sync.Mutex
	var results []Resource

	for _, account := range cfg.Accounts {
		account := account
		awsCfg, err := e.Auth.ForAccount(ctx, account)
		if err != nil {
			return nil, err
		}
		for _, region := range cfg.Regions {
			region := region
			if err := sem.Acquire(ctx, 1); err != nil {
				return nil, err
			}
			g.Go(func() error {
				defer sem.Release(1)
				var batch []Resource
				for _, s := range e.Scanners {
					found, err := s.Scan(ctx, awsCfg, region, account.ID)
					if err != nil {
						return err
					}
					batch = append(batch, found...)
				}
				mu.Lock()
				results = append(results, batch...)
				mu.Unlock()
				return nil
			})
		}
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	return results, nil
}

// DefaultScanners returns all built-in scanners.
func DefaultScanners() []Scanner {
	return []Scanner{
		EBSScanner{},
		ElasticIPScanner{},
		EC2Scanner{},
		RDSSnapshotScanner{},
		ALBScanner{},
		SecurityGroupScanner{},
	}
}
