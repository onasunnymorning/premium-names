package workflow

import (
	"path"
	"strings"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/yourorg/zone-names/internal/types"
)

func ZoneNamesWorkflow(ctx workflow.Context, p types.WorkflowParams) (types.MergeStats, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 4 * time.Hour,
		HeartbeatTimeout:    1 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	}
	// Base options used for partition. We'll use longer heartbeat timeouts for
	// dedupe and merge, as those phases can run longer between safe heartbeat points.
	ctx = workflow.WithActivityOptions(ctx, ao)
	dedupeAO := ao
	dedupeAO.HeartbeatTimeout = 5 * time.Minute
	dedupeCtx := workflow.WithActivityOptions(ctx, dedupeAO)
	mergeAO := ao
	mergeAO.HeartbeatTimeout = 5 * time.Minute
	mergeCtx := workflow.WithActivityOptions(ctx, mergeAO)

	var part types.PartitionResult
	if err := workflow.ExecuteActivity(ctx, "Activities.StreamPartition", p).Get(ctx, &part); err != nil {
		return types.MergeStats{}, err
	}

	// fan-out dedupe
	stats := make([]types.ShardStats, len(part.ShardURIs))
	futures := make([]workflow.Future, len(part.ShardURIs))
	for i, shard := range part.ShardURIs {
		out := shard + ".sorted"
		dp := types.ShardDedupeParams{ShardURI: shard, OutputURI: out}
		futures[i] = workflow.ExecuteActivity(dedupeCtx, "Activities.ShardDedupeBadger", dp)
	}
	for i := range futures {
		if err := futures[i].Get(ctx, &stats[i]); err != nil {
			return types.MergeStats{}, err
		}
	}

	// build out paths
	outNames := p.OutputURI
	manURI := manifestPath(outNames)

	mp := types.MergeParams{
		SortedShardURIs: make([]string, len(part.ShardURIs)),
		OutURI:          outNames,
		ManifestURI:     manURI,
		Params:          p,
		ShardStats:      stats,
		TotalSeen:       part.Records,
	}
	for i, shard := range part.ShardURIs {
		mp.SortedShardURIs[i] = shard + ".sorted"
	}

	var ms types.MergeStats
	if err := workflow.ExecuteActivity(mergeCtx, "Activities.MergeSortedAndWriteManifest", mp).Get(ctx, &ms); err != nil {
		return types.MergeStats{}, err
	}
	return ms, nil
}

func manifestPath(out string) string {
	// replace "names.txt" with "manifest.json" if present; otherwise append ".manifest.json"
	if strings.HasSuffix(strings.ToLower(out), "names.txt") {
		return out[:len(out)-len("names.txt")] + "manifest.json"
	}
	dir, file := path.Split(out)
	return dir + strings.TrimSuffix(file, ".txt") + ".manifest.json"
}
