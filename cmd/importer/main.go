package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	xls "github.com/extrame/xls"
	"github.com/xuri/excelize/v2"

	dbpkg "github.com/yourorg/zone-names/internal/db"
	"github.com/yourorg/zone-names/internal/normalize"
	"github.com/yourorg/zone-names/internal/storage"
)

func main() {
	ctx := context.Background()
	cfg := dbpkg.FromEnv()
	pool, err := dbpkg.Connect(ctx, cfg)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	jr := dbpkg.NewJobRepo(pool)
	br := dbpkg.NewBatchRepo(pool)
	lr := dbpkg.NewLabelRepo(pool)

	s3c, err := storage.NewS3(ctx)
	if err != nil {
		log.Fatalf("s3 init: %v", err)
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		// Try to claim a job
		j, err := jr.ClaimNext(ctx)
		if err != nil {
			if errors.Is(err, dbpkg.ErrNotFound) {
				// No job; wait and retry
				<-ticker.C
				continue
			}
			log.Printf("claim error: %v", err)
			<-ticker.C
			continue
		}

		// Process the job
		if err := processJob(ctx, s3c, br, lr, jr, j); err != nil {
			em := err.Error()
			_ = jr.UpdateStatus(ctx, j.ID, "failed", &em, nil)
			log.Printf("job %d failed: %v", j.ID, err)
		}
	}
}

func processJob(ctx context.Context, s storage.ObjectStore, br dbpkg.BatchRepository, lr dbpkg.LabelRepository, jr dbpkg.JobRepository, j dbpkg.Job) error {
	// Validate batch exists
	if _, err := br.Get(ctx, j.BatchID); err != nil {
		return fmt.Errorf("batch %d: %w", j.BatchID, err)
	}
	rc, _, err := s.Get(ctx, j.ObjectURI)
	if err != nil {
		return err
	}
	defer rc.Close()

	// Read a small head to detect content and rebuild reader
	head := make([]byte, 4096)
	n, _ := rc.Read(head)
	head = head[:n]
	rest := io.MultiReader(bytes.NewReader(head), rc)

	// Detect by extension or content signature
	ext := strings.ToLower(path.Ext(j.ObjectURI))
	ct := http.DetectContentType(head)
	var labels []string
	switch {
	case ext == ".xlsx" || strings.HasPrefix(ct, "application/zip"):
		b, rerr := io.ReadAll(rest)
		if rerr != nil {
			return rerr
		}
		labels, err = readXLSX(b)
	case ext == ".xls" || bytes.HasPrefix(head, []byte{0xD0, 0xCF, 0x11, 0xE0}): // OLE Compound File
		b, rerr := io.ReadAll(rest)
		if rerr != nil {
			return rerr
		}
		labels, err = readXLS(b)
	default:
		labels, err = readCSV(rest)
	}
	if err != nil {
		return err
	}

	// Normalize, upsert, and link to batch in chunks
	var links []dbpkg.BatchLabelLink
	const chunk = 2000
	pos := 0
	seen := make(map[string]struct{}, len(labels)) // in-file dedupe on normalized ascii
	for _, raw := range labels {
		pos++
		in := strings.TrimSpace(raw)
		if in == "" {
			continue
		}
		// Extract first label if a domain
		label := normalize.ExtractFirstLabel(in)
		if label == "" {
			continue
		}
		ascii, unicode, nerr := normalize.NormalizeInput(label)
		if nerr != nil {
			continue
		}
		if _, ok := seen[ascii]; ok { // skip duplicate
			continue
		}
		seen[ascii] = struct{}{}
		lrec, err := lr.UpsertByASCII(ctx, ascii, unicode)
		if err != nil {
			return err
		}
		p := pos
		links = append(links, dbpkg.BatchLabelLink{LabelID: lrec.ID, Pos: &p})
		if len(links) >= chunk {
			if _, err := br.LinkLabelsCopy(ctx, j.BatchID, links); err != nil {
				return err
			}
			links = links[:0]
		}
	}
	if len(links) > 0 {
		if _, err := br.LinkLabelsCopy(ctx, j.BatchID, links); err != nil {
			return err
		}
	}
	return jr.UpdateStatus(ctx, j.ID, "done", nil, nil)
}

func readCSV(r io.Reader) ([]string, error) {
	br := bufio.NewReader(r)
	sample, _ := br.Peek(4096)
	delim := detectDelimiter(sample)
	cr := csv.NewReader(br)
	cr.Comma = delim
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true
	cr.TrimLeadingSpace = true

	out := make([]string, 0, 1024)
	first := true
	for {
		rec, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			// Fallback for very dirty inputs
			return readLinesFallback(br)
		}
		if len(rec) == 0 {
			continue
		}
		v := strings.TrimSpace(rec[0])
		if v == "" {
			continue
		}
		if first && looksLikeHeader(v) {
			first = false
			continue
		}
		first = false
		out = append(out, v)
	}
	return out, nil
}

func detectDelimiter(b []byte) rune {
	cComma := bytes.Count(b, []byte{','})
	cTab := bytes.Count(b, []byte{'\t'})
	cSemi := bytes.Count(b, []byte{';'})
	if cTab > cComma && cTab > cSemi {
		return '\t'
	}
	if cSemi > cComma {
		return ';'
	}
	return ','
}

func looksLikeHeader(s string) bool {
	lower := strings.ToLower(s)
	for _, h := range []string{"domain", "name", "label", "host"} {
		if strings.Contains(lower, h) {
			return true
		}
	}
	if strings.ContainsRune(s, ' ') && !strings.ContainsRune(s, '.') {
		return true
	}
	return false
}

func readLinesFallback(r io.Reader) ([]string, error) {
	s := bufio.NewScanner(r)
	out := make([]string, 0, 1024)
	for s.Scan() {
		v := strings.TrimSpace(s.Text())
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func readXLSX(b []byte) ([]string, error) {
	f, err := excelize.OpenReader(bytesReader{b: b})
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, nil
	}
	sh := sheets[0]
	rows, err := f.Rows(sh)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0, 1024)
	for rows.Next() {
		cols, err := rows.Columns()
		if err != nil {
			return nil, err
		}
		if len(cols) == 0 {
			continue
		}
		out = append(out, cols[0])
	}
	return out, rows.Error()
}

func readXLS(b []byte) ([]string, error) {
	wb, err := xls.OpenReader(bytes.NewReader(b), "utf-8")
	if err != nil {
		return nil, err
	}
	if wb.NumSheets() == 0 {
		return nil, nil
	}
	sh := wb.GetSheet(0)
	if sh == nil {
		return nil, nil
	}
	out := make([]string, 0, sh.MaxRow)
	for i := 0; i <= int(sh.MaxRow); i++ {
		row := sh.Row(i)
		if row == nil {
			continue
		}
		v := strings.TrimSpace(row.Col(0))
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out, nil
}

// bytesReader is a minimal io.ReaderAt and io.Reader for excelize
type bytesReader struct{ b []byte }

func (r bytesReader) Read(p []byte) (int, error) {
	n := copy(p, r.b)
	r.b = r.b[n:]
	if n == 0 {
		return 0, io.EOF
	}
	return n, nil
}

func (r bytesReader) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(r.b)) {
		return 0, io.EOF
	}
	n := copy(p, r.b[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

// utility (unused currently) so lints don't flag filepath import
func _keep(_ string) { _ = filepath.Base("x") }
