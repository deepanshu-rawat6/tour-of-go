package concurrency

import (
	"sync"
	"testing"

	"tour_of_go/projects/distributed-scheduler/internal/domain"
)

var testRules = map[string]int{
	"$jobName":      5,
	"$tenant_$env":  2,
}

func testConfigFn(jobName string) map[string]int { return testRules }

func makeJob(id int64, tenant int, env string) *domain.JobProjection {
	return &domain.JobProjection{
		ID:                 id,
		JobName:            "DataIngestion",
		Tenant:             tenant,
		ConcurrencyControl: map[string]string{"env": env},
	}
}

func TestEvaluateJobPublish_AllowsWhenCapacityAvailable(t *testing.T) {
	m := NewManager(testConfigFn)
	job := makeJob(1, 1, "prod")
	if !m.EvaluateJobPublish(testRules, job) {
		t.Fatal("expected approval")
	}
	snap := m.Snapshot()
	if snap["DataIngestion"] != 1 {
		t.Errorf("expected DataIngestion=1, got %d", snap["DataIngestion"])
	}
}

func TestEvaluateJobPublish_BlocksAtLimit(t *testing.T) {
	m := NewManager(testConfigFn)
	job := makeJob(1, 1, "prod")
	// Fill the $tenant_$env slot (limit=2)
	m.EvaluateJobPublish(testRules, job)
	m.EvaluateJobPublish(testRules, makeJob(2, 1, "prod"))
	// Third should be blocked
	if m.EvaluateJobPublish(testRules, makeJob(3, 1, "prod")) {
		t.Fatal("expected rejection at limit")
	}
}

func TestHandleFinishedJob_DecrementsAndFloorsAtZero(t *testing.T) {
	m := NewManager(testConfigFn)
	job := makeJob(1, 1, "prod")
	m.EvaluateJobPublish(testRules, job)

	fullJob := &domain.Job{
		JobName:            "DataIngestion",
		Tenant:             1,
		ConcurrencyControl: map[string]string{"env": "prod"},
	}
	m.HandleFinishedJob(fullJob)
	snap := m.Snapshot()
	if snap["DataIngestion"] != 0 {
		t.Errorf("expected 0 after decrement, got %d", snap["DataIngestion"])
	}

	// Double-decrement should not go negative
	m.HandleFinishedJob(fullJob)
	snap = m.Snapshot()
	if snap["DataIngestion"] < 0 {
		t.Error("counter went negative")
	}
}

func TestConcurrentEvaluate_RaceDetector(t *testing.T) {
	m := NewManager(testConfigFn)
	var wg sync.WaitGroup
	approved := make(chan bool, 100)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			job := makeJob(id, 1, "prod")
			approved <- m.EvaluateJobPublish(testRules, job)
		}(int64(i))
	}

	wg.Wait()
	close(approved)

	count := 0
	for ok := range approved {
		if ok {
			count++
		}
	}
	// $tenant_$env limit is 2, $jobName limit is 5 — bottleneck is $tenant_$env=2
	if count > 2 {
		t.Errorf("expected at most 2 approvals (tenant_env limit), got %d", count)
	}
}

func TestKeyGeneration(t *testing.T) {
	job := makeJob(1, 42, "staging")
	keys := GenerateKeys(testRules, job)
	if keys["$jobName"] != "DataIngestion" {
		t.Errorf("expected DataIngestion, got %s", keys["$jobName"])
	}
	if keys["$tenant_$env"] != "42_staging" {
		t.Errorf("expected 42_staging, got %s", keys["$tenant_$env"])
	}
}
