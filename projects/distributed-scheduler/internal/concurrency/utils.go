// Package concurrency implements the in-memory concurrency tracking system.
// It is the Go equivalent of Java's InMemoryConcurrencyManager + ConcurrencyUtils.
package concurrency

import (
	"fmt"
	"strings"

	"tour_of_go/projects/distributed-scheduler/internal/domain"
)

// GenerateKeys expands rule templates into concrete concurrency keys for a given job.
//
// Rule templates use $-prefixed placeholders that are substituted with job properties:
//   - $jobName  → job.JobName  (e.g., "DataIngestion")
//   - $tenant   → job.Tenant   (e.g., "42")
//   - Any key in job.ConcurrencyControl (e.g., $env → "prod")
//
// Example:
//
//	rules:   {"$jobName": 10, "$tenant_$env": 3}
//	job:     {JobName:"DataIngestion", Tenant:1, ConcurrencyControl:{"env":"prod"}}
//	result:  {"$jobName":"DataIngestion", "$tenant_$env":"1_prod"}
func GenerateKeys(rules map[string]int, job *domain.JobProjection) map[string]string {
	props := buildProps(job)
	result := make(map[string]string, len(rules))
	for template := range rules {
		result[template] = expandTemplate(template, props)
	}
	return result
}

// GenerateConcurrencyMap returns a map of concrete key → limit for a job.
// Used by EvaluateJobPublish to check and increment counters.
func GenerateConcurrencyMap(rules map[string]int, job *domain.JobProjection) map[string]int {
	props := buildProps(job)
	result := make(map[string]int, len(rules))
	for template, limit := range rules {
		result[expandTemplate(template, props)] = limit
	}
	return result
}

func buildProps(job *domain.JobProjection) map[string]string {
	props := map[string]string{
		"$jobName": job.JobName,
		"$tenant":  fmt.Sprintf("%d", job.Tenant),
	}
	for k, v := range job.ConcurrencyControl {
		props["$"+k] = v
	}
	return props
}

func expandTemplate(template string, props map[string]string) string {
	result := template
	for placeholder, value := range props {
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}
