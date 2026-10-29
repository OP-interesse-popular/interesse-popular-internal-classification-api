package services

import (
	"testing"

	"github.com/OP-interesse-popular/interesse-popular-internal-classification/models"
)

func TestApplyPartyClassifications(t *testing.T) {
	items := []models.Parliamentarian{{Name: "Deputado", Party: "pl"}, {Name: "Sem partido", Party: "OUTRO"}}
	classifications := []models.PartyPoliticalDirection{{PartyAcronym: "PL", Direction: models.DirectionRight, Ideology: models.IdeologyLiberalRight}}

	result := applyPartyClassifications(items, classifications)
	if result[0].PartyDirection != models.DirectionRight || result[0].PartyIdeology != models.IdeologyLiberalRight {
		t.Fatalf("classification = %#v", result[0])
	}
	if result[1].PartyDirection != "" || result[1].PartyIdeology != "" {
		t.Fatalf("unclassified party changed: %#v", result[1])
	}
}
