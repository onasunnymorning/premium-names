package api

import (
	"encoding/csv"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"github.com/yourorg/zone-names/internal/domain"
	"github.com/yourorg/zone-names/internal/models"
	"gorm.io/gorm"
)

type UploadHandler struct {
	db *gorm.DB
}

func NewUploadHandler(db *gorm.DB) *UploadHandler {
	return &UploadHandler{db: db}
}

type UploadRequest struct {
	Tags      string `form:"tags" binding:"required"`
	CreatedBy string `form:"created_by" binding:"required"`
}

func (h *UploadHandler) UploadFile(c *gin.Context) {
	var req UploadRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse comma-separated tags
	var tagNames []string
	if req.Tags != "" {
		for _, tag := range strings.Split(req.Tags, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				tagNames = append(tagNames, tag)
			}
		}
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File upload error: " + err.Error()})
		return
	}
	defer file.Close()

	var labels []string
	switch {
	case strings.HasSuffix(header.Filename, ".csv"):
		labels, err = parseCSV(file)
	case strings.HasSuffix(header.Filename, ".xlsx"), strings.HasSuffix(header.Filename, ".xls"):
		labels, err = parseExcel(file)
	case strings.HasSuffix(header.Filename, ".tsv"):
		labels, err = parseTSV(file)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported file type"})
		return
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file parsing error: " + err.Error()})
		return
	}

	result, err := h.processLabels(labels, tagNames, req.CreatedBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "processing error: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"processed": len(labels),
		"saved":     len(result),
		"labels":    result,
	})
}

func (h *UploadHandler) processLabels(inputs, tagNames []string, createdBy string) ([]models.DomainLabel, error) {
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var tags []models.Tag
	for _, tagName := range tagNames {
		var tag models.Tag
		if err := tx.Where("name = ?", tagName).FirstOrCreate(&tag, models.Tag{Name: tagName}).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
		tags = append(tags, tag)
	}

	var savedLabels []models.DomainLabel
	seen := make(map[string]bool)

	for _, input := range inputs {
		if input == "" {
			continue
		}

		label, err := domain.NormalizeLabel(input)
		if err != nil {
			continue
		}

		if seen[label] {
			continue
		}
		seen[label] = true

		var existing models.DomainLabel
		if err := tx.Where("label = ?", label).First(&existing).Error; err == nil {
			if err := tx.Model(&existing).Association("Tags").Append(tags); err != nil {
				tx.Rollback()
				return nil, err
			}
			savedLabels = append(savedLabels, existing)
			continue
		}

		newLabel := models.DomainLabel{
			Label:     label,
			Original:  input,
			Tags:      tags,
			CreatedBy: createdBy,
		}

		if err := tx.Create(&newLabel).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		savedLabels = append(savedLabels, newLabel)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	return savedLabels, nil
}

func parseCSV(r io.Reader) ([]string, error) {
	reader := csv.NewReader(r)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var labels []string
	for _, record := range records {
		if len(record) > 0 {
			labels = append(labels, record[0])
		}
	}
	return labels, nil
}

func parseTSV(r io.Reader) ([]string, error) {
	reader := csv.NewReader(r)
	reader.Comma = '\t'
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	var labels []string
	for _, record := range records {
		if len(record) > 0 {
			labels = append(labels, record[0])
		}
	}
	return labels, nil
}

func parseExcel(r io.Reader) ([]string, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, err
	}

	var labels []string
	for _, sheet := range f.GetSheetMap() {
		rows, err := f.GetRows(sheet)
		if err != nil {
			continue
		}
		for _, row := range rows {
			if len(row) > 0 {
				labels = append(labels, row[0])
			}
		}
	}
	return labels, nil
}
