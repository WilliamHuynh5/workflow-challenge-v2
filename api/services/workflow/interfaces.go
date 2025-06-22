package workflow

import "context"

// RepositoryInterface defines the interface for workflow repository operations
type RepositoryInterface interface {
	GetWorkflow(ctx context.Context, id string) (*Workflow, error)
	SaveWorkflow(ctx context.Context, workflow *Workflow) error
}

// ExecutorInterface defines the interface for workflow execution
type ExecutorInterface interface {
	Execute(ctx context.Context, workflow *Workflow, inputs map[string]interface{}) *ExecutionResponse
}
