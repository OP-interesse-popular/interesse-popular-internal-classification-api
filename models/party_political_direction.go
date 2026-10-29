package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	DirectionLeft  = "esquerda"
	DirectionRight = "direita"

	IdeologyRadicalLeft  = "esquerda_radical"
	IdeologyLiberalLeft  = "esquerda_liberal"
	IdeologyLiberalRight = "direita_liberal"
	IdeologyFarRight     = "extrema_direita"
)

// PartyPoliticalDirection records an editorial classification, plus the evidence
// and methodology needed to audit that classification later.
type PartyPoliticalDirection struct {
	ID                 string     `gorm:"type:uuid;primaryKey" json:"id"`
	PartyTSENumber     int        `gorm:"not null;uniqueIndex" json:"partyTseNumber"`
	PartyAcronym       string     `gorm:"size:30;not null;uniqueIndex" json:"partyAcronym"`
	PartyName          string     `gorm:"size:255;not null;uniqueIndex" json:"partyName"`
	Direction          string     `gorm:"size:20;not null" json:"direction"`
	Ideology           string     `gorm:"size:40;not null" json:"ideology"`
	MethodologyVersion string     `gorm:"size:100;not null" json:"methodologyVersion"`
	SourceName         string     `gorm:"size:255;not null" json:"sourceName"`
	SourceURL          string     `gorm:"size:2048" json:"sourceUrl"`
	EvidenceNote       string     `gorm:"type:text" json:"evidenceNote"`
	ValidFrom          time.Time  `gorm:"not null" json:"validFrom"`
	ValidTo            *time.Time `json:"validTo,omitempty"`
	ReviewedAt         *time.Time `json:"reviewedAt,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

func (PartyPoliticalDirection) TableName() string { return "party_political_directions" }

func (p *PartyPoliticalDirection) BeforeCreate(_ *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	return nil
}
