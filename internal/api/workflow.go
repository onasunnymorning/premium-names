package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.temporal.io/sdk/client"

	"github.com/yourorg/zone-names/internal/models"
	"github.com/yourorg/zone-names/internal/types"
	"gorm.io/gorm"
)

type WorkflowHandler struct {
	db             *gorm.DB
	temporalClient client.Client
}

func NewWorkflowHandler(db *gorm.DB, temporalClient client.Client) *WorkflowHandler {
	return &WorkflowHandler{
		db:             db,
		temporalClient: temporalClient,
	}
}

type StartWorkflowRequest struct {
	FileURI     string   `json:"file_uri" binding:"required"`
	Tags        []string `json:"tags" binding:"required"`
	CreatedBy   string   `json:"created_by" binding:"required"`
	Description string   `json:"description"`
}

type StartWorkflowResponse struct {
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
}

// StartDomainLabelWorkflow starts a new Temporal workflow to process domain labels
func (h *WorkflowHandler) StartDomainLabelWorkflow(c *gin.Context) {
	var req StartWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Prepare workflow parameters
	params := types.DomainLabelWorkflowParams{
		FileURI:     req.FileURI,
		Tags:        req.Tags,
		CreatedBy:   req.CreatedBy,
		Description: req.Description,
	}

	// Start the workflow
	options := client.StartWorkflowOptions{
		TaskQueue: "zone-names", // Use the same task queue as the worker
	}

	workflowRun, err := h.temporalClient.ExecuteWorkflow(
		c.Request.Context(),
		options,
		"DomainLabelsWorkflow", // Must match the registered workflow name
		params,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start workflow: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, StartWorkflowResponse{
		WorkflowID: workflowRun.GetID(),
		RunID:      workflowRun.GetRunID(),
	})
}

// GetWorkflowStatus gets the status of a workflow execution
func (h *WorkflowHandler) GetWorkflowStatus(c *gin.Context) {
	workflowID := c.Param("id")
	if workflowID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Workflow ID is required"})
		return
	}

	// Get workflow execution
	workflowRun := h.temporalClient.GetWorkflow(c.Request.Context(), workflowID, "")

	// Try to get the result (this will be available when workflow completes)
	var result types.DomainLabelProcessResult
	err := workflowRun.Get(c.Request.Context(), &result)

	if err != nil {
		// Workflow is still running or failed
		describe, descErr := h.temporalClient.DescribeWorkflowExecution(
			c.Request.Context(),
			workflowID,
			"",
		)
		if descErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to describe workflow: " + descErr.Error()})
			return
		}

		status := describe.WorkflowExecutionInfo.Status.String()
		c.JSON(http.StatusOK, gin.H{
			"workflow_id": workflowID,
			"status":      status,
			"start_time":  describe.WorkflowExecutionInfo.StartTime,
		})
		return
	}

	// Workflow completed successfully
	c.JSON(http.StatusOK, gin.H{
		"workflow_id": workflowID,
		"status":      "COMPLETED",
		"result":      result,
	})
}

// Enhanced label handlers
func (h *Handler) GetLabelsWithPagination(c *gin.Context) {
	var labels []models.DomainLabel
	query := h.db.Model(&models.DomainLabel{}).Preload("Tags")

	// Apply filters
	if tag := c.Query("tag"); tag != "" {
		query = query.Joins("JOIN domain_label_tags ON domain_labels.id = domain_label_tags.domain_label_id").
			Joins("JOIN tags ON domain_label_tags.tag_id = tags.id").
			Where("tags.name = ?", tag)
	}

	if search := c.Query("q"); search != "" {
		query = query.Where("label ILIKE ?", "%"+search+"%")
	}

	if createdBy := c.Query("created_by"); createdBy != "" {
		query = query.Where("created_by ILIKE ?", "%"+createdBy+"%")
	}

	// Apply pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	// Get total count
	var total int64
	countQuery := h.db.Model(&models.DomainLabel{})
	if tag := c.Query("tag"); tag != "" {
		countQuery = countQuery.Joins("JOIN domain_label_tags ON domain_labels.id = domain_label_tags.domain_label_id").
			Joins("JOIN tags ON domain_label_tags.tag_id = tags.id").
			Where("tags.name = ?", tag)
	}
	if search := c.Query("q"); search != "" {
		countQuery = countQuery.Where("label ILIKE ?", "%"+search+"%")
	}
	if createdBy := c.Query("created_by"); createdBy != "" {
		countQuery = countQuery.Where("created_by ILIKE ?", "%"+createdBy+"%")
	}
	countQuery.Count(&total)

	// Apply ordering
	orderBy := c.DefaultQuery("order_by", "created_at")
	orderDir := c.DefaultQuery("order_dir", "desc")
	if orderDir != "asc" && orderDir != "desc" {
		orderDir = "desc"
	}

	query = query.Order(orderBy + " " + orderDir).Limit(limit).Offset(offset)

	if err := query.Find(&labels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"labels": labels,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetTags returns all available tags
func (h *Handler) GetTags(c *gin.Context) {
	var tags []models.Tag
	if err := h.db.Find(&tags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tags": tags})
}

// GetTagStats returns statistics about tags and their usage
func (h *Handler) GetTagStats(c *gin.Context) {
	type TagStat struct {
		TagID   uint   `json:"tag_id"`
		TagName string `json:"tag_name"`
		Count   int64  `json:"count"`
		Color   string `json:"color"`
	}

	var stats []TagStat
	err := h.db.Raw(`
		SELECT 
			t.id as tag_id,
			t.name as tag_name,
			t.color as color,
			COUNT(dlt.domain_label_id) as count
		FROM tags t
		LEFT JOIN domain_label_tags dlt ON t.id = dlt.tag_id
		GROUP BY t.id, t.name, t.color
		ORDER BY count DESC, t.name ASC
	`).Scan(&stats).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tag_stats": stats})
}

// DeleteLabel removes a domain label
func (h *Handler) DeleteLabel(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	// Check if label exists
	var label models.DomainLabel
	if err := h.db.First(&label, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Label not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Delete the label (soft delete)
	if err := h.db.Delete(&label).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Label deleted successfully"})
}

// UpdateLabelTags updates the tags for a domain label
func (h *Handler) UpdateLabelTags(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var req struct {
		Tags []string `json:"tags" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the label
	var label models.DomainLabel
	if err := h.db.Preload("Tags").First(&label, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Label not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Get or create new tags
	var newTags []models.Tag
	for _, tagName := range req.Tags {
		var tag models.Tag
		if err := h.db.Where("name = ?", tagName).FirstOrCreate(&tag, models.Tag{Name: tagName}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tag: " + err.Error()})
			return
		}
		newTags = append(newTags, tag)
	}

	// Replace tags
	if err := h.db.Model(&label).Association("Tags").Replace(newTags); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update tags: " + err.Error()})
		return
	}

	// Return updated label
	if err := h.db.Preload("Tags").First(&label, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, label)
}
