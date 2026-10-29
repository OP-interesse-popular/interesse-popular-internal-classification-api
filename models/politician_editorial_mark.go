package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PoliticianEditorialMark is a manual, evidence-backed classification used for cards.
type PoliticianEditorialMark struct {
	ID                 string     `gorm:"type:uuid;primaryKey" json:"id"`
	PoliticianID       string     `gorm:"type:uuid;not null;uniqueIndex" json:"politicianId"`
	Direction          string     `gorm:"size:20;not null" json:"direction"`
	Ideology           string     `gorm:"size:40;not null" json:"ideology"`
	CardEnabled        bool       `gorm:"not null;default:false" json:"cardEnabled"`
	MethodologyVersion string     `gorm:"size:100;not null" json:"methodologyVersion"`
	SourceName         string     `gorm:"size:255;not null" json:"sourceName"`
	SourceURL          string     `gorm:"size:2048" json:"sourceUrl,omitempty"`
	EvidenceNote       string     `gorm:"type:text;not null" json:"evidenceNote"`
	ValidFrom          time.Time  `gorm:"not null" json:"validFrom"`
	ValidTo            *time.Time `json:"validTo,omitempty"`
	ReviewedAt         *time.Time `json:"reviewedAt,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

func (PoliticianEditorialMark) TableName() string { return "politician_editorial_marks" }
func (p *PoliticianEditorialMark) BeforeCreate(_ *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	return nil
}
