package build

import (
	"encoding/json"
	"testing"
)

func TestDeriveProviderArtifactIdentifiers_FromStructuredArtifacts(t *testing.T) {
	payload := []Artifact{
		{Name: "aws", Value: "us-east-1: ami-0a1b2c3d4e5f67890"},
		{Name: "azure", Value: "/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg-images/providers/Microsoft.Compute/images/base-image"},
		{Name: "gcp", Value: "projects/demo-project/global/images/base-image-20260327"},
		{Name: "vmware", Value: "vsphere://dc-1/vm/Templates/base-template"},
	}
	raw, _ := json.Marshal(payload)

	identifiers := deriveProviderArtifactIdentifiers(raw)
	if identifiers == nil {
		t.Fatalf("expected identifiers to be derived")
	}
	if len(identifiers["aws"]) != 1 || identifiers["aws"][0] != "ami-0a1b2c3d4e5f67890" {
		t.Fatalf("expected aws ami identifier, got %#v", identifiers["aws"])
	}
	if len(identifiers["azure"]) != 1 {
		t.Fatalf("expected azure identifier, got %#v", identifiers["azure"])
	}
	if len(identifiers["gcp"]) != 1 {
		t.Fatalf("expected gcp identifier, got %#v", identifiers["gcp"])
	}
	if len(identifiers["vmware"]) != 1 {
		t.Fatalf("expected vmware identifier, got %#v", identifiers["vmware"])
	}
}

func TestDeriveProviderArtifactIdentifiers_FromNestedObjectPayload(t *testing.T) {
	raw := []byte(`{
		"builds": [
			{
				"artifact_id": "us-west-2:ami-0123456789abcdef0",
				"azure_image_id": "/subscriptions/22222222-2222-2222-2222-222222222222/resourceGroups/rg/providers/Microsoft.Compute/images/base-v2",
				"gcp_image": "https://www.googleapis.com/compute/v1/projects/demo/global/images/base-v3",
				"vmware_template": "/dc1/vm/Templates/base-vm-template"
			}
		]
	}`)

	identifiers := deriveProviderArtifactIdentifiers(raw)
	if identifiers == nil {
		t.Fatalf("expected identifiers to be derived")
	}
	if len(identifiers["aws"]) != 1 || identifiers["aws"][0] != "ami-0123456789abcdef0" {
		t.Fatalf("expected aws ami identifier from nested payload, got %#v", identifiers["aws"])
	}
	if len(identifiers["azure"]) != 1 {
		t.Fatalf("expected azure identifier from nested payload, got %#v", identifiers["azure"])
	}
	if len(identifiers["gcp"]) != 1 {
		t.Fatalf("expected gcp identifier from nested payload, got %#v", identifiers["gcp"])
	}
	if len(identifiers["vmware"]) != 1 {
		t.Fatalf("expected vmware identifier from nested payload, got %#v", identifiers["vmware"])
	}
}

func TestDeriveProviderArtifactIdentifiers_ReturnsNilWhenNoIdentifiers(t *testing.T) {
	raw := []byte(`{"message":"build completed without provider-native ids"}`)
	identifiers := deriveProviderArtifactIdentifiers(raw)
	if identifiers != nil {
		t.Fatalf("expected nil identifiers, got %#v", identifiers)
	}
}
