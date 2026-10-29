package models

type Parliamentarian struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Party            string `json:"party"`
	PartyDirection   string `json:"party_direction,omitempty"`
	PartyIdeology    string `json:"party_ideology,omitempty"`
	State            string `json:"state"`
	LegislativeHouse string `json:"legislative_house"`
	PhotoURL         string `json:"photo_url"`
	SourceURL        string `json:"source_url"`
}
