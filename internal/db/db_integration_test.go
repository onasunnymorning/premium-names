package db_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/yourorg/zone-names/internal/db"
	"github.com/yourorg/zone-names/internal/normalize"
)

func testDSN() string {
	if dsn := os.Getenv("DB_TEST_DSN"); dsn != "" {
		return dsn
	}
	// Default to local compose
	return "postgres://temporal:temporal@localhost:5432/premium_names?sslmode=disable"
}

func connect(t *testing.T) *db.Pool {
	t.Helper()
	cfg := db.FromEnv()
	if cfg.DSN == "" {
		cfg.DSN = testDSN()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p, err := db.Connect(ctx, cfg)
	if err != nil {
		t.Skipf("skipping integration test; cannot connect to DB: %v", err)
	}
	return p
}

func mustExec(t *testing.T, p *db.Pool, sql string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := p.Exec(ctx, sql); err != nil {
		t.Fatalf("exec failed: %v", err)
	}
}

func TestBulkLabelTagAndBatchCopy(t *testing.T) {
	p := connect(t)
	defer p.Close()

	// Clean tables
	mustExec(t, p, `TRUNCATE label_tag, batch_label, tag, label, batch RESTART IDENTITY CASCADE`)

	ctx := context.Background()

	// Repos
	lr := db.NewLabelRepo(p)
	tr := db.NewTagRepo(p)
	br := db.NewBatchRepo(p)
	ltr := db.NewLabelTagRepo(p)

	// Seed labels
	inputs := []string{"Café.example", "Paris.travel", "madrid.es"}
	var labelIDs []int64
	for _, in := range inputs {
		ascii, uni, err := normalize.NormalizeInput(in)
		if err != nil {
			t.Fatalf("normalize(%q): %v", in, err)
		}
		l, err := lr.UpsertByASCII(ctx, ascii, uni)
		if err != nil {
			t.Fatalf("upsert label: %v", err)
		}
		labelIDs = append(labelIDs, l.ID)
	}

	// Create batch and link labels with positions
	b, err := br.Create(ctx, "test-batch", ptr("test.csv"), ptr("tester"))
	if err != nil {
		t.Fatalf("create batch: %v", err)
	}
	links := []db.BatchLabelLink{
		{LabelID: labelIDs[0], Pos: intPtr(1)},
		{LabelID: labelIDs[1], Pos: intPtr(2)},
		{LabelID: labelIDs[2], Pos: intPtr(3)},
	}
	if n, err := br.LinkLabelsCopy(ctx, b.ID, links); err != nil || n != int64(len(links)) {
		t.Fatalf("copy links: n=%d err=%v", n, err)
	}

	// Create tags and bulk-apply
	city, err := tr.Create(ctx, "cities", nil)
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}
	travel, err := tr.Create(ctx, "travel", nil)
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}

	// Apply 'cities' to all labels via filter ANY on batch
	added, err := ltr.AddTagToFilter(ctx, city.ID, db.LabelListFilter{Batch: &b.ID, Mode: "any"}, ptr("tester"))
	if err != nil {
		t.Fatalf("add tag to filter: %v", err)
	}
	if added != int64(len(labelIDs)) {
		t.Fatalf("expected %d added, got %d", len(labelIDs), added)
	}

	// Apply 'travel' to first two labels explicitly
	added, err = ltr.AddTagToLabels(ctx, travel.ID, labelIDs[:2], ptr("tester"))
	if err != nil {
		t.Fatalf("add tag to labels: %v", err)
	}
	if added != 2 {
		t.Fatalf("expected 2 added got %d", added)
	}

	// Query labels with ANY {cities,travel} => all 3
	ls, err := lr.List(ctx, db.LabelListFilter{Tags: []string{"cities", "travel"}, Mode: "any", Batch: &b.ID})
	if err != nil {
		t.Fatalf("list any: %v", err)
	}
	if len(ls) != 3 {
		t.Fatalf("expected 3 labels any, got %d", len(ls))
	}

	// Query labels with ALL {cities,travel} => first two
	ls, err = lr.List(ctx, db.LabelListFilter{Tags: []string{"cities", "travel"}, Mode: "all", Batch: &b.ID})
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(ls) != 2 {
		t.Fatalf("expected 2 labels all, got %d", len(ls))
	}

	// Remove 'travel' from one label and verify ALL drops to 1
	removed, err := ltr.RemoveTagFromLabels(ctx, travel.ID, []int64{labelIDs[0]})
	if err != nil {
		t.Fatalf("remove tag: %v", err)
	}
	if removed != 1 {
		t.Fatalf("expected removed=1 got %d", removed)
	}
	ls, err = lr.List(ctx, db.LabelListFilter{Tags: []string{"cities", "travel"}, Mode: "all", Batch: &b.ID})
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(ls) != 1 {
		t.Fatalf("expected 1 label all after remove, got %d", len(ls))
	}
}

func ptr[T any](v T) *T { return &v }
func intPtr(i int) *int { return &i }
