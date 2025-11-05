package api

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
    "github.com/yourorg/zone-names/internal/models"
    "gorm.io/gorm"
)

type Handler struct {
    db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
    return &Handler{db: db}
}

func (h *Handler) GetLabels(c *gin.Context) {
    var labels []models.DomainLabel
    query := h.db.Model(&models.DomainLabel{}).Preload("Tags")

    if tag := c.Query("tag"); tag != "" {
        query = query.Joins("JOIN domain_label_tags ON domain_labels.id = domain_label_tags.domain_label_id").
            Joins("JOIN tags ON domain_label_tags.tag_id = tags.id").
            Where("tags.name = ?", tag)
    }

    if search := c.Query("q"); search != "" {
        query = query.Where("label ILIKE ?", "%"+search+"%")
    }

    if err := query.Find(&labels).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, labels)
}

func (h *Handler) GetLabel(c *gin.Context) {
    id, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
        return
    }

    var label models.DomainLabel
    if err := h.db.Preload("Tags").First(&label, id).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            c.JSON(http.StatusNotFound, gin.H{"error": "Label not found"})
            return
        }
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, label)
}
