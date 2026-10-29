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

// CamaraClient is the boundary between the application and Câmara's open-data API.
type CamaraClient interface {
	ListActiveDeputies(context.Context) ([]models.Parliamentarian, error)
}

type CamaraHTTPClient struct {
	baseURL    *url.URL
	httpClient *http.Client
}

func NewCamaraHTTPClient(baseURL string, httpClient *http.Client) (*CamaraHTTPClient, error) {
	parsedURL, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, fmt.Errorf("invalid Camara base URL")
	}
	if httpClient == nil {
		return nil, fmt.Errorf("Camara HTTP client is required")
	}
	return &CamaraHTTPClient{baseURL: parsedURL, httpClient: httpClient}, nil
}

// ListActiveDeputies retrieves all current federal deputies, following the API's next links.
func (c *CamaraHTTPClient) ListActiveDeputies(ctx context.Context) ([]models.Parliamentarian, error) {
	endpoint := c.baseURL.JoinPath("deputados")
	query := endpoint.Query()
	query.Set("itens", "100")
	query.Set("ordem", "ASC")
	query.Set("ordenarPor", "nome")
	endpoint.RawQuery = query.Encode()
	items := make([]models.Parliamentarian, 0, 600)
	for page := 1; endpoint != nil; page++ {
		if page > 20 {
			return nil, fmt.Errorf("Camara pagination limit reached")
		}
		payload, err := c.getPage(ctx, endpoint.String())
		if err != nil {
			return nil, fmt.Errorf("request Camara deputies page %d: %w", page, err)
		}
		for _, deputy := range payload.Data {
			items = append(items, models.Parliamentarian{ID: fmt.Sprint(deputy.ID), Name: deputy.Name, Party: deputy.Party, State: deputy.State, LegislativeHouse: "camara", PhotoURL: deputy.PhotoURL, SourceURL: deputy.URI})
		}
		endpoint = nil
		for _, link := range payload.Links {
			if link.Rel == "next" {
				next, err := url.Parse(link.Href)
				if err != nil {
					return nil, fmt.Errorf("parse Camara next page: %w", err)
				}
				endpoint = next
				break
			}
		}
	}
	return items, nil
}

func (c *CamaraHTTPClient) getPage(ctx context.Context, endpoint string) (camaraDeputiesResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return camaraDeputiesResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	response, err := c.httpClient.Do(req)
	if err != nil {
		return camaraDeputiesResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return camaraDeputiesResponse{}, fmt.Errorf("unexpected status %d", response.StatusCode)
	}
	var payload camaraDeputiesResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return camaraDeputiesResponse{}, fmt.Errorf("decode response: %w", err)
	}
	return payload, nil
}

type camaraDeputiesResponse struct {
	Data  []camaraDeputy `json:"dados"`
	Links []camaraLink   `json:"links"`
}
type camaraDeputy struct {
	ID       int    `json:"id"`
	Name     string `json:"nome"`
	Party    string `json:"siglaPartido"`
	State    string `json:"siglaUf"`
	PhotoURL string `json:"urlFoto"`
	URI      string `json:"uri"`
}
type camaraLink struct {
	Href string `json:"href"`
	Rel  string `json:"rel"`
}
