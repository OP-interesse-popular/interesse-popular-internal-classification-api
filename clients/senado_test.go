package clients

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSenadoClientMapsActiveSenators(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ListaParlamentarEmExercicio":{"Parlamentares":{"Parlamentar":[{"IdentificacaoParlamentar":{"CodigoParlamentar":"10","NomeParlamentar":"Senadora A","SiglaPartidoParlamentar":"PL","UfParlamentar":"SP","UrlFotoParlamentar":"photo","UrlPaginaParlamentar":"profile"}}]}}}`))
	}))
	defer server.Close()

	client, err := NewSenadoHTTPClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	items, err := client.ListActiveSenators(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Name != "Senadora A" || items[0].LegislativeHouse != "senado" {
		t.Fatalf("unexpected parliamentarians: %#v", items)
	}
}
