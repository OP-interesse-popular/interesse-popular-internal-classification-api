package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/OP-interesse-popular/interesse-popular-internal-classification/models"
)

// SenadoClient is the boundary between the application and Senado's open-data API.
type SenadoClient interface {
	ListActiveSenators(context.Context) ([]models.Parliamentarian, error)
}
type SenadoHTTPClient struct {
	baseURL    *url.URL
	httpClient *http.Client
}

func NewSenadoHTTPClient(baseURL string, httpClient *http.Client) (*SenadoHTTPClient, error) {
	parsedURL, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("invalid Senado base URL")
	}
	if httpClient == nil {
		return nil, fmt.Errorf("Senado HTTP client is required")
	}
	return &SenadoHTTPClient{baseURL: parsedURL, httpClient: httpClient}, nil
}

func (c *SenadoHTTPClient) ListActiveSenators(ctx context.Context) ([]models.Parliamentarian, error) {
	endpoint := c.baseURL.JoinPath("senador", "lista", "atual.json")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request Senado senators: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Senado senators: unexpected status %d", response.StatusCode)
	}
	var payload senadoSenatorsResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode Senado senators: %w", err)
	}
	items := make([]models.Parliamentarian, 0, len(payload.List.Parliamentarians.Parliamentarian))
	for _, senator := range payload.List.Parliamentarians.Parliamentarian {
		identity := senator.Identity
		items = append(items, models.Parliamentarian{ID: identity.Code, Name: identity.Name, Party: identity.Party, State: identity.State, LegislativeHouse: "senado", PhotoURL: identity.PhotoURL, SourceURL: identity.ProfileURL})
	}
	return items, nil
}

type senadoSenatorsResponse struct {
	List struct {
		Parliamentarians struct {
			Parliamentarian []senadoSenator `json:"Parlamentar"`
		} `json:"Parlamentares"`
	} `json:"ListaParlamentarEmExercicio"`
}
type senadoSenator struct {
	Identity struct {
		Code       string `json:"CodigoParlamentar"`
		Name       string `json:"NomeParlamentar"`
		Party      string `json:"SiglaPartidoParlamentar"`
		State      string `json:"UfParlamentar"`
		PhotoURL   string `json:"UrlFotoParlamentar"`
		ProfileURL string `json:"UrlPaginaParlamentar"`
	} `json:"IdentificacaoParlamentar"`
}
