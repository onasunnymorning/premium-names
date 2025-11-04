package activities

import (
	"bufio"
	"context"
	"path/filepath"
	"time"

	"go.temporal.io/sdk/activity"

	"github.com/dgraph-io/badger/v4"
	iopkg "github.com/yourorg/zone-names/internal/iopkg"
	znmetrics "github.com/yourorg/zone-names/internal/metrics"
	"github.com/yourorg/zone-names/internal/types"
)

func (a *Activities) ShardDedupeBadger(ctx context.Context, p types.ShardDedupeParams) (types.ShardStats, error) {
	in, err := iopkg.OpenReader(p.ShardURI)
	if err != nil {
		return types.ShardStats{}, err
	}
	defer in.Close()

	dbpath := filepath.Join(a.cfg.ScratchDir, filepath.Base(p.ShardURI)+".badger")
	opts := badger.DefaultOptions(dbpath).WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		return types.ShardStats{}, err
	}
	defer db.Close()

	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 1024), 1024*1024)
	var total uint64
	lastHB := time.Now()
	for sc.Scan() {
		k := append([]byte(nil), sc.Bytes()...)
		err := db.Update(func(txn *badger.Txn) error {
			_, e := txn.Get(k)
			if e == badger.ErrKeyNotFound {
				return txn.Set(k, []byte{1})
			}
			return nil
		})
		if err != nil {
			return types.ShardStats{}, err
		}
		total++
		// Heartbeat frequently by count and also time-based as a safety net.
		if total%5000 == 0 || time.Since(lastHB) > 10*time.Second {
			activity.RecordHeartbeat(ctx, total)
			lastHB = time.Now()
		}
	}
	if err := sc.Err(); err != nil {
		return types.ShardStats{}, err
	}

	out, closeOut, err := iopkg.CreateWriter(p.OutputURI)
	if err != nil {
		return types.ShardStats{}, err
	}
	defer closeOut.Close()
	bw := bufio.NewWriterSize(out, 1<<20)

	var uniq uint64
	// Continue to heartbeat while iterating and writing out uniques.
	// This phase can be long on large shards, so keep the server updated.
	lastHB = time.Now()
	err = db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.KeyCopy(nil)
			if _, err := bw.Write(k); err != nil {
				return err
			}
			if err := bw.WriteByte('\n'); err != nil {
				return err
			}
			uniq++
			if uniq%10000 == 0 || time.Since(lastHB) > 10*time.Second {
				activity.RecordHeartbeat(ctx, map[string]any{"total": total, "unique": uniq})
				lastHB = time.Now()
			}
		}
		return nil
	})
	if err != nil {
		return types.ShardStats{}, err
	}
	if err := bw.Flush(); err != nil {
		return types.ShardStats{}, err
	}

	// metrics
	znmetrics.DedupeInput.Add(float64(total))
	znmetrics.DedupeUnique.Add(float64(uniq))

	return types.ShardStats{Total: total, Unique: uniq}, nil
}
