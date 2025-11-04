package types

type WorkflowParams struct {
	ZoneURI   string   // file:// or s3://
	OutputURI string   // where names.txt goes (same scheme); manifest.json at same prefix
	Shards    int
	Filters   []string // e.g. ["A","AAAA","CNAME"]
	IDNMode   string   // "alabel"|"ulabel"|"none"
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
