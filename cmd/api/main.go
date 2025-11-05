package main

import (
	"context"
	_ "embed"
	"encoding/csv"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/zone-names/internal/db"
	"github.com/yourorg/zone-names/internal/storage"
)

func main() {
	// Setup DB
	cfg := db.FromEnv()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := db.Connect(ctx, cfg)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	r := gin.Default()

	// CORS (dev-friendly, adjust as needed)
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// Liveness
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Readiness (DB ping)
	r.GET("/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 1*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not-ready", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	// Repos
	lr := db.NewLabelRepo(pool)
	tr := db.NewTagRepo(pool)
	br := db.NewBatchRepo(pool)
	ltr := db.NewLabelTagRepo(pool)
	jr := db.NewJobRepo(pool)

	api := r.Group("/api")

	// Serve embedded OpenAPI spec at /api/openapi.yaml
	api.GET("/openapi.yaml", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/yaml", openapiSpec)
	})

	// POST /api/batches
	api.POST("/batches", func(c *gin.Context) {
		var req struct {
			Name           string  `json:"name" binding:"required"`
			SourceFilename *string `json:"source_filename"`
			CreatedBy      *string `json:"created_by"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		b, err := br.Create(c.Request.Context(), req.Name, req.SourceFilename, req.CreatedBy)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, b)
	})

	// GET /api/labels?tags=a,b&mode=any|all&batch=123&limit=100&offset=0
	// Returns labels and their tags.
	api.GET("/labels", func(c *gin.Context) {
		tagsParam := strings.TrimSpace(c.Query("tags"))
		var tags []string
		if tagsParam != "" {
			for _, t := range strings.Split(tagsParam, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					tags = append(tags, t)
				}
			}
		}
		mode := c.DefaultQuery("mode", "any")
		var batch *int64
		if b := c.Query("batch"); b != "" {
			if id, err := strconv.ParseInt(b, 10, 64); err == nil {
				batch = &id
			}
		}
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		list, err := lr.List(c.Request.Context(), db.LabelListFilter{
			Tags: tags, Mode: mode, Batch: batch, Limit: limit, Offset: offset,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// Fetch tags for returned labels in one query
		type labelWithTags struct {
			db.Label
			Tags []db.Tag
		}
		if len(list) == 0 {
			c.JSON(http.StatusOK, []labelWithTags{})
			return
		}
		ids := make([]int64, 0, len(list))
		for _, l := range list {
			ids = append(ids, l.ID)
		}
		// label_id -> tags
		m := make(map[int64][]db.Tag, len(list))
		const q = `select lt.label_id, t.id, t.name, t.group_name, t.created_at
				   from label_tag lt
				   join tag t on t.id = lt.tag_id
				   where lt.label_id = any($1)
				   order by lt.label_id asc, t.name asc`
		rows, err := pool.Query(c.Request.Context(), q, ids)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		for rows.Next() {
			var labelID int64
			var t db.Tag
			if err := rows.Scan(&labelID, &t.ID, &t.Name, &t.GroupName, &t.CreatedAt); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			m[labelID] = append(m[labelID], t)
		}
		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		out := make([]labelWithTags, 0, len(list))
		for _, l := range list {
			out = append(out, labelWithTags{Label: l, Tags: m[l.ID]})
		}
		c.JSON(http.StatusOK, out)
	})

	// POST /api/labels/:id/tags -> { "tagId": 1, "addedBy":"user" }
	api.POST("/labels/:id/tags", func(c *gin.Context) {
		labelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var req struct {
			TagID   int64   `json:"tagId" binding:"required"`
			AddedBy *string `json:"addedBy"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		_, err = ltr.AddTagToLabels(c.Request.Context(), req.TagID, []int64{labelID}, req.AddedBy)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})

	// DELETE /api/labels/:id/tags/:tagId
	api.DELETE("/labels/:id/tags/:tagId", func(c *gin.Context) {
		labelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		tagID, err := strconv.ParseInt(c.Param("tagId"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tagId"})
			return
		}
		_, err = ltr.RemoveTagFromLabels(c.Request.Context(), tagID, []int64{labelID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})

	// POST /api/labels/tags/apply -> { tagId, filter:{ tags, mode, batch, limit, offset }, addedBy }
	api.POST("/labels/tags/apply", func(c *gin.Context) {
		var req struct {
			TagID   int64              `json:"tagId" binding:"required"`
			Filter  db.LabelListFilter `json:"filter"`
			AddedBy *string            `json:"addedBy"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		n, err := ltr.AddTagToFilter(c.Request.Context(), req.TagID, req.Filter, req.AddedBy)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"added": n})
	})

	// Tags
	// GET /api/tags?prefix=ca&limit=20
	api.GET("/tags", func(c *gin.Context) {
		prefix := c.Query("prefix")
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		out, err := tr.FindByPrefix(c.Request.Context(), prefix, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, out)
	})

	// POST /api/tags -> { name, groupName }
	api.POST("/tags", func(c *gin.Context) {
		var req struct {
			Name      string  `json:"name" binding:"required"`
			GroupName *string `json:"groupName"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		t, err := tr.Create(c.Request.Context(), req.Name, req.GroupName)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, t)
	})

	// PATCH /api/tags/:id -> { name?, groupName? }
	api.PATCH("/tags/:id", func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var req struct {
			Name      *string `json:"name"`
			GroupName *string `json:"groupName"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// Preserve if nil
		current, err := tr.Get(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		newName := current.Name
		if req.Name != nil {
			newName = *req.Name
		}
		t, err := tr.Rename(c.Request.Context(), id, newName, req.GroupName)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, t)
	})

	// DELETE /api/tags/:id
	api.DELETE("/tags/:id", func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := tr.Delete(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	})

	// GET /api/export?tags=...&mode=...&batch=... => CSV
	api.GET("/export", func(c *gin.Context) {
		tagsParam := strings.TrimSpace(c.Query("tags"))
		var tags []string
		if tagsParam != "" {
			for _, t := range strings.Split(tagsParam, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					tags = append(tags, t)
				}
			}
		}
		mode := c.DefaultQuery("mode", "any")
		var batch *int64
		if b := c.Query("batch"); b != "" {
			if id, err := strconv.ParseInt(b, 10, 64); err == nil {
				batch = &id
			}
		}
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=labels.csv")
		w := csv.NewWriter(c.Writer)
		_ = w.Write([]string{"id", "label_ascii", "label_unicode", "created_at"})
		// Stream out in pages
		offset := 0
		for {
			list, err := lr.List(c.Request.Context(), db.LabelListFilter{Tags: tags, Mode: mode, Batch: batch, Limit: 1000, Offset: offset})
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if len(list) == 0 {
				break
			}
			for _, l := range list {
				_ = w.Write([]string{strconv.FormatInt(l.ID, 10), l.LabelASCII, l.LabelUnicode, l.CreatedAt.Format(time.RFC3339)})
			}
			w.Flush()
			if len(list) < 1000 {
				break
			}
			offset += len(list)
		}
	})

	// POST /api/batches/:id/upload
	// Accepts multipart/form-data with file field "file". Uploads to S3/MinIO and enqueues a job.
	api.POST("/batches/:id/upload", func(c *gin.Context) {
		batchID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid batch id"})
			return
		}
		// Parse upload
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
			return
		}
		defer file.Close()
		// Store to S3
		ctx := c.Request.Context()
		s3c, err := storage.NewS3(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		bucket := os.Getenv("IMPORT_BUCKET")
		if bucket == "" {
			bucket = "zone-names"
		}
		key := fmt.Sprintf("uploads/batch-%d/%d-%s", batchID, time.Now().Unix(), header.Filename)
		uri := fmt.Sprintf("s3://%s/%s", bucket, key)
		if _, err := s3c.Put(ctx, uri, file); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// Enqueue job
		job, err := jr.Enqueue(ctx, batchID, uri)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, job)
	})

	// Port from env or default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	r.Run(":" + port)
}

// Embed OpenAPI spec (generated/maintained in this package directory)
//
//go:embed openapi.yaml
var openapiSpec []byte
