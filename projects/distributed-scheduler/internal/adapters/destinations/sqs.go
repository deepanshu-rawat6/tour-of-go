package destinations

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"tour_of_go/projects/distributed-scheduler/internal/domain"
)

const sqsBatchSize = 10 // SQS max batch size

// SQSDestination publishes jobs to AWS SQS.
// Supports batch publish (up to 10 per API call) for throughput.
type SQSDestination struct {
	client   *sqs.Client
	queueURL string
	log      *slog.Logger
}

func NewSQSDestination(client *sqs.Client, queueURL string) *SQSDestination {
	return &SQSDestination{client: client, queueURL: queueURL, log: slog.Default()}
}

func (d *SQSDestination) Publish(ctx context.Context, job *domain.Job) error {
	body, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job: %w", err)
	}
	_, err = d.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(d.queueURL),
		MessageBody: aws.String(string(body)),
	})
	return err
}

// BatchPublish sends up to 10 jobs per SQS SendMessageBatch call.
// Returns a map of jobID → error for any failures.
func (d *SQSDestination) BatchPublish(ctx context.Context, jobs []*domain.Job) map[int64]error {
	failures := make(map[int64]error)

	for i := 0; i < len(jobs); i += sqsBatchSize {
		end := i + sqsBatchSize
		if end > len(jobs) {
			end = len(jobs)
		}
		batch := jobs[i:end]

		entries := make([]types.SendMessageBatchRequestEntry, len(batch))
		idToJob := make(map[string]*domain.Job, len(batch))

		for j, job := range batch {
			body, _ := json.Marshal(job)
			msgID := fmt.Sprintf("%d", job.ID)
			entries[j] = types.SendMessageBatchRequestEntry{
				Id:          aws.String(msgID),
				MessageBody: aws.String(string(body)),
			}
			idToJob[msgID] = job
		}

		result, err := d.client.SendMessageBatch(ctx, &sqs.SendMessageBatchInput{
			QueueUrl: aws.String(d.queueURL),
			Entries:  entries,
		})
		if err != nil {
			// Entire batch failed
			for _, job := range batch {
				failures[job.ID] = err
			}
			continue
		}

		for _, failed := range result.Failed {
			if job, ok := idToJob[*failed.Id]; ok {
				failures[job.ID] = fmt.Errorf("SQS error %s: %s", *failed.Code, *failed.Message)
			}
		}
	}

	return failures
}

func (d *SQSDestination) SupportsBatch() bool          { return true }
func (d *SQSDestination) Type() domain.DestinationType { return domain.DestinationSQS }
