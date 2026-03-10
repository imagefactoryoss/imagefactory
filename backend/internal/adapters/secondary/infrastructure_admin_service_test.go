package secondary

import (
	"context"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestNewInfrastructureAdminService(t *testing.T) {
	logger := zap.NewNop()
	svc := NewInfrastructureAdminService(nil, logger)
	if svc == nil {
		t.Fatal("expected service instance")
	}
	if svc.db != nil {
		t.Fatal("expected nil db to be stored")
	}
	if svc.logger != logger {
		t.Fatal("expected logger to be assigned")
	}
}

func TestCreateNodeValidationErrors(t *testing.T) {
	svc := NewInfrastructureAdminService(nil, zap.NewNop())

	_, err := svc.CreateNode(context.Background(), &CreateNodeRequest{
		Name:        "ab",
		TotalCPU:    2,
		TotalMemory: 8,
		TotalDisk:   100,
	})
	if err == nil || !strings.Contains(err.Error(), "name must be at least 3 characters") {
		t.Fatalf("expected name validation error, got: %v", err)
	}

	_, err = svc.CreateNode(context.Background(), &CreateNodeRequest{
		Name:        "node-1",
		TotalCPU:    0.5,
		TotalMemory: 8,
		TotalDisk:   100,
	})
	if err == nil || !strings.Contains(err.Error(), "CPU, memory, and disk must be at least 1") {
		t.Fatalf("expected resource validation error, got: %v", err)
	}
}
