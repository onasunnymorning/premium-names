package activities

import (
	"bufio"
	"compress/gzip"
	"context"
	"hash/fnv"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"go.temporal.io/sdk/activity"

	"github.com/miekg/dns"
	"golang.org/x/net/idna"

	iopkg "github.com/yourorg/zone-names/internal/iopkg"
	znmetrics "github.com/yourorg/zone-names/internal/metrics"
	"github.com/yourorg/zone-names/internal/types"
)

type Config struct {
	ScratchDir string
}

type Activities struct {
	cfg Config
}

func New(cfg Config) *Activities { return &Activities{cfg: cfg} }

func (a *Activities) StreamPartition(ctx context.Context, p types.WorkflowParams) (types.PartitionResult, error) {
	rc, size, err := iopkg.Open(p.ZoneURI)
	if err != nil {
		return types.PartitionResult{}, err
	}
	defer rc.Close()

	var r io.Reader = rc
	if strings.HasSuffix(strings.ToLower(p.ZoneURI), ".gz") {
		gr, err := gzip.NewReader(rc)
		if err != nil {
			return types.PartitionResult{}, err
		}
		defer gr.Close()
		r = gr
	}

	shards := p.Shards
	if shards <= 0 {
		shards = 32
	}

	paths := make([]string, shards)
	wrs := make([]*bufio.Writer, shards)
	closers := make([]io.Closer, shards)
	for i := 0; i < shards; i++ {
		path := filepath.Join(a.cfg.ScratchDir, "shard-%02d.txt")
		path = strings.Replace(path, "%02d", two(i), 1)
		w, c, err := iopkg.Create(path)
		if err != nil {
			return types.PartitionResult{}, err
		}
		paths[i] = "file://" + path
		wrs[i] = bufio.NewWriterSize(w, 1<<20)
		closers[i] = c
	}
	defer func() {
		for i := range wrs {
			if wrs[i] != nil {
				_ = wrs[i].Flush()
			}
			if closers[i] != nil {
				_ = closers[i].Close()
			}
		}
	}()

	filter := make(map[uint16]bool)
	for _, t := range p.Filters {
		filter[typeFromString(strings.ToUpper(t))] = true
	}

	var toASCII, toUnicode func(string) (string, error)
	switch p.IDNMode {
	case "alabel":
		toASCII = idna.ToASCII
	case "ulabel":
		toUnicode = idna.ToUnicode
	}

	zp := dns.NewZoneParser(r, "", "")
	var n uint64
	var lastReported uint64
	const hbEvery = 10000
	for {
		rr, ok := zp.Next()
		if !ok {
			break
		}
		if err := zp.Err(); err != nil {
			return types.PartitionResult{}, err
		}
		h := rr.Header()
		if len(filter) > 0 && !filter[h.Rrtype] {
			continue
		}

		owner := strings.ToLower(strings.TrimSuffix(h.Name, "."))
		var err error
		if toASCII != nil {
			owner, err = toASCII(owner)
		} else if toUnicode != nil {
			owner, err = toUnicode(owner)
		}
		if err != nil {
			continue
		}

		idx := int(fnv32a(owner) % uint32(shards))
		if _, err := wrs[idx].WriteString(owner + "\n"); err != nil {
			return types.PartitionResult{}, err
		}

		n++
		if n%hbEvery == 0 {
			activity.RecordHeartbeat(ctx, map[string]any{"records": n})
			znmetrics.RecordsPartitioned.Add(float64(n - lastReported))
			lastReported = n
		}
	}

	if n > lastReported {
		znmetrics.RecordsPartitioned.Add(float64(n - lastReported))
	}
	for _, bw := range wrs {
		_ = bw.Flush()
	}
	return types.PartitionResult{ShardURIs: paths, Records: n, SizeBytes: size}, nil
}

func typeFromString(s string) uint16 {
	switch s {
	case "A":
		return dns.TypeA
	case "AAAA":
		return dns.TypeAAAA
	case "CNAME":
		return dns.TypeCNAME
	case "MX":
		return dns.TypeMX
	case "TXT":
		return dns.TypeTXT
	case "NS":
		return dns.TypeNS
	case "SRV":
		return dns.TypeSRV
	default:
		return 0
	}
}

func fnv32a(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}
func two(i int) string {
	if i < 10 {
		return "0" + itoa(i)
	}
	return itoa(i)
}
func itoa(i int) string { return fmtInt(i) }

func fmtInt(i int) string { return strconv.Itoa(i) }
