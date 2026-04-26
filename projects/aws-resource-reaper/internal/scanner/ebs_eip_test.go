package scanner

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// mockEC2 implements ec2API for testing.
type mockEC2 struct {
	volumes    []ec2types.Volume
	addresses  []ec2types.Address
	instances  []ec2types.Reservation
	enis       []ec2types.NetworkInterface
	secGroups  []ec2types.SecurityGroup
}

func (m *mockEC2) DescribeVolumes(_ context.Context, in *ec2.DescribeVolumesInput, _ ...func(*ec2.Options)) (*ec2.DescribeVolumesOutput, error) {
	return &ec2.DescribeVolumesOutput{Volumes: m.volumes}, nil
}
func (m *mockEC2) DescribeAddresses(_ context.Context, _ *ec2.DescribeAddressesInput, _ ...func(*ec2.Options)) (*ec2.DescribeAddressesOutput, error) {
	return &ec2.DescribeAddressesOutput{Addresses: m.addresses}, nil
}
func (m *mockEC2) DescribeInstances(_ context.Context, _ *ec2.DescribeInstancesInput, _ ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	return &ec2.DescribeInstancesOutput{Reservations: m.instances}, nil
}
func (m *mockEC2) DescribeNetworkInterfaces(_ context.Context, _ *ec2.DescribeNetworkInterfacesInput, _ ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error) {
	return &ec2.DescribeNetworkInterfacesOutput{NetworkInterfaces: m.enis}, nil
}
func (m *mockEC2) DescribeSecurityGroups(_ context.Context, _ *ec2.DescribeSecurityGroupsInput, _ ...func(*ec2.Options)) (*ec2.DescribeSecurityGroupsOutput, error) {
	return &ec2.DescribeSecurityGroupsOutput{SecurityGroups: m.secGroups}, nil
}

func TestEBSScanner(t *testing.T) {
	mock := &mockEC2{
		volumes: []ec2types.Volume{
			{VolumeId: aws.String("vol-aaa"), Size: aws.Int32(100), VolumeType: ec2types.VolumeTypeGp2},
			{VolumeId: aws.String("vol-bbb"), Size: aws.Int32(50), VolumeType: ec2types.VolumeTypeGp3},
		},
	}
	resources, err := scanEBS(context.Background(), mock, "us-east-1", "123456789012")
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 2 {
		t.Errorf("expected 2 resources, got %d", len(resources))
	}
	if resources[0].ID != "vol-aaa" {
		t.Errorf("unexpected ID: %s", resources[0].ID)
	}
	if resources[0].Type != "ebs-volume" {
		t.Errorf("unexpected type: %s", resources[0].Type)
	}
}

func TestElasticIPScanner(t *testing.T) {
	mock := &mockEC2{
		addresses: []ec2types.Address{
			{AllocationId: aws.String("eipalloc-111"), PublicIp: aws.String("1.2.3.4")},                                          // unassociated
			{AllocationId: aws.String("eipalloc-222"), PublicIp: aws.String("5.6.7.8"), AssociationId: aws.String("eipassoc-x")}, // in use
		},
	}
	resources, err := scanElasticIPs(context.Background(), mock, "us-east-1", "123456789012")
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 {
		t.Errorf("expected 1 unassociated EIP, got %d", len(resources))
	}
	if resources[0].ID != "eipalloc-111" {
		t.Errorf("unexpected ID: %s", resources[0].ID)
	}
}
