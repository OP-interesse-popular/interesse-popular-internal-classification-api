package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	PoliticianStatusActive         = "active"
	PoliticianStatusElectedPending = "elected_pending"
	PoliticianStatusFormer         = "former"

	ExternalSourceCamara = "camara"
	ExternalSourceSenado = "senado"
	ExternalSourceTSE    = "tse"

	ExternalIdentifierParliamentarian = "parliamentarian"
	ExternalIdentifierCandidate       = "candidate"

	MatchStatusAutomatic     = "automatic"
	MatchStatusPendingReview = "pending_review"
	MatchStatusVerified      = "verified"
	MatchStatusRejected      = "rejected"
)

// Politician is the platform's stable identity, independent from a source or a mandate.
type Politician struct {
	ID             string                 `gorm:"type:uuid;primaryKey" json:"id"`
	FullName       string                 `gorm:"size:255;not null" json:"fullName"`
	NormalizedName string                 `gorm:"size:255;not null;index" json:"-"`
	ExternalIDs    []PoliticianExternalID `gorm:"foreignKey:PoliticianID" json:"externalIds,omitempty"`
	Mandates       []ParliamentaryMandate `gorm:"foreignKey:PoliticianID" json:"mandates,omitempty"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
}

// PoliticianAlias preserves an official source name (for example, a ballot name)
// so the same person can be reconciled when another source uses a shorter name.
type PoliticianAlias struct {
	ID              string    `gorm:"type:uuid;primaryKey" json:"id"`
	PoliticianID    string    `gorm:"type:uuid;not null;uniqueIndex:ux_politician_alias" json:"politicianId"`
	Value           string    `gorm:"size:255;not null" json:"value"`
	NormalizedValue string    `gorm:"size:255;not null;uniqueIndex:ux_politician_alias" json:"-"`
	Source          string    `gorm:"size:30;not null;uniqueIndex:ux_politician_alias" json:"source"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

func (PoliticianAlias) TableName() string { return "politician_aliases" }
func (p *PoliticianAlias) BeforeCreate(_ *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	return nil
}

func (Politician) TableName() string { return "politicians" }
func (p *Politician) BeforeCreate(_ *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	return nil
}

type PoliticianExternalID struct {
	ID             string    `gorm:"type:uuid;primaryKey" json:"id"`
	PoliticianID   string    `gorm:"type:uuid;not null;uniqueIndex:ux_external_identifier" json:"politicianId"`
	Source         string    `gorm:"size:30;not null;uniqueIndex:ux_external_identifier" json:"source"`
	IdentifierType string    `gorm:"size:50;not null;uniqueIndex:ux_external_identifier" json:"identifierType"`
	ExternalID     string    `gorm:"size:100;not null;uniqueIndex:ux_external_identifier" json:"externalId"`
	SourceURL      string    `gorm:"size:2048" json:"sourceUrl,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func (PoliticianExternalID) TableName() string { return "politician_external_ids" }
func (p *PoliticianExternalID) BeforeCreate(_ *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	return nil
}

type ParliamentaryMandate struct {
	ID               string `gorm:"type:uuid;primaryKey" json:"id"`
	PoliticianID     string `gorm:"type:uuid;not null;index;uniqueIndex:ux_mandate" json:"politicianId"`
	LegislativeHouse string `gorm:"size:20;not null;uniqueIndex:ux_mandate" json:"legislativeHouse"`
	Office           string `gorm:"size:100;not null;uniqueIndex:ux_mandate" json:"office"`
	State            string `gorm:"size:2;not null;uniqueIndex:ux_mandate" json:"state"`
	ElectionYear     *int   `gorm:"uniqueIndex:ux_mandate" json:"electionYear,omitempty"`
	PartyAcronym     string `gorm:"size:30" json:"partyAcronym"`
	// PartyDirection and PartyIdeology are a cache projection of the current
	// editorial party classification; they are not historical source data.
	PartyDirection   string     `gorm:"-" json:"partyDirection,omitempty"`
	PartyIdeology    string     `gorm:"-" json:"partyIdeology,omitempty"`
	Status           string     `gorm:"size:30;not null;index" json:"status"`
	StartsAt         *time.Time `json:"startsAt,omitempty"`
	EndsAt           *time.Time `json:"endsAt,omitempty"`
	SourceLastSeenAt *time.Time `json:"sourceLastSeenAt,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

func (ParliamentaryMandate) TableName() string { return "parliamentary_mandates" }
func (p *ParliamentaryMandate) BeforeCreate(_ *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	return nil
}

// PoliticianIdentityMatch retains an auditable record whenever two sources are reconciled.
type PoliticianIdentityMatch struct {
	ID           string     `gorm:"type:uuid;primaryKey" json:"id"`
	PoliticianID string     `gorm:"type:uuid;not null;index" json:"politicianId"`
	Source       string     `gorm:"size:30;not null;uniqueIndex:ux_identity_match" json:"source"`
	ExternalID   string     `gorm:"size:100;not null;uniqueIndex:ux_identity_match" json:"externalId"`
	MatchMethod  string     `gorm:"size:100;not null" json:"matchMethod"`
	Status       string     `gorm:"size:30;not null;index" json:"status"`
	ReviewedAt   *time.Time `json:"reviewedAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

func (PoliticianIdentityMatch) TableName() string { return "politician_identity_matches" }
func (p *PoliticianIdentityMatch) BeforeCreate(_ *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	return nil
}
