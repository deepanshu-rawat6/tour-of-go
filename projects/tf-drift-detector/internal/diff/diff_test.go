package diff

import "testing"

func TestDiff_NoDrift(t *testing.T) {
	expected := map[string]interface{}{"instance_type": "t3.micro", "ami": "ami-xyz"}
	actual := map[string]interface{}{"instance_type": "t3.micro", "ami": "ami-xyz"}
	drifts := Diff("aws_instance", expected, actual, nil)
	if len(drifts) != 0 {
		t.Errorf("expected no drift, got %d", len(drifts))
	}
}

func TestDiff_DetectsDrift(t *testing.T) {
	expected := map[string]interface{}{"instance_type": "t3.micro"}
	actual := map[string]interface{}{"instance_type": "t3.large"}
	drifts := Diff("aws_instance", expected, actual, nil)
	if len(drifts) != 1 {
		t.Fatalf("expected 1 drift, got %d", len(drifts))
	}
	if drifts[0].Path != "instance_type" {
		t.Errorf("unexpected path: %s", drifts[0].Path)
	}
}

func TestDiff_HardcodedIgnore(t *testing.T) {
	// "arn" is hardcoded-ignored for aws_s3_bucket
	expected := map[string]interface{}{"bucket": "my-bucket", "arn": "arn:old"}
	actual := map[string]interface{}{"bucket": "my-bucket", "arn": "arn:new"}
	drifts := Diff("aws_s3_bucket", expected, actual, nil)
	if len(drifts) != 0 {
		t.Errorf("expected no drift (arn is ignored), got %d", len(drifts))
	}
}

func TestDiff_ConfigIgnore(t *testing.T) {
	expected := map[string]interface{}{"instance_type": "t3.micro", "key_name": "old-key"}
	actual := map[string]interface{}{"instance_type": "t3.micro", "key_name": "new-key"}
	drifts := Diff("aws_instance", expected, actual, []string{"key_name"})
	if len(drifts) != 0 {
		t.Errorf("expected no drift (key_name user-ignored), got %d", len(drifts))
	}
}

func TestDiff_TypeCoercion(t *testing.T) {
	// TF stores memory_size as string "128", AWS returns int32(128)
	expected := map[string]interface{}{"memory_size": "128"}
	actual := map[string]interface{}{"memory_size": int32(128)}
	drifts := Diff("aws_lambda_function", expected, actual, nil)
	if len(drifts) != 0 {
		t.Errorf("expected no drift after type coercion, got %d", len(drifts))
	}
}

func TestDiff_MissingLiveField(t *testing.T) {
	// If live API doesn't return a field, it's not a drift
	expected := map[string]interface{}{"instance_type": "t3.micro", "computed_field": "val"}
	actual := map[string]interface{}{"instance_type": "t3.micro"}
	drifts := Diff("aws_instance", expected, actual, nil)
	if len(drifts) != 0 {
		t.Errorf("expected no drift for missing live field, got %d", len(drifts))
	}
}
