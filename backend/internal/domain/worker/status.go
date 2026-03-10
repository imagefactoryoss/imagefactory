package worker

// Status represents the health status of a worker
type Status string

const (
	StatusAvailable Status = "available"
	StatusBusy      Status = "busy"
	StatusOffline   Status = "offline"
)

// IsValid checks if the status is valid
func (s Status) IsValid() bool {
	switch s {
	case StatusAvailable, StatusBusy, StatusOffline:
		return true
	default:
		return false
	}
}

// WorkerType represents the type of build worker
type WorkerType string

const (
	WorkerTypeDocker      WorkerType = "docker"
	WorkerTypeKubernetes  WorkerType = "kubernetes"
	WorkerTypeLambda      WorkerType = "lambda"
)

// IsValid checks if the worker type is valid
func (wt WorkerType) IsValid() bool {
	switch wt {
	case WorkerTypeDocker, WorkerTypeKubernetes, WorkerTypeLambda:
		return true
	default:
		return false
	}
}

// Capacity represents the maximum concurrent builds a worker can handle
type Capacity int

// IsValid checks if the capacity is valid
func (c Capacity) IsValid() bool {
	return c > 0
}
