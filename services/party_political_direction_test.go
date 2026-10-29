package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/OP-interesse-popular/interesse-popular-internal-classification/models"
	"github.com/OP-interesse-popular/interesse-popular-internal-classification/repositories"
)

type partyRepositoryStub struct{}

type cacheRefresherStub struct{ calls int }

func (stub *cacheRefresherStub) Refresh(context.Context) { stub.calls++ }

func (partyRepositoryStub) Create(item *models.PartyPoliticalDirection) error {
	if item.PartyAcronym == "EXISTS" {
		return repositories.ErrConflict
	}
	return nil
}
func (partyRepositoryStub) List() ([]models.PartyPoliticalDirection, error) { return nil, nil }
func (partyRepositoryStub) ListActive() ([]models.PartyPoliticalDirection, error) {
	return nil, nil
}
func (partyRepositoryStub) ListActiveByDirection(string) ([]models.PartyPoliticalDirection, error) {
	return nil, nil
}
func (partyRepositoryStub) ListActiveByDirectionAndIdeologies(string, []string) ([]models.PartyPoliticalDirection, error) {
	return nil, nil
}
func (partyRepositoryStub) FindByID(string) (*models.PartyPoliticalDirection, error) {
	return nil, repositories.ErrNotFound
}
func (partyRepositoryStub) Update(*models.PartyPoliticalDirection) error { return nil }
func (partyRepositoryStub) Delete(string) error                          { return nil }

func validInput() SavePartyPoliticalDirectionInput {
	return SavePartyPoliticalDirectionInput{PartyTSENumber: 99, PartyAcronym: "SIGLA", PartyName: "Partido de teste", Direction: models.DirectionLeft, Ideology: models.IdeologyLiberalLeft, MethodologyVersion: "v1.0", SourceName: "Fonte", ValidFrom: time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)}
}

func TestPartyPoliticalDirectionFromAcceptsCompatibleClassification(t *testing.T) {
	item, err := partyPoliticalDirectionFrom(validInput())
	if err != nil {
		t.Fatalf("partyPoliticalDirectionFrom returned error: %v", err)
	}
	if item.PartyAcronym != "SIGLA" {
		t.Fatalf("party acronym = %q", item.PartyAcronym)
	}
}

func TestPartyPoliticalDirectionFromRejectsIncompatibleClassification(t *testing.T) {
	input := validInput()
	input.Direction = models.DirectionLeft
	input.Ideology = models.IdeologyFarRight
	_, err := partyPoliticalDirectionFrom(input)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
}

func TestCreateManyReturnsOneResultPerInput(t *testing.T) {
	service := NewPartyPoliticalDirection(partyRepositoryStub{})
	valid := validInput()
	conflict := validInput()
	conflict.PartyAcronym = "EXISTS"

	results := service.CreateMany([]SavePartyPoliticalDirectionInput{valid, conflict})
	if len(results) != 2 {
		t.Fatalf("results length = %d, want 2", len(results))
	}
	if results[0].Status != "success" {
		t.Fatalf("first status = %q", results[0].Status)
	}
	if results[1].Status != "error" || results[1].Error.Code != "conflict" {
		t.Fatalf("second result = %#v", results[1])
	}
}

func TestCreateRequestsCacheRefresh(t *testing.T) {
	refresher := &cacheRefresherStub{}
	service := NewPartyPoliticalDirection(partyRepositoryStub{}, refresher)
	if _, err := service.Create(validInput()); err != nil {
		t.Fatal(err)
	}
	if refresher.calls != 1 {
		t.Fatalf("refresh calls = %d, want 1", refresher.calls)
	}
}
