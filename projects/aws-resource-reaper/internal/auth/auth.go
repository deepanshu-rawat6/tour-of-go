package auth

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	appconfig "github.com/tour-of-go/aws-resource-reaper/internal/config"
)

// Provider resolves AWS configs per account, caching assumed-role credentials.
type Provider struct {
	base aws.Config
	mu   sync.Mutex
	cache map[string]aws.Config
}

// New loads the base AWS config from the default credential chain
// (IMDS on EC2, env vars, ~/.aws for local dev).
func New(ctx context.Context) (*Provider, error) {
	base, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading base AWS config: %w", err)
	}
	return &Provider{base: base, cache: make(map[string]aws.Config)}, nil
}

// ForAccount returns an aws.Config scoped to the target account via STS AssumeRole.
// Results are cached so each role is assumed only once regardless of how many regions are scanned.
func (p *Provider) ForAccount(ctx context.Context, account appconfig.AccountConfig) (aws.Config, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if cfg, ok := p.cache[account.ID]; ok {
		return cfg, nil
	}

	stsClient := sts.NewFromConfig(p.base)
	provider := stscreds.NewAssumeRoleProvider(stsClient, account.RoleARN)
	cfg := p.base.Copy()
	cfg.Credentials = aws.NewCredentialsCache(provider)

	p.cache[account.ID] = cfg
	return cfg, nil
}
