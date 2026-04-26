package diff

import (
	"fmt"
	"strings"
)

// DriftField represents a single attribute that has drifted.
type DriftField struct {
	Path     string
	Expected interface{}
	Actual   interface{}
}

// hardcodedIgnore lists fields per resource type that AWS returns but TF ignores.
// These are computed/default fields that would cause false positives.
var hardcodedIgnore = map[string][]string{
	"aws_instance": {
		"metadata_options",
		"credit_specification",
		"enclave_options",
		"maintenance_options",
		"private_dns_name_options",
		"root_block_device.0.volume_id",
		"root_block_device.0.device_name",
	},
	"aws_s3_bucket": {
		"arn",
		"bucket_domain_name",
		"bucket_regional_domain_name",
		"hosted_zone_id",
		"region",
		"website_endpoint",
	},
	"aws_security_group": {
		"owner_id",
	},
	"aws_db_instance": {
		"address",
		"endpoint",
		"hosted_zone_id",
		"latest_restorable_time",
		"resource_id",
	},
	"aws_iam_role": {
		"create_date",
		"unique_id",
	},
	"aws_lambda_function": {
		"arn",
		"invoke_arn",
		"last_modified",
		"qualified_arn",
		"version",
		"source_code_hash",
		"source_code_size",
	},
}

// Diff compares expected (from TF state) vs actual (from live AWS API).
// ignoreFields is the user-configured list merged with hardcoded defaults.
// Returns only fields that genuinely differ.
func Diff(resourceType string, expected, actual map[string]interface{}, extraIgnore []string) []DriftField {
	ignore := buildIgnoreSet(resourceType, extraIgnore)
	var drifts []DriftField

	for key, expectedVal := range expected {
		if ignore[key] || hasIgnorePrefix(key, ignore) {
			continue
		}
		actualVal, exists := actual[key]
		if !exists {
			continue // live API may not return all TF fields; not a drift
		}
		if !equal(expectedVal, actualVal) {
			drifts = append(drifts, DriftField{
				Path:     key,
				Expected: expectedVal,
				Actual:   actualVal,
			})
		}
	}
	return drifts
}

func buildIgnoreSet(resourceType string, extra []string) map[string]bool {
	s := make(map[string]bool)
	for _, f := range hardcodedIgnore[resourceType] {
		s[f] = true
	}
	for _, f := range extra {
		s[f] = true
	}
	return s
}

func hasIgnorePrefix(key string, ignore map[string]bool) bool {
	for pattern := range ignore {
		if strings.HasSuffix(pattern, "*") && strings.HasPrefix(key, strings.TrimSuffix(pattern, "*")) {
			return true
		}
	}
	return false
}

// equal compares two values with type coercion for common TF/AWS mismatches.
// TF often stores numbers as strings; AWS returns them as int32/int64.
func equal(a, b interface{}) bool {
	if a == b {
		return true
	}
	// Coerce both to string for comparison
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}
