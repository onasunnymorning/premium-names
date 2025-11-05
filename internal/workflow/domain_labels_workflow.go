package workflow

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/yourorg/zone-names/internal/types"
)

// DomainLabelsWorkflow processes uploaded files containing domain names/labels
// It extracts domain labels from CSV/TSV/Excel files and saves them to the database
func DomainLabelsWorkflow(ctx workflow.Context, params types.DomainLabelWorkflowParams) (types.DomainLabelProcessResult, error) {
	// Activity options for domain label processing
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		HeartbeatTimeout:    2 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Parse and extract domain labels from the file
	var parseResult types.DomainLabelProcessResult
	if err := workflow.ExecuteActivity(ctx, "Activities.ParseDomainLabelFile", params).Get(ctx, &parseResult); err != nil {
		return types.DomainLabelProcessResult{}, err
	}

	// Step 2: Process and save the labels to the database
	var finalResult types.DomainLabelProcessResult
	if err := workflow.ExecuteActivity(ctx, "Activities.SaveDomainLabels", params, parseResult).Get(ctx, &finalResult); err != nil {
		return types.DomainLabelProcessResult{}, err
	}

	return finalResult, nil
}
