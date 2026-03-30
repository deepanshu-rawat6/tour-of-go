// Package search implements the SearchService port using Bleve (embedded search).
//
// Design decision (see docs/adr/010-bleve-over-pg-gin.md):
// Bleve mirrors the Java Hibernate Search + Lucene architecture:
//   - In-process, no network hop
//   - Fast iterative queries for the scheduling algorithm
//   - Supports complex token-based exclusion filters
package search

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"

	"tour_of_go/projects/distributed-scheduler/internal/domain"
)

// jobDoc is the Bleve document structure.
type jobDoc struct {
	ID      string `json:"id"`
	JobName string `json:"jobName"`
	Status  string `json:"status"`
	Tenant  string `json:"tenant"`
}

// BleveSearch implements ports.SearchService.
type BleveSearch struct {
	index bleve.Index
	log   *slog.Logger
}

func NewBleveSearch(indexPath string) (*BleveSearch, error) {
	mapping := bleve.NewIndexMapping()
	var idx bleve.Index
	var err error

	if indexPath == "" || indexPath == ":memory:" {
		idx, err = bleve.NewMemOnly(mapping)
	} else {
		idx, err = bleve.New(indexPath, mapping)
		if err != nil {
			// Open existing index
			idx, err = bleve.Open(indexPath)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("bleve open: %w", err)
	}
	return &BleveSearch{index: idx, log: slog.Default()}, nil
}

func (s *BleveSearch) Index(job *domain.Job) error {
	if !job.Status.IsSchedulable() {
		return s.Remove(job.ID)
	}
	doc := jobDoc{
		ID:      fmt.Sprintf("%d", job.ID),
		JobName: job.JobName,
		Status:  string(job.Status),
		Tenant:  fmt.Sprintf("%d", job.Tenant),
	}
	return s.index.Index(doc.ID, doc)
}

func (s *BleveSearch) Remove(jobID int64) error {
	return s.index.Delete(fmt.Sprintf("%d", jobID))
}

func (s *BleveSearch) MassIndex(jobs []*domain.Job) error {
	batch := s.index.NewBatch()
	for _, job := range jobs {
		if !job.Status.IsSchedulable() {
			continue
		}
		doc := jobDoc{
			ID:      fmt.Sprintf("%d", job.ID),
			JobName: job.JobName,
			Status:  string(job.Status),
			Tenant:  fmt.Sprintf("%d", job.Tenant),
		}
		batch.Index(doc.ID, doc)
	}
	return s.index.Batch(batch)
}

// FindWaitingJobs returns schedulable jobs for a job name, excluding exhausted combinations.
func (s *BleveSearch) FindWaitingJobs(_ context.Context, jobName string, excludeCombinations []map[string]string, limit int) ([]*domain.JobProjection, error) {
	// Build: jobName=X AND status IN (WAITING,RETRY)
	jobNameQ := bleve.NewMatchQuery(jobName)
	jobNameQ.SetField("jobName")

	waitingQ := bleve.NewMatchQuery("WAITING")
	waitingQ.SetField("status")
	retryQ := bleve.NewMatchQuery("RETRY")
	retryQ.SetField("status")
	statusQ := bleve.NewDisjunctionQuery(waitingQ, retryQ)

	var mustNot []query.Query
	for _, combo := range excludeCombinations {
		if tenant, ok := combo["tenant"]; ok {
			tq := bleve.NewMatchQuery(tenant)
			tq.SetField("tenant")
			mustNot = append(mustNot, tq)
		}
	}

	var finalQ query.Query
	if len(mustNot) > 0 {
		boolQ := bleve.NewBooleanQuery()
		boolQ.AddMust(jobNameQ, statusQ)
		boolQ.AddMustNot(mustNot...)
		finalQ = boolQ
	} else {
		boolQ := bleve.NewBooleanQuery()
		boolQ.AddMust(jobNameQ, statusQ)
		finalQ = boolQ
	}

	req := bleve.NewSearchRequestOptions(finalQ, limit, 0, false)
	req.Fields = []string{"*"}

	result, err := s.index.Search(req)
	if err != nil {
		return nil, err
	}

	projs := make([]*domain.JobProjection, 0, len(result.Hits))
	for _, hit := range result.Hits {
		id := int64(0)
		fmt.Sscanf(hit.ID, "%d", &id)
		tenant := 0
		if t, ok := hit.Fields["tenant"].(string); ok {
			fmt.Sscanf(t, "%d", &tenant)
		}
		projs = append(projs, &domain.JobProjection{
			ID:      id,
			JobName: jobName,
			Tenant:  tenant,
			Status:  domain.StatusWaiting,
		})
	}
	return projs, nil
}

// FindDistinctJobNames returns all job names that have schedulable jobs.
func (s *BleveSearch) FindDistinctJobNames(_ context.Context) ([]string, error) {
	waitingQ := bleve.NewMatchQuery("WAITING")
	waitingQ.SetField("status")
	retryQ := bleve.NewMatchQuery("RETRY")
	retryQ.SetField("status")
	q := bleve.NewDisjunctionQuery(waitingQ, retryQ)

	req := bleve.NewSearchRequestOptions(q, 10000, 0, false)
	req.Fields = []string{"jobName"}

	result, err := s.index.Search(req)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	for _, hit := range result.Hits {
		if name, ok := hit.Fields["jobName"].(string); ok {
			seen[name] = struct{}{}
		}
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	return names, nil
}
