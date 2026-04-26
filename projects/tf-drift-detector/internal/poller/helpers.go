package poller

import ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

func flattenEC2Tags(tags []ec2types.Tag) map[string]string {
	m := make(map[string]string, len(tags))
	for _, t := range tags {
		if t.Key != nil && t.Value != nil {
			m[*t.Key] = *t.Value
		}
	}
	return m
}

// DefaultComparators returns all built-in comparators keyed by resource type.
func DefaultComparators() map[string]Comparator {
	comparators := []Comparator{
		EC2Comparator{},
		S3Comparator{},
		SGComparator{},
		RDSComparator{},
		IAMRoleComparator{},
		LambdaComparator{},
	}
	m := make(map[string]Comparator, len(comparators))
	for _, c := range comparators {
		m[c.ResourceType()] = c
	}
	return m
}
