package providers

import (
	"context"
	"fmt"
)

// BaseClusterConnector provides a basic implementation of ClusterConnector
type BaseClusterConnector struct {
	config ProviderConfig
}

// Connect establishes connection to the cluster
func (c *BaseClusterConnector) Connect(ctx context.Context) error {
	return fmt.Errorf("connection not yet implemented for provider")
}

// Disconnect closes the connection to the cluster
func (c *BaseClusterConnector) Disconnect(ctx context.Context) error {
	return nil
}

// GetClusterInfo returns information about the connected cluster
func (c *BaseClusterConnector) GetClusterInfo(ctx context.Context) (*ClusterInfo, error) {
	return nil, fmt.Errorf("cluster info retrieval not yet implemented")
}

// ExecuteBuild executes a build on the cluster
func (c *BaseClusterConnector) ExecuteBuild(ctx context.Context, build *BuildRequest) (*BuildResult, error) {
	return nil, fmt.Errorf("build execution not yet implemented")
}

// IsHealthy checks if the cluster connection is healthy
func (c *BaseClusterConnector) IsHealthy(ctx context.Context) (bool, error) {
	return false, fmt.Errorf("health check not yet implemented")
}