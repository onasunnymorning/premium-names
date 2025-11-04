package types

type WorkflowParams struct {
	ZoneURI   string // file:// or s3://
	OutputURI string // where names.txt goes (same scheme); manifest.json at same prefix
	Shards    int
	Filters   []string // e.g. ["A","AAAA","CNAME"]
	IDNMode   string   // "alabel"|"ulabel"|"none"
	// Optional relative subdirectory under scratch root where this workflow writes temp files.
	// If empty, activities may use the scratch root directly.
	ScratchSubdir string
	// If true, workflow will skip cleaning up the scratch subdir after completion/failure.
	KeepScratch bool
}

type PartitionResult struct {
	ShardURIs []string
	Records   uint64
	SizeBytes int64
	InputHash string // optional; not filled in this scaffold
}

type ShardDedupeParams struct {
	ShardURI  string // input shard
	OutputURI string // output sorted unique shard
}

type ShardStats struct {
	Total  uint64
	Unique uint64
}

type MergeParams struct {
	SortedShardURIs []string
	OutURI          string // final names.txt
	ManifestURI     string // manifest.json
	Params          WorkflowParams
	ShardStats      []ShardStats
	TotalSeen       uint64
}
type MergeStats struct {
	Emitted uint64
}

// CleanupParams instructs the cleanup activity which subdir to remove.
type CleanupParams struct {
	ScratchSubdir string
}
