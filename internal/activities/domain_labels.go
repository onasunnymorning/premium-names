package activities

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
	"go.temporal.io/sdk/activity"
	"gorm.io/gorm"

	"github.com/yourorg/zone-names/internal/db"
	"github.com/yourorg/zone-names/internal/domain"
	iopkg "github.com/yourorg/zone-names/internal/iopkg"
	"github.com/yourorg/zone-names/internal/models"
	"github.com/yourorg/zone-names/internal/types"
)

// ParseDomainLabelFile parses uploaded files and extracts domain labels from the first column
func (a *Activities) ParseDomainLabelFile(ctx context.Context, params types.DomainLabelWorkflowParams) (types.DomainLabelProcessResult, error) {
	activity.GetLogger(ctx).Info("Starting to parse domain label file", "fileURI", params.FileURI)

	// Open the file from URI (supports file:// and s3://)
	rc, _, err := iopkg.Open(params.FileURI)
	if err != nil {
		return types.DomainLabelProcessResult{}, fmt.Errorf("failed to open file %s: %w", params.FileURI, err)
	}
	defer rc.Close()

	// Determine file type from extension
	var labels []string
	fileName := strings.ToLower(filepath.Base(params.FileURI))

	switch {
	case strings.HasSuffix(fileName, ".csv"):
		labels, err = parseCSVFromReader(rc)
	case strings.HasSuffix(fileName, ".tsv"):
		labels, err = parseTSVFromReader(rc)
	case strings.HasSuffix(fileName, ".xlsx") || strings.HasSuffix(fileName, ".xls"):
		labels, err = parseExcelFromReader(rc)
	default:
		// Default to plain text parsing (one domain per line)
		activity.GetLogger(ctx).Info("File extension not recognized, attempting plain text parsing", "fileName", fileName)
		labels, err = parsePlainTextFromReader(rc)
	}

	if err != nil {
		return types.DomainLabelProcessResult{}, fmt.Errorf("failed to parse file: %w", err)
	}

	// Process and normalize labels
	var processedLabels []types.ProcessedDomainLabel
	var errors []string
	seen := make(map[string]bool)

	for i, rawLabel := range labels {
		// Skip empty lines
		rawLabel = strings.TrimSpace(rawLabel)
		if rawLabel == "" {
			continue
		}

		// Skip header-like entries (common headers to ignore)
		if i == 0 && isLikelyHeader(rawLabel) {
			continue
		}

		// Normalize the domain label
		normalizedLabel, err := domain.NormalizeLabel(rawLabel)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Invalid label '%s': %s", rawLabel, err.Error()))
			continue
		}

		// Skip duplicates within this file
		if seen[normalizedLabel] {
			continue
		}
		seen[normalizedLabel] = true

		processedLabels = append(processedLabels, types.ProcessedDomainLabel{
			Label:    normalizedLabel,
			Original: rawLabel,
			Tags:     params.Tags,
		})

		// Heartbeat periodically for large files
		if len(processedLabels)%1000 == 0 {
			activity.RecordHeartbeat(ctx, fmt.Sprintf("Processed %d labels", len(processedLabels)))
		}
	}

	activity.GetLogger(ctx).Info("Completed parsing domain label file",
		"processed", len(processedLabels),
		"errors", len(errors),
		"total_lines", len(labels))

	return types.DomainLabelProcessResult{
		ProcessedCount: len(labels),
		SavedCount:     0, // Will be set in the save activity
		SkippedCount:   len(labels) - len(processedLabels) - len(errors),
		ErrorCount:     len(errors),
		Labels:         processedLabels,
		Errors:         errors,
	}, nil
}

// SaveDomainLabels saves the parsed domain labels to the database with tags
func (a *Activities) SaveDomainLabels(ctx context.Context, params types.DomainLabelWorkflowParams, parseResult types.DomainLabelProcessResult) (types.DomainLabelProcessResult, error) {
	activity.GetLogger(ctx).Info("Starting to save domain labels to database", "labelCount", len(parseResult.Labels))

	// Connect to the database
	dbConfig := db.Config{
		Host:     getEnvOrDefault("DB_HOST", "localhost"),
		Port:     getEnvOrDefault("DB_PORT", "5432"),
		User:     getEnvOrDefault("DB_USER", "postgres"),
		Password: getEnvOrDefault("DB_PASSWORD", "postgres"),
		DBName:   getEnvOrDefault("DB_NAME", "domain_labels"),
		SSLMode:  getEnvOrDefault("DB_SSLMODE", "disable"),
	}

	database, err := db.NewDatabase(dbConfig)
	if err != nil {
		return types.DomainLabelProcessResult{}, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer database.Close()

	// Start a database transaction
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create or get tags
	var tags []models.Tag
	for _, tagName := range params.Tags {
		var tag models.Tag
		if err := tx.Where("name = ?", tagName).FirstOrCreate(&tag, models.Tag{
			Name: tagName,
		}).Error; err != nil {
			tx.Rollback()
			return types.DomainLabelProcessResult{}, fmt.Errorf("failed to create/get tag %s: %w", tagName, err)
		}
		tags = append(tags, tag)
	}

	// Save domain labels
	var savedLabels []types.ProcessedDomainLabel
	var additionalErrors []string
	savedCount := 0

	for i, label := range parseResult.Labels {
		// Check if label already exists
		var existing models.DomainLabel
		err := tx.Where("label = ?", label.Label).First(&existing).Error

		if err == nil {
			// Label exists, add tags to it
			if err := tx.Model(&existing).Association("Tags").Append(tags); err != nil {
				additionalErrors = append(additionalErrors, fmt.Sprintf("Failed to add tags to existing label %s: %s", label.Label, err.Error()))
				continue
			}

			// Load the updated tags for response
			var tagNames []string
			for _, tag := range tags {
				tagNames = append(tagNames, tag.Name)
			}

			savedLabels = append(savedLabels, types.ProcessedDomainLabel{
				ID:       existing.ID,
				Label:    existing.Label,
				Original: label.Original,
				Tags:     tagNames,
				Created:  false,
			})
		} else if err == gorm.ErrRecordNotFound {
			// Create new label
			newLabel := models.DomainLabel{
				Label:     label.Label,
				Tags:      tags,
				CreatedBy: params.CreatedBy,
				CreatedAt: time.Now(),
			}

			if err := tx.Create(&newLabel).Error; err != nil {
				additionalErrors = append(additionalErrors, fmt.Sprintf("Failed to create label %s: %s", label.Label, err.Error()))
				continue
			}

			var tagNames []string
			for _, tag := range tags {
				tagNames = append(tagNames, tag.Name)
			}

			savedLabels = append(savedLabels, types.ProcessedDomainLabel{
				ID:       newLabel.ID,
				Label:    newLabel.Label,
				Original: label.Original,
				Tags:     tagNames,
				Created:  true,
			})
			savedCount++
		} else {
			additionalErrors = append(additionalErrors, fmt.Sprintf("Database error for label %s: %s", label.Label, err.Error()))
			continue
		}

		// Heartbeat for large batches
		if i%500 == 0 {
			activity.RecordHeartbeat(ctx, fmt.Sprintf("Saved %d/%d labels", i+1, len(parseResult.Labels)))
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return types.DomainLabelProcessResult{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	activity.GetLogger(ctx).Info("Completed saving domain labels",
		"saved", len(savedLabels),
		"additional_errors", len(additionalErrors))

	// Merge original errors with save errors
	allErrors := append(parseResult.Errors, additionalErrors...)

	return types.DomainLabelProcessResult{
		ProcessedCount: parseResult.ProcessedCount,
		SavedCount:     len(savedLabels),
		SkippedCount:   parseResult.ProcessedCount - len(savedLabels) - len(allErrors),
		ErrorCount:     len(allErrors),
		Labels:         savedLabels,
		Errors:         allErrors,
	}, nil
}

// Helper functions for parsing different file types

func parseCSVFromReader(r io.Reader) ([]string, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	var labels []string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if len(record) > 0 {
			labels = append(labels, record[0])
		}
	}
	return labels, nil
}

func parseTSVFromReader(r io.Reader) ([]string, error) {
	var labels []string
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "\t")
		if len(parts) > 0 {
			labels = append(labels, parts[0])
		}
	}

	return labels, scanner.Err()
}

func parseExcelFromReader(r io.Reader) ([]string, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var labels []string

	// Process all sheets
	for _, sheetName := range f.GetSheetList() {
		rows, err := f.GetRows(sheetName)
		if err != nil {
			continue // Skip sheets with errors
		}

		for _, row := range rows {
			if len(row) > 0 {
				labels = append(labels, row[0])
			}
		}
	}

	return labels, nil
}

// isLikelyHeader checks if a string looks like a common header
func isLikelyHeader(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	headers := []string{
		"domain", "label", "name", "hostname",
		"domain_name", "domain_label", "site", "url",
		"website", "host", "subdomain", "fqdn",
	}

	for _, header := range headers {
		if s == header {
			return true
		}
	}
	return false
}

// parsePlainTextFromReader parses plain text files with one domain per line
func parsePlainTextFromReader(r io.Reader) ([]string, error) {
	var labels []string
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			labels = append(labels, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return labels, nil
}

// getEnvOrDefault gets environment variable or returns default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
