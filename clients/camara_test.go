package clients

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCamaraClientFollowsNextPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("pagina") {
		case "2":
			_, _ = w.Write([]byte(`{"dados":[{"id":2,"nome":"Deputada B","siglaPartido":"PL","siglaUf":"RJ"}],"links":[]}`))
		default:
			_, _ = w.Write([]byte(`{"dados":[{"id":1,"nome":"Deputado A","siglaPartido":"PL","siglaUf":"SP"}],"links":[{"rel":"next","href":"` + serverURL(r) + `/deputados?pagina=2"}]}`))
		}
	}))
	defer server.Close()

	client, err := NewCamaraHTTPClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	items, err := client.ListActiveDeputies(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ID != "1" || items[1].ID != "2" {
		t.Fatalf("unexpected parliamentarians: %#v", items)
	}
}

func serverURL(r *http.Request) string { return "http://" + r.Host }
