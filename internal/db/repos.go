package db

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
)

// Label represents a record in the label table.
type Label struct {
	ID           int64
	LabelASCII   string
	LabelUnicode string
	CreatedAt    time.Time
}

// Tag represents a record in the tag table.
type Tag struct {
	ID        int64
	Name      string
	GroupName *string
	CreatedAt time.Time
}

// Batch represents an upload batch.
type Batch struct {
	ID             int64
	Name           string
	SourceFilename *string
	CreatedBy      *string
	CreatedAt      time.Time
}

// LabelRepository offers CRUD and listing with filters.
type LabelRepository interface {
	Get(ctx context.Context, id int64) (Label, error)
	// UpsertByASCII creates or updates a label by its ASCII normalized value.
	UpsertByASCII(ctx context.Context, ascii, unicode string) (Label, error)
	// List returns labels filtered by tags (ANY/ALL) and optional batch, with pagination.
	List(ctx context.Context, f LabelListFilter) ([]Label, error)
}

// TagRepository handles tag CRUD.
type TagRepository interface {
	Create(ctx context.Context, name string, groupName *string) (Tag, error)
	Get(ctx context.Context, id int64) (Tag, error)
	// FindByPrefix returns tags whose name starts with the given prefix (case-insensitive due to citext).
	FindByPrefix(ctx context.Context, prefix string, limit int) ([]Tag, error)
	// List returns tags optionally filtered by group and paginated.
	List(ctx context.Context, groupName *string, limit, offset int) ([]Tag, error)
	// Delete removes a tag by id.
	Delete(ctx context.Context, id int64) error
	// Rename updates a tag's name and optionally its group_name, returning the updated tag.
	Rename(ctx context.Context, id int64, newName string, newGroupName *string) (Tag, error)
}

// BatchRepository handles batches.
type BatchRepository interface {
	Create(ctx context.Context, name string, sourceFilename, createdBy *string) (Batch, error)
	Get(ctx context.Context, id int64) (Batch, error)
	// LinkLabelsCopy bulk inserts membership rows into batch_label using COPY for speed.
	LinkLabelsCopy(ctx context.Context, batchID int64, rows []BatchLabelLink) (int64, error)
}

// NewLabelRepo returns a repository bound to the pool.
func NewLabelRepo(p *Pool) LabelRepository { return &labelRepo{p: p} }

// NewTagRepo returns a repository bound to the pool.
func NewTagRepo(p *Pool) TagRepository { return &tagRepo{p: p} }

// NewBatchRepo returns a repository bound to the pool.
func NewBatchRepo(p *Pool) BatchRepository { return &batchRepo{p: p} }

type labelRepo struct{ p *Pool }
type tagRepo struct{ p *Pool }
type batchRepo struct{ p *Pool }

func (r *labelRepo) Get(ctx context.Context, id int64) (Label, error) {
	const q = `select id, label_ascii, label_unicode, created_at from label where id=$1`
	var l Label
	err := r.p.QueryRow(ctx, q, id).Scan(&l.ID, &l.LabelASCII, &l.LabelUnicode, &l.CreatedAt)
	if err != nil {
		return Label{}, mapRowErr(err)
	}
	return l, nil
}

func (r *labelRepo) UpsertByASCII(ctx context.Context, ascii, unicode string) (Label, error) {
	const q = `
insert into label (label_ascii, label_unicode)
values ($1, $2)
on conflict (label_ascii)
do update set label_unicode = excluded.label_unicode
returning id, label_ascii, label_unicode, created_at`
	var l Label
	err := r.p.QueryRow(ctx, q, ascii, unicode).Scan(&l.ID, &l.LabelASCII, &l.LabelUnicode, &l.CreatedAt)
	if err != nil {
		return Label{}, mapPgErr(err)
	}
	return l, nil
}

func (r *tagRepo) Create(ctx context.Context, name string, groupName *string) (Tag, error) {
	const q = `insert into tag (name, group_name) values ($1, $2) returning id, name, group_name, created_at`
	var t Tag
	err := r.p.QueryRow(ctx, q, name, groupName).Scan(&t.ID, &t.Name, &t.GroupName, &t.CreatedAt)
	if err != nil {
		return Tag{}, mapPgErr(err)
	}
	return t, nil
}

func (r *tagRepo) Get(ctx context.Context, id int64) (Tag, error) {
	const q = `select id, name, group_name, created_at from tag where id=$1`
	var t Tag
	err := r.p.QueryRow(ctx, q, id).Scan(&t.ID, &t.Name, &t.GroupName, &t.CreatedAt)
	if err != nil {
		return Tag{}, mapRowErr(err)
	}
	return t, nil
}

func (r *tagRepo) FindByPrefix(ctx context.Context, prefix string, limit int) ([]Tag, error) {
	if limit <= 0 || limit > 1000 {
		limit = 20
	}
	// citext makes ILIKE mostly redundant; however, prefix index not available; use ILIKE prefix
	const base = `select id, name, group_name, created_at from tag where name ILIKE $1 || '%' order by name asc limit $2`
	rows, err := r.p.Query(ctx, base, prefix, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.GroupName, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *tagRepo) List(ctx context.Context, groupName *string, limit, offset int) ([]Tag, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	var (
		sb   strings.Builder
		args []any
	)
	sb.WriteString("select id, name, group_name, created_at from tag ")
	if groupName != nil && *groupName != "" {
		sb.WriteString("where group_name = $1 ")
		args = append(args, *groupName)
	}
	if len(args) == 0 {
		sb.WriteString("order by name asc limit $1 offset $2")
		args = append(args, limit, offset)
	} else {
		sb.WriteString("order by name asc limit $2 offset $3")
		args = append(args, limit, offset)
	}
	rows, err := r.p.Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.GroupName, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *tagRepo) Delete(ctx context.Context, id int64) error {
	const q = `delete from tag where id=$1`
	ct, err := r.p.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *tagRepo) Rename(ctx context.Context, id int64, newName string, newGroupName *string) (Tag, error) {
	const q = `update tag set name=$1, group_name=$2 where id=$3 returning id, name, group_name, created_at`
	var t Tag
	if err := r.p.QueryRow(ctx, q, newName, newGroupName, id).Scan(&t.ID, &t.Name, &t.GroupName, &t.CreatedAt); err != nil {
		return Tag{}, mapPgErr(err)
	}
	return t, nil
}

func (r *batchRepo) Create(ctx context.Context, name string, sourceFilename, createdBy *string) (Batch, error) {
	const q = `insert into batch (name, source_filename, created_by) values ($1,$2,$3) returning id, name, source_filename, created_by, created_at`
	var b Batch
	err := r.p.QueryRow(ctx, q, name, sourceFilename, createdBy).Scan(&b.ID, &b.Name, &b.SourceFilename, &b.CreatedBy, &b.CreatedAt)
	if err != nil {
		return Batch{}, mapPgErr(err)
	}
	return b, nil
}

func (r *batchRepo) Get(ctx context.Context, id int64) (Batch, error) {
	const q = `select id, name, source_filename, created_by, created_at from batch where id=$1`
	var b Batch
	err := r.p.QueryRow(ctx, q, id).Scan(&b.ID, &b.Name, &b.SourceFilename, &b.CreatedBy, &b.CreatedAt)
	if err != nil {
		return Batch{}, mapRowErr(err)
	}
	return b, nil
}

// BatchLabelLink represents a membership row in batch_label.
type BatchLabelLink struct {
	LabelID int64
	Pos     *int
	Meta    []byte // JSON document (optional); if nil, will use '{}'
}

// LinkLabelsCopy performs a COPY into batch_label for the given batch.
func (r *batchRepo) LinkLabelsCopy(ctx context.Context, batchID int64, rows []BatchLabelLink) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	// Prepare rows for COPY
	vals := make([][]any, 0, len(rows))
	for _, row := range rows {
		var meta any
		if len(row.Meta) == 0 {
			meta = "{}"
		} else {
			meta = string(row.Meta)
		}
		var pos any
		if row.Pos != nil {
			pos = *row.Pos
		} else {
			pos = nil
		}
		vals = append(vals, []any{batchID, row.LabelID, pos, meta})
	}
	ct, err := r.p.CopyFrom(ctx,
		pgx.Identifier{"batch_label"},
		[]string{"batch_id", "label_id", "pos", "meta"},
		pgx.CopyFromRows(vals),
	)
	if err != nil {
		return 0, err
	}
	return ct, nil
}

// ---- Label-Tag linking operations ----

// LabelTagRepository encapsulates bulk tag add/remove operations.
type LabelTagRepository interface {
	AddTagToLabels(ctx context.Context, tagID int64, labelIDs []int64, addedBy *string) (int64, error)
	RemoveTagFromLabels(ctx context.Context, tagID int64, labelIDs []int64) (int64, error)
	AddTagToFilter(ctx context.Context, tagID int64, f LabelListFilter, addedBy *string) (int64, error)
}

func NewLabelTagRepo(p *Pool) LabelTagRepository { return &labelTagRepo{p: p} }

type labelTagRepo struct{ p *Pool }

// AddTagToLabels inserts (label_id, tag_id) pairs, ignoring duplicates; returns number of new rows.
func (r *labelTagRepo) AddTagToLabels(ctx context.Context, tagID int64, labelIDs []int64, addedBy *string) (int64, error) {
	if len(labelIDs) == 0 {
		return 0, nil
	}
	q := `insert into label_tag (label_id, tag_id, added_by)
		  select x, $1, $2 from unnest($3::bigint[]) as x
		  on conflict do nothing`
	ct, err := r.p.Exec(ctx, q, tagID, addedBy, labelIDs)
	if err != nil {
		return 0, mapPgErr(err)
	}
	return ct.RowsAffected(), nil
}

func (r *labelTagRepo) RemoveTagFromLabels(ctx context.Context, tagID int64, labelIDs []int64) (int64, error) {
	if len(labelIDs) == 0 {
		return 0, nil
	}
	q := `delete from label_tag where tag_id=$1 and label_id = any($2)`
	ct, err := r.p.Exec(ctx, q, tagID, labelIDs)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

// AddTagToFilter applies a tag to the set of labels matching the filter.
// If f.Limit <= 0, it will apply to the entire filtered set; otherwise limited subset.
func (r *labelTagRepo) AddTagToFilter(ctx context.Context, tagID int64, f LabelListFilter, addedBy *string) (int64, error) {
	sql, args := buildLabelIDsQuery(f)
	sb := strings.Builder{}
	sb.WriteString("insert into label_tag (label_id, tag_id, added_by) ")
	sb.WriteString("select id, $")
	sb.WriteString(fmt.Sprint(len(args) + 1))
	sb.WriteString(", $")
	sb.WriteString(fmt.Sprint(len(args) + 2))
	sb.WriteString(" from (")
	sb.WriteString(sql)
	sb.WriteString(") ids on conflict do nothing")
	args = append(args, tagID, addedBy)
	ct, err := r.p.Exec(ctx, sb.String(), args...)
	if err != nil {
		return 0, err
	}
	return ct.RowsAffected(), nil
}

// buildLabelIDsQuery builds a SELECT id ... query aligned with LabelListFilter semantics.
func buildLabelIDsQuery(f LabelListFilter) (string, []any) {
	mode := strings.ToLower(f.Mode)
	if mode != "all" {
		mode = "any"
	}
	limit := f.Limit
	offset := f.Offset
	applyLimit := limit > 0 // if <=0, apply to the entire filtered set
	if offset < 0 {
		offset = 0
	}
	var (
		sb   strings.Builder
		args []any
		argN = 1
	)
	if len(f.Tags) == 0 {
		sb.WriteString("select l.id from label l ")
		if f.Batch != nil {
			sb.WriteString("join batch_label bl on bl.label_id = l.id and bl.batch_id = $")
			sb.WriteString(strconv.Itoa(argN))
			args = append(args, *f.Batch)
			argN++
		}
		sb.WriteString(" order by l.id asc")
		if applyLimit {
			sb.WriteString(" limit $")
			sb.WriteString(strconv.Itoa(argN))
			args = append(args, limit)
			argN++
			sb.WriteString(" offset $")
			sb.WriteString(strconv.Itoa(argN))
			args = append(args, offset)
		}
		return sb.String(), args
	}
	if mode == "any" {
		sb.WriteString("select l.id from label l ")
		sb.WriteString("join label_tag lt on lt.label_id = l.id ")
		sb.WriteString("join tag t on t.id = lt.tag_id ")
		if f.Batch != nil {
			sb.WriteString("join batch_label bl on bl.label_id = l.id and bl.batch_id = $")
			sb.WriteString(strconv.Itoa(argN))
			args = append(args, *f.Batch)
			argN++
		}
		sb.WriteString("where t.name = any($")
		sb.WriteString(strconv.Itoa(argN))
		sb.WriteString(") group by l.id order by l.id asc")
		args = append(args, f.Tags)
		argN++
		if applyLimit {
			sb.WriteString(" limit $")
			sb.WriteString(strconv.Itoa(argN))
			args = append(args, limit)
			argN++
			sb.WriteString(" offset $")
			sb.WriteString(strconv.Itoa(argN))
			args = append(args, offset)
		}
		return sb.String(), args
	}
	// all
	sb.WriteString("with wanted as (select id from tag where name = any($")
	sb.WriteString(strconv.Itoa(argN))
	sb.WriteString(")) ")
	args = append(args, f.Tags)
	argN++
	sb.WriteString("select l.id from label l ")
	sb.WriteString("join label_tag lt on lt.label_id = l.id ")
	if f.Batch != nil {
		sb.WriteString("join batch_label bl on bl.label_id = l.id and bl.batch_id = $")
		sb.WriteString(strconv.Itoa(argN))
		args = append(args, *f.Batch)
		argN++
	}
	sb.WriteString("where lt.tag_id in (select id from wanted) group by l.id having count(distinct lt.tag_id) = (select count(*) from wanted) order by l.id asc")
	if applyLimit {
		sb.WriteString(" limit $")
		sb.WriteString(strconv.Itoa(argN))
		args = append(args, limit)
		argN++
		sb.WriteString(" offset $")
		sb.WriteString(strconv.Itoa(argN))
		args = append(args, offset)
	}
	return sb.String(), args
}

// LabelListFilter captures filtering semantics for listing labels.
type LabelListFilter struct {
	Tags   []string // tag names; case-insensitive due to citext equality
	Mode   string   // "any" (default) or "all"
	Batch  *int64   // optional batch_id to scope membership
	Limit  int
	Offset int
}

func (r *labelRepo) List(ctx context.Context, f LabelListFilter) ([]Label, error) {
	mode := strings.ToLower(f.Mode)
	if mode != "all" {
		mode = "any"
	}
	limit := f.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	var (
		sb   strings.Builder
		args []any
		argN = 1
	)
	// Base select
	if len(f.Tags) == 0 {
		sb.WriteString("select l.id, l.label_ascii, l.label_unicode, l.created_at from label l ")
		if f.Batch != nil {
			sb.WriteString("join batch_label bl on bl.label_id = l.id and bl.batch_id = $")
			sb.WriteString(strconv.Itoa(argN))
			args = append(args, *f.Batch)
			argN++
		}
		sb.WriteString(" order by l.id asc limit $")
		sb.WriteString(strconv.Itoa(argN))
		args = append(args, limit)
		argN++
		sb.WriteString(" offset $")
		sb.WriteString(strconv.Itoa(argN))
		args = append(args, offset)
		// Run
		rows, err := r.p.Query(ctx, sb.String(), args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var out []Label
		for rows.Next() {
			var l Label
			if err := rows.Scan(&l.ID, &l.LabelASCII, &l.LabelUnicode, &l.CreatedAt); err != nil {
				return nil, err
			}
			out = append(out, l)
		}
		return out, rows.Err()
	}

	// With tags
	if mode == "any" {
		// ANY: union-style
		sb.WriteString("select l.id, l.label_ascii, l.label_unicode, l.created_at from label l ")
		sb.WriteString("join label_tag lt on lt.label_id = l.id ")
		sb.WriteString("join tag t on t.id = lt.tag_id ")
		if f.Batch != nil {
			sb.WriteString("join batch_label bl on bl.label_id = l.id and bl.batch_id = $")
			sb.WriteString(strconv.Itoa(argN))
			args = append(args, *f.Batch)
			argN++
		}
		sb.WriteString("where t.name = any($")
		sb.WriteString(strconv.Itoa(argN))
		sb.WriteString(") group by l.id order by l.id asc limit $")
		args = append(args, f.Tags)
		argN++
		sb.WriteString(strconv.Itoa(argN))
		args = append(args, limit)
		argN++
		sb.WriteString(" offset $")
		sb.WriteString(strconv.Itoa(argN))
		args = append(args, offset)
		rows, err := r.p.Query(ctx, sb.String(), args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var out []Label
		for rows.Next() {
			var l Label
			if err := rows.Scan(&l.ID, &l.LabelASCII, &l.LabelUnicode, &l.CreatedAt); err != nil {
				return nil, err
			}
			out = append(out, l)
		}
		return out, rows.Err()
	}

	// ALL: intersection-style
	sb.WriteString("with wanted as (select id from tag where name = any($")
	sb.WriteString(strconv.Itoa(argN))
	sb.WriteString(")) ")
	args = append(args, f.Tags)
	argN++
	sb.WriteString("select l.id, l.label_ascii, l.label_unicode, l.created_at from label l ")
	sb.WriteString("join label_tag lt on lt.label_id = l.id ")
	if f.Batch != nil {
		sb.WriteString("join batch_label bl on bl.label_id = l.id and bl.batch_id = $")
		sb.WriteString(strconv.Itoa(argN))
		args = append(args, *f.Batch)
		argN++
	}
	sb.WriteString("where lt.tag_id in (select id from wanted) group by l.id having count(distinct lt.tag_id) = (select count(*) from wanted) order by l.id asc limit $")
	sb.WriteString(strconv.Itoa(argN))
	args = append(args, limit)
	argN++
	sb.WriteString(" offset $")
	sb.WriteString(strconv.Itoa(argN))
	args = append(args, offset)

	rows, err := r.p.Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Label
	for rows.Next() {
		var l Label
		if err := rows.Scan(&l.ID, &l.LabelASCII, &l.LabelUnicode, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// mapPgErr maps common pg errors to friendly domain errors
func mapPgErr(err error) error {
	if err == nil {
		return nil
	}
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		switch pe.Code {
		case "23505": // unique_violation
			return ErrConflict
		}
	}
	return err
}

// mapRowErr translates not found cases to ErrNotFound
func mapRowErr(err error) error {
	if err == nil {
		return nil
	}
	// pgx returns no rows as a sentinel error
	if errors.Is(err, ErrNotFound) { // already mapped
		return err
	}
	// We can't import pgx directly just to check ErrNoRows; use string compare fallback if needed
	if err.Error() == "no rows in result set" {
		return ErrNotFound
	}
	return err
}
