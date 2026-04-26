package scanner

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbtypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
)

// --- EC2 ---

func TestEC2Scanner_FlagsX86(t *testing.T) {
	mock := &mockEC2{
		instances: []ec2types.Reservation{
			{Instances: []ec2types.Instance{
				{InstanceId: aws.String("i-aaa"), InstanceType: "m5.large"},  // should flag
				{InstanceId: aws.String("i-bbb"), InstanceType: "m6g.large"}, // Graviton, skip
				{InstanceId: aws.String("i-ccc"), InstanceType: "c5.xlarge"}, // should flag
			}},
		},
	}
	resources, err := scanEC2(context.Background(), mock, "us-east-1", "123")
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 2 {
		t.Errorf("expected 2 flagged instances, got %d", len(resources))
	}
	if resources[0].Metadata["graviton_suggest"] != "m6g.large" {
		t.Errorf("unexpected suggestion: %s", resources[0].Metadata["graviton_suggest"])
	}
}

// --- RDS ---

type mockRDS struct{ snaps []rdstypes.DBSnapshot }

func (m *mockRDS) DescribeDBSnapshots(_ context.Context, _ *rds.DescribeDBSnapshotsInput, _ ...func(*rds.Options)) (*rds.DescribeDBSnapshotsOutput, error) {
	return &rds.DescribeDBSnapshotsOutput{DBSnapshots: m.snaps}, nil
}

func TestRDSSnapshotScanner(t *testing.T) {
	old := time.Now().AddDate(0, 0, -40)
	recent := time.Now().AddDate(0, 0, -5)
	mock := &mockRDS{snaps: []rdstypes.DBSnapshot{
		{DBSnapshotIdentifier: aws.String("snap-old"), SnapshotCreateTime: &old, AllocatedStorage: aws.Int32(100), Engine: aws.String("mysql"), DBInstanceIdentifier: aws.String("db-1")},
		{DBSnapshotIdentifier: aws.String("snap-new"), SnapshotCreateTime: &recent, AllocatedStorage: aws.Int32(50), Engine: aws.String("mysql"), DBInstanceIdentifier: aws.String("db-2")},
	}}
	resources, err := scanRDSSnapshots(context.Background(), mock, "us-east-1", "123")
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 {
		t.Errorf("expected 1 old snapshot, got %d", len(resources))
	}
	if resources[0].ID != "snap-old" {
		t.Errorf("unexpected ID: %s", resources[0].ID)
	}
}

// --- ALB ---

type mockELB struct{ lbs []elbtypes.LoadBalancer }

func (m *mockELB) DescribeLoadBalancers(_ context.Context, _ *elasticloadbalancingv2.DescribeLoadBalancersInput, _ ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeLoadBalancersOutput, error) {
	return &elasticloadbalancingv2.DescribeLoadBalancersOutput{LoadBalancers: m.lbs}, nil
}

type mockCW struct{ sum float64 }

func (m *mockCW) GetMetricStatistics(_ context.Context, _ *cloudwatch.GetMetricStatisticsInput, _ ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricStatisticsOutput, error) {
	return &cloudwatch.GetMetricStatisticsOutput{
		Datapoints: []cwtypes.Datapoint{{Sum: &m.sum}},
	}, nil
}

func TestALBScanner_ZeroTraffic(t *testing.T) {
	elbMock := &mockELB{lbs: []elbtypes.LoadBalancer{
		{LoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/app/my-alb/abc"), DNSName: aws.String("my-alb.elb.amazonaws.com"), Type: "application"},
	}}
	cwMock := &mockCW{sum: 0}
	resources, err := scanALBs(context.Background(), elbMock, cwMock, "us-east-1", "123")
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 {
		t.Errorf("expected 1 zero-traffic ALB, got %d", len(resources))
	}
}

func TestALBScanner_WithTraffic(t *testing.T) {
	elbMock := &mockELB{lbs: []elbtypes.LoadBalancer{
		{LoadBalancerArn: aws.String("arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/app/my-alb/abc"), DNSName: aws.String("my-alb.elb.amazonaws.com"), Type: "application"},
	}}
	cwMock := &mockCW{sum: 1000}
	resources, err := scanALBs(context.Background(), elbMock, cwMock, "us-east-1", "123")
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 0 {
		t.Errorf("expected 0 resources for ALB with traffic, got %d", len(resources))
	}
}

// --- Security Groups ---

func TestSecurityGroupScanner(t *testing.T) {
	mock := &mockEC2{
		enis: []ec2types.NetworkInterface{
			{Groups: []ec2types.GroupIdentifier{{GroupId: aws.String("sg-used")}}},
		},
		secGroups: []ec2types.SecurityGroup{
			{GroupId: aws.String("sg-used"), GroupName: aws.String("used-sg"), VpcId: aws.String("vpc-1")},
			{GroupId: aws.String("sg-unused"), GroupName: aws.String("unused-sg"), VpcId: aws.String("vpc-1")},
			{GroupId: aws.String("sg-default"), GroupName: aws.String("default"), VpcId: aws.String("vpc-1")},
		},
	}
	resources, err := scanSecurityGroups(context.Background(), mock, "us-east-1", "123")
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 {
		t.Errorf("expected 1 unused SG, got %d", len(resources))
	}
	if resources[0].ID != "sg-unused" {
		t.Errorf("unexpected ID: %s", resources[0].ID)
	}
}

func TestDescribeNetworkInterfaces(_ *testing.T) {
	// Covered by TestSecurityGroupScanner above — ENI mock is exercised there.
}

// Satisfy the ec2API interface for DescribeInstances in mockEC2.
func (m *mockEC2) DescribeInstances2(_ context.Context, in *ec2.DescribeInstancesInput, _ ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	return &ec2.DescribeInstancesOutput{Reservations: m.instances}, nil
}
