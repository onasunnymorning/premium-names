package activities

import (
	"bufio"
	"container/heap"
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"go.temporal.io/sdk/activity"

	iopkg "github.com/yourorg/zone-names/internal/iopkg"
	znmetrics "github.com/yourorg/zone-names/internal/metrics"
	"github.com/yourorg/zone-names/internal/types"
)

func (a *Activities) MergeSortedAndWriteManifest(ctx context.Context, p types.MergeParams) (types.MergeStats, error) {
	type src struct{ r *bufio.Reader; closer io.Closer; uri string }
	readers := make([]src, 0, len(p.SortedShardURIs))
	for _, u := range p.SortedShardURIs {
		rc, err := iopkg.OpenReader(u)
		if err != nil { return types.MergeStats{}, err }
		readers = append(readers, src{ r: bufio.NewReader(rc), closer: rc, uri: u })
	}

	out, outCloser, err := iopkg.CreateWriter(p.OutURI)
	if err != nil { return types.MergeStats{}, err }
	defer outCloser.Close()
	bw := bufio.NewWriter(out)

	type item struct{ val string; i int }
	h := &minHeap{}; heap.Init(h)
	for i := range readers {
		if s, ok := readLine(readers[i].r); ok { heap.Push(h, item{val: s, i: i}) }
	}

	var last string
	var emitted uint64
	const hbEvery = 50000
	for h.Len() > 0 {
		it := heap.Pop(h).(item)
		if it.val != last {
			if _, err := bw.WriteString(it.val + "\n"); err != nil { return types.MergeStats{}, err }
			last = it.val
			emitted++
			if emitted%hbEvery == 0 { activity.RecordHeartbeat(ctx, emitted) }
		}
		if s, ok := readLine(readers[it.i].r); ok {
			heap.Push(h, item{val: s, i: it.i})
		}
	}
	if err := bw.Flush(); err != nil { return types.MergeStats{}, err }
	for _, s := range readers { _ = s.closer.Close() }

	// manifest
	man := map[string]any{
		"output":      p.OutURI,
		"manifest":    p.ManifestURI,
		"params":      p.Params,
		"total_seen":  p.TotalSeen,
		"shard_stats": p.ShardStats,
		"unique":      emitted,
		"started_at":  time.Now().UTC().Format(time.RFC3339),
	}
	mb, _ := json.MarshalIndent(man, "", "  ")
	mw, cw, err := iopkg.CreateWriter(p.ManifestURI)
	if err == nil {
		_, _ = mw.Write(mb)
		_ = cw.Close()
	}

	// metrics
	znmetrics.MergedEmitted.Add(float64(emitted))
	return types.MergeStats{Emitted: emitted}, nil
}

func readLine(r *bufio.Reader) (string, bool) {
	b, err := r.ReadBytes('\n')
	if err != nil {
		if err == io.EOF && len(b) > 0 {
			return strings.TrimRight(string(b), "\n"), true
		}
		return "", false
	}
	return strings.TrimRight(string(b), "\n"), true
}

type minHeap []item
type item struct{ val string; i int }
func (h minHeap) Len() int { return len(h) }
func (h minHeap) Less(i, j int) bool { return h[i].val < h[j].val }
func (h minHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *minHeap) Push(x any) { *h = append(*h, x.(item)) }
func (h *minHeap) Pop() any { old := *h; n := len(old); x := old[n-1]; *h = old[:n-1]; return x }
