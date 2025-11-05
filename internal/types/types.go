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

// DomainLabelWorkflowParams defines the input for processing domain label files
type DomainLabelWorkflowParams struct {
	FileURI     string   `json:"file_uri"`    // file:// or s3:// path to the input file
	Tags        []string `json:"tags"`        // tags to apply to all processed labels
	CreatedBy   string   `json:"created_by"`  // email/user who created these labels
	Description string   `json:"description"` // optional description for this batch
}

// DomainLabelProcessResult contains the results of processing domain labels
type DomainLabelProcessResult struct {
	ProcessedCount int                    `json:"processed_count"` // total rows processed
	SavedCount     int                    `json:"saved_count"`     // labels successfully saved
	SkippedCount   int                    `json:"skipped_count"`   // invalid/duplicate labels skipped
	ErrorCount     int                    `json:"error_count"`     // processing errors
	Labels         []ProcessedDomainLabel `json:"labels"`          // list of processed labels
	Errors         []string               `json:"errors"`          // list of processing errors
}

// ProcessedDomainLabel represents a single processed domain label
type ProcessedDomainLabel struct {
	ID       uint     `json:"id"`       // database ID if saved
	Label    string   `json:"label"`    // normalized domain label
	Original string   `json:"original"` // original input from file
	Tags     []string `json:"tags"`     // assigned tag names
	Created  bool     `json:"created"`  // true if newly created, false if existed
}
