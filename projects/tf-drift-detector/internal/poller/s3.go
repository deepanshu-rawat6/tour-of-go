package poller

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type s3API interface {
	GetBucketVersioning(ctx context.Context, in *s3.GetBucketVersioningInput, opts ...func(*s3.Options)) (*s3.GetBucketVersioningOutput, error)
	GetBucketEncryption(ctx context.Context, in *s3.GetBucketEncryptionInput, opts ...func(*s3.Options)) (*s3.GetBucketEncryptionOutput, error)
	GetBucketTagging(ctx context.Context, in *s3.GetBucketTaggingInput, opts ...func(*s3.Options)) (*s3.GetBucketTaggingOutput, error)
}

// S3Comparator fetches live S3 bucket attributes.
type S3Comparator struct{}

func (S3Comparator) ResourceType() string { return "aws_s3_bucket" }

func (S3Comparator) FetchLive(ctx context.Context, cfg aws.Config, id string) (map[string]interface{}, error) {
	return fetchS3(ctx, s3.NewFromConfig(cfg), id)
}

func fetchS3(ctx context.Context, client s3API, bucket string) (map[string]interface{}, error) {
	attrs := map[string]interface{}{"bucket": bucket}

	if v, err := client.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{Bucket: aws.String(bucket)}); err == nil {
		attrs["versioning"] = string(v.Status)
	}

	if enc, err := client.GetBucketEncryption(ctx, &s3.GetBucketEncryptionInput{Bucket: aws.String(bucket)}); err == nil {
		if len(enc.ServerSideEncryptionConfiguration.Rules) > 0 {
			rule := enc.ServerSideEncryptionConfiguration.Rules[0]
			if rule.ApplyServerSideEncryptionByDefault != nil {
				attrs["sse_algorithm"] = string(rule.ApplyServerSideEncryptionByDefault.SSEAlgorithm)
			}
		}
	}

	if tags, err := client.GetBucketTagging(ctx, &s3.GetBucketTaggingInput{Bucket: aws.String(bucket)}); err == nil {
		attrs["tags"] = flattenS3Tags(tags.TagSet)
	}

	return attrs, nil
}

func flattenS3Tags(tags []s3types.Tag) map[string]string {
	m := make(map[string]string, len(tags))
	for _, t := range tags {
		m[aws.ToString(t.Key)] = aws.ToString(t.Value)
	}
	return m
}
