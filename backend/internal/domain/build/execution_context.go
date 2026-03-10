package build

import "context"
import "github.com/google/uuid"

type executionIDContextKey struct{}
type buildContextKey struct{}

// WithExecutionID stores the build execution ID in the context.
func WithExecutionID(ctx context.Context, executionID string) context.Context {
	return context.WithValue(ctx, executionIDContextKey{}, executionID)
}

// ExecutionIDFromContext retrieves the build execution ID from context.
func ExecutionIDFromContext(ctx context.Context) (string, bool) {
	value := ctx.Value(executionIDContextKey{})
	if value == nil {
		return "", false
	}
	id, ok := value.(string)
	return id, ok && id != ""
}

// ResolveExecutionID prefers the execution ID in context and falls back to the provided string.
func ResolveExecutionID(ctx context.Context, fallback string) (uuid.UUID, error) {
	if ctxExecutionID, ok := ExecutionIDFromContext(ctx); ok {
		return uuid.Parse(ctxExecutionID)
	}
	return uuid.Parse(fallback)
}

// WithBuild stores the build object in context for method executors.
func WithBuild(ctx context.Context, build *Build) context.Context {
	return context.WithValue(ctx, buildContextKey{}, build)
}

// BuildFromContext retrieves the build object from context.
func BuildFromContext(ctx context.Context) (*Build, bool) {
	value := ctx.Value(buildContextKey{})
	if value == nil {
		return nil, false
	}
	build, ok := value.(*Build)
	return build, ok && build != nil
}
