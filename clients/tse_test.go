package clients

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseTSECandidatesKeepsOnlyElectedFederalParliamentarians(t *testing.T) {
	raw := []byte("SQ_CANDIDATO;NM_CANDIDATO;NM_URNA_CANDIDATO;SG_PARTIDO;SG_UF;DS_CARGO;ANO_ELEICAO;DS_SIT_TOT_TURNO\n1;Pessoa Eleita;Pessoa;PL;SP;DEPUTADO FEDERAL;2026;ELEITO\n2;Pessoa Não Eleita;Outra;PT;SP;DEPUTADO FEDERAL;2026;NÃO ELEITO\n3;Senadora Eleita;Senadora;PSOL;PE;SENADOR;2026;ELEITA\n")
	items, err := parseTSECandidates(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("items length = %d, want 2", len(items))
	}
	if items[0].House != "camara" || items[0].Office != "deputado_federal" {
		t.Fatalf("first item = %#v", items[0])
	}
	if items[1].House != "senado" || items[1].Office != "senador" {
		t.Fatalf("second item = %#v", items[1])
	}
}

func TestTSEHTTPClientReadsEveryCSVInArchive(t *testing.T) {
	archive := bytes.NewBuffer(nil)
	writer := zip.NewWriter(archive)
	for name, content := range map[string]string{
		"consulta_cand_2026_AM.csv": "SQ_CANDIDATO;NM_CANDIDATO;SG_UF;DS_CARGO;ANO_ELEICAO;DS_SIT_TOT_TURNO\n1;Pessoa AM;AM;DEPUTADO FEDERAL;2026;ELEITO\n",
		"consulta_cand_2026_PE.csv": "SQ_CANDIDATO;NM_CANDIDATO;SG_UF;DS_CARGO;ANO_ELEICAO;DS_SIT_TOT_TURNO\n2;Jones Manoel da Silva;PE;DEPUTADO FEDERAL;2026;ELEITO POR QP\n",
	} {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = file.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(archive.Bytes())
	}))
	defer server.Close()

	client, err := NewTSEHTTPClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	items, err := client.ListElectedFederalParliamentarians(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[1].FullName != "Jones Manoel da Silva" {
		t.Fatalf("expected candidates from both files, got %#v", items)
	}
}
