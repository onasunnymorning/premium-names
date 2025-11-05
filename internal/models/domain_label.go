package models

import (
    "time"

    "github.com/yourorg/zone-names/internal/domain"
    "gorm.io/gorm"
)

type DomainLabel struct {
    ID          uint           `json:"id" gorm:"primaryKey"`
    Label       string         `json:"label" gorm:"uniqueIndex;not null"`
    Original    string         `json:"original,omitempty" gorm:"-"`
    Tags        []Tag          `json:"tags,omitempty" gorm:"many2many:domain_label_tags;"`
    CreatedBy   string         `json:"created_by" gorm:"not null"`
    CreatedAt   time.Time      `json:"created_at"`
    UpdatedAt   time.Time      `json:"updated_at"`
    DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (d *DomainLabel) BeforeCreate(tx *gorm.DB) error {
    normalized, err := domain.NormalizeLabel(d.Label)
    if err != nil {
        return err
    }
    d.Label = normalized
    return nil
}

type Tag struct {
    ID          uint       `json:"id" gorm:"primaryKey"`
    Name        string     `json:"name" gorm:"uniqueIndex;not null"`
    Description string     `json:"description,omitempty"`
    Color       string     `json:"color,omitempty" gorm:"size:7"`
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
    DeletedAt   *time.Time `json:"-" gorm:"index"`
}

type DomainLabelTag struct {
    DomainLabelID uint `gorm:"primaryKey"`
    TagID         uint `gorm:"primaryKey"`
    CreatedAt     time.Time
}
