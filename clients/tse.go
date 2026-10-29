package clients

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// ElectedCandidate is the election-specific data used to create a future mandate.
type ElectedCandidate struct {
	CandidateID  string
	FullName     string
	BallotName   string
	Party        string
	State        string
	House        string
	Office       string
	ElectionYear int
}

type TSEClient interface {
	ListElectedFederalParliamentarians(context.Context) ([]ElectedCandidate, error)
}

// TSEHTTPClient reads a CSV or ZIP/CSV export configured from the official TSE data portal.
type TSEHTTPClient struct {
	url        string
	httpClient *http.Client
}

func NewTSEHTTPClient(url string, httpClient *http.Client) (*TSEHTTPClient, error) {
	if strings.TrimSpace(url) == "" {
		return nil, nil
	}
	if httpClient == nil {
		return nil, fmt.Errorf("TSE HTTP client is required")
	}
	return &TSEHTTPClient{url: url, httpClient: httpClient}, nil
}
func (c *TSEHTTPClient) ListElectedFederalParliamentarians(ctx context.Context) ([]ElectedCandidate, error) {
	if c == nil {
		return nil, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return nil, err
	}
	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request TSE elected candidates: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TSE elected candidates: unexpected status %d", response.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, 512<<20))
	if err != nil {
		return nil, err
	}
	if len(raw) >= 4 && bytes.Equal(raw[:4], []byte{'P', 'K', 3, 4}) {
		archive, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
		if err != nil {
			return nil, err
		}
		items := make([]ElectedCandidate, 0)
		seen := make(map[string]struct{})
		for _, file := range archive.File {
			if strings.HasSuffix(strings.ToLower(file.Name), ".csv") {
				reader, err := file.Open()
				if err != nil {
					return nil, err
				}
				csvRaw, err := io.ReadAll(reader)
				reader.Close()
				if err != nil {
					return nil, err
				}
				candidates, err := parseTSECandidates(csvRaw)
				if err != nil {
					return nil, fmt.Errorf("parse TSE CSV %s: %w", file.Name, err)
				}
				for _, candidate := range candidates {
					if _, exists := seen[candidate.CandidateID]; exists {
						continue
					}
					seen[candidate.CandidateID] = struct{}{}
					items = append(items, candidate)
				}
			}
		}
		if len(items) == 0 {
			return nil, fmt.Errorf("TSE archive does not contain elected federal candidates")
		}
		return items, nil
	}
	return parseTSECandidates(raw)
}

func parseTSECandidates(raw []byte) ([]ElectedCandidate, error) {
	decoded := raw
	if !utf8.Valid(raw) {
		var err error
		decoded, err = charmap.ISO8859_1.NewDecoder().Bytes(raw)
		if err != nil {
			return nil, err
		}
	}
	reader := csv.NewReader(strings.NewReader(string(decoded)))
	reader.Comma = ';'
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read TSE header: %w", err)
	}
	columns := make(map[string]int, len(header))
	for index, value := range header {
		columns[strings.TrimPrefix(strings.TrimSpace(value), "\ufeff")] = index
	}
	required := []string{"SQ_CANDIDATO", "NM_CANDIDATO", "SG_UF", "DS_CARGO", "ANO_ELEICAO", "DS_SIT_TOT_TURNO"}
	for _, key := range required {
		if _, ok := columns[key]; !ok {
			return nil, fmt.Errorf("TSE export is missing required column %s", key)
		}
	}
	items := make([]ElectedCandidate, 0)
	seen := make(map[string]struct{})
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read TSE row: %w", err)
		}
		value := func(key string) string {
			index := columns[key]
			if index >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[index])
		}
		office, result := strings.ToUpper(value("DS_CARGO")), strings.ToUpper(value("DS_SIT_TOT_TURNO"))
		isElected := strings.HasPrefix(result, "ELEITO") || strings.HasPrefix(result, "ELEITA")
		if !isElected || (office != "DEPUTADO FEDERAL" && office != "SENADOR") {
			continue
		}
		candidateID := value("SQ_CANDIDATO")
		if candidateID == "" {
			continue
		}
		if _, ok := seen[candidateID]; ok {
			continue
		}
		seen[candidateID] = struct{}{}
		year := 0
		fmt.Sscanf(value("ANO_ELEICAO"), "%d", &year)
		house, normalizedOffice := "camara", "deputado_federal"
		if office == "SENADOR" {
			house, normalizedOffice = "senado", "senador"
		}
		items = append(items, ElectedCandidate{CandidateID: candidateID, FullName: value("NM_CANDIDATO"), BallotName: value("NM_URNA_CANDIDATO"), Party: value("SG_PARTIDO"), State: value("SG_UF"), House: house, Office: normalizedOffice, ElectionYear: year})
	}
	return items, nil
}
