package domain

// DestinationType identifies the execution backend for a job type.
type DestinationType string

const (
	DestinationSQS      DestinationType = "SQS"
	DestinationInMemory DestinationType = "IN_MEMORY"
)

// DestinationInfo holds the routing config for a job type's execution backend.
type DestinationInfo struct {
	Type    DestinationType
	QueueURL string // for SQS
	// Priority-based routing: priority level → queue URL
	PriorityQueues map[int]string
}

// JobConfig holds the scheduling rules for a named job type.
// ConcurrencyRules maps rule templates to their limits.
//
// Example:
//
//	ConcurrencyRules: {
//	  "$jobName":       10,   // max 10 DataIngestion jobs globally
//	  "$tenant_$env":   3,    // max 3 per tenant+env combination
//	}
type JobConfig struct {
	JobName          string
	ConcurrencyRules map[string]int
	Destination      DestinationInfo
}
