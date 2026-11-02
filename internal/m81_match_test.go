package internal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	xxhash32 "github.com/OneOfOne/xxhash"
	"ottplay-foss-server-epg/pkg/config"
	"ottplay-foss-server-epg/pkg/mem_storage"
)

func TestM81MatchReturnsOriginalIdentityAndRejectsHashOnly(t *testing.T) {
	old := PO
	defer func() { PO = old }()
	channel := &mem_storage.EpgChannelData{Id: "xml.canal", Names: []string{"canal um"}, LastEpg: time.Now().Add(time.Hour).Unix()}
	PO.Epg = []*mem_storage.ProviderEpgData{{
		Prov:   &mem_storage.ProviderElement{Id: "fonte-a"},
		ById:   mem_storage.HashEpgStruct{xxhash32.ChecksumString32("xml.canal"): channel},
		ByName: mem_storage.HashEpgStruct{xxhash32.ChecksumString32("canal um"): channel},
	}}
	request := httptest.NewRequest(http.MethodPost, "/api/m81/match", strings.NewReader(`{"channels":[{"key":"m81-1","tvg_id":"xml.canal"},{"key":"m81-2","name":"Canal Um"},{"key":"m81-3","tvg_id":"outro"}]}`))
	w := httptest.NewRecorder()
	M81Match(w, request)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var response m81Response
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Suggestions) != 2 || response.Suggestions[0].XMLTVID != "xml.canal" || response.Suggestions[1].SourceID != "fonte-a" {
		t.Fatalf("unexpected suggestions: %+v", response.Suggestions)
	}
}

func TestM81MatchRequiresAdminTokenAndBoundsInput(t *testing.T) {
	old := Config
	Config = &config.ConfigFile{AdminTokens: []string{"secret-for-test"}}
	defer func() { Config = old }()
	unauthorized := httptest.NewRecorder()
	PrivateApi(unauthorized, httptest.NewRequest(http.MethodPost, "/api/m81/match", strings.NewReader(`{"channels":[]}`)))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", unauthorized.Code)
	}
	if unauthorized.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("M81 admin matcher must not expose wildcard CORS")
	}
	oversized := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/m81/match", strings.NewReader(`{"channels":[]}`+strings.Repeat(" ", maxM81MatchBody)))
	request.Header.Set("X-Token", "secret-for-test")
	PrivateApi(oversized, request)
	if oversized.Code != http.StatusBadRequest {
		t.Fatalf("expected bounded input, got %d", oversized.Code)
	}
}

func TestM81MatchDoesNotTrustHashWithoutOriginalIdentity(t *testing.T) {
	old := PO
	defer func() { PO = old }()
	channel := &mem_storage.EpgChannelData{Id: "outro.xml", Names: []string{"outro nome"}, LastEpg: time.Now().Add(time.Hour).Unix()}
	PO.Epg = []*mem_storage.ProviderEpgData{{
		Prov:   &mem_storage.ProviderElement{Id: "fonte-a"},
		ById:   mem_storage.HashEpgStruct{xxhash32.ChecksumString32("alvo.xml"): channel},
		ByName: mem_storage.HashEpgStruct{xxhash32.ChecksumString32("nome alvo"): channel},
	}}
	w := httptest.NewRecorder()
	M81Match(w, httptest.NewRequest(http.MethodPost, "/api/m81/match", strings.NewReader(`{"channels":[{"key":"m81-1","tvg_id":"alvo.xml","name":"Nome Alvo"}]}`)))
	var response m81Response
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Suggestions) != 0 {
		t.Fatalf("hash-only match accepted: %+v", response.Suggestions)
	}
}
