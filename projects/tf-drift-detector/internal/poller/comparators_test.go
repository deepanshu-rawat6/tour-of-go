package poller

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// --- EC2 mock ---

type mockEC2 struct{}

func (mockEC2) DescribeInstances(_ context.Context, _ *ec2.DescribeInstancesInput, _ ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	return &ec2.DescribeInstancesOutput{
		Reservations: []ec2types.Reservation{{
			Instances: []ec2types.Instance{{
				InstanceId:   aws.String("i-abc123"),
				InstanceType: ec2types.InstanceTypeT3Micro,
				ImageId:      aws.String("ami-xyz"),
				SubnetId:     aws.String("subnet-1"),
				VpcId:        aws.String("vpc-1"),
				Tags:         []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("web")}},
			}},
		}},
	}, nil
}

func TestEC2Comparator(t *testing.T) {
	attrs, err := fetchEC2(context.Background(), mockEC2{}, "i-abc123")
	if err != nil {
		t.Fatal(err)
	}
	if attrs["instance_type"] != "t3.micro" {
		t.Errorf("unexpected instance_type: %v", attrs["instance_type"])
	}
	if attrs["ami"] != "ami-xyz" {
		t.Errorf("unexpected ami: %v", attrs["ami"])
	}
}

// --- S3 mock ---

type mockS3 struct{}

func (mockS3) GetBucketVersioning(_ context.Context, _ *s3.GetBucketVersioningInput, _ ...func(*s3.Options)) (*s3.GetBucketVersioningOutput, error) {
	return &s3.GetBucketVersioningOutput{Status: s3types.BucketVersioningStatusEnabled}, nil
}
func (mockS3) GetBucketEncryption(_ context.Context, _ *s3.GetBucketEncryptionInput, _ ...func(*s3.Options)) (*s3.GetBucketEncryptionOutput, error) {
	return &s3.GetBucketEncryptionOutput{
		ServerSideEncryptionConfiguration: &s3types.ServerSideEncryptionConfiguration{
			Rules: []s3types.ServerSideEncryptionRule{{
				ApplyServerSideEncryptionByDefault: &s3types.ServerSideEncryptionByDefault{
					SSEAlgorithm: s3types.ServerSideEncryptionAes256,
				},
			}},
		},
	}, nil
}
func (mockS3) GetBucketTagging(_ context.Context, _ *s3.GetBucketTaggingInput, _ ...func(*s3.Options)) (*s3.GetBucketTaggingOutput, error) {
	return &s3.GetBucketTaggingOutput{TagSet: []s3types.Tag{{Key: aws.String("env"), Value: aws.String("prod")}}}, nil
}

func TestS3Comparator(t *testing.T) {
	attrs, err := fetchS3(context.Background(), mockS3{}, "my-bucket")
	if err != nil {
		t.Fatal(err)
	}
	if attrs["versioning"] != "Enabled" {
		t.Errorf("unexpected versioning: %v", attrs["versioning"])
	}
	if attrs["sse_algorithm"] != "AES256" {
		t.Errorf("unexpected sse_algorithm: %v", attrs["sse_algorithm"])
	}
}

func TestDefaultComparators(t *testing.T) {
	m := DefaultComparators()
	for _, typ := range []string{"aws_instance", "aws_s3_bucket", "aws_security_group", "aws_db_instance", "aws_iam_role", "aws_lambda_function"} {
		if _, ok := m[typ]; !ok {
			t.Errorf("missing comparator for %s", typ)
		}
	}
}
