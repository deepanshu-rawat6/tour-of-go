package state

import (
	"testing"
)

var fixtureState = []byte(`{
  "resources": [
    {
      "type": "aws_instance",
      "name": "web",
      "instances": [{"attributes": {"id": "i-abc123", "instance_type": "t3.micro", "ami": "ami-xyz"}}]
    },
    {
      "type": "aws_s3_bucket",
      "name": "data",
      "instances": [{"attributes": {"id": "my-bucket", "bucket": "my-bucket"}}]
    },
    {
      "type": "aws_cloudfront_distribution",
      "name": "cdn",
      "instances": [{"attributes": {"id": "EDFDVBD6EXAMPLE"}}]
    }
  ]
}`)

func TestParse_FiltersUnsupportedTypes(t *testing.T) {
	resources, err := parse(fixtureState)
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 2 {
		t.Errorf("expected 2 supported resources, got %d", len(resources))
	}
}

func TestParse_ExtractsAttributes(t *testing.T) {
	resources, err := parse(fixtureState)
	if err != nil {
		t.Fatal(err)
	}
	ec2 := resources[0]
	if ec2.Type != "aws_instance" {
		t.Errorf("unexpected type: %s", ec2.Type)
	}
	if ec2.ID != "i-abc123" {
		t.Errorf("unexpected ID: %s", ec2.ID)
	}
	if ec2.Attributes["instance_type"] != "t3.micro" {
		t.Errorf("unexpected instance_type: %v", ec2.Attributes["instance_type"])
	}
}

func TestExtractID(t *testing.T) {
	tests := []struct {
		attrs map[string]interface{}
		want  string
	}{
		{map[string]interface{}{"id": "i-abc"}, "i-abc"},
		{map[string]interface{}{"arn": "arn:aws:..."}, "arn:aws:..."},
		{map[string]interface{}{"other": "val"}, ""},
	}
	for _, tc := range tests {
		if got := extractID(tc.attrs); got != tc.want {
			t.Errorf("extractID(%v) = %q, want %q", tc.attrs, got, tc.want)
		}
	}
}
