package worker

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

// EnsureDefaultNamespace tries to make sure that the namespace
// that this app uses is existent.
//
// Returns an error when the namespace cannot be created or described.
func EnsureDefaultNamespace(ctx context.Context, cli workflowservice.WorkflowServiceClient) error {
	// First see if it exists
	ns, err := cli.DescribeNamespace(ctx, &workflowservice.DescribeNamespaceRequest{
		Namespace: "default",
	})
	if err != nil {
		return fmt.Errorf("error describing default namespace: %s", err)
	}
	if ns != nil {
		return nil
	}

	if _, err := cli.RegisterNamespace(ctx, &workflowservice.RegisterNamespaceRequest{
		Namespace:                        "default",
		WorkflowExecutionRetentionPeriod: durationpb.New(72 * time.Hour),
	}); err != nil {
		return fmt.Errorf("error creating namespace: %s", err)
	}

	return nil
}
