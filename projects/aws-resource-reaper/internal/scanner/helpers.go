package scanner

import (
	"fmt"

	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func tagsFromEC2(tags []ec2types.Tag) map[string]string {
	if len(tags) == 0 {
		return nil
	}
	m := make(map[string]string, len(tags))
	for _, t := range tags {
		if t.Key != nil && t.Value != nil {
			m[*t.Key] = *t.Value
		}
	}
	return m
}

func formatInt(n int) string { return fmt.Sprintf("%d", n) }
