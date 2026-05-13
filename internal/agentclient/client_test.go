package agentclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNormalizeBaseURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "adds http", raw: "10.0.1.4:8082/", want: "http://10.0.1.4:8082"},
		{name: "keeps https path", raw: "https://example.test/agent/", want: "https://example.test/agent"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeBaseURL(tt.raw)
			if err != nil {
				t.Fatalf("NormalizeBaseURL() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("NormalizeBaseURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClientSearchSendsBearerAndDecodesPage(t *testing.T) {
	capturedAuth := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/assets/search" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		capturedAuth = r.Header.Get("Authorization")
		var request SearchRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if request.Collection.Kind != "search" || request.Page.Size != 20 {
			t.Fatalf("request = %+v", request)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(SearchPage{
			CollectionKey: "search_v1:test",
			Page:          SearchPageRequest{Index: 0, Size: 20},
			Items: []Asset{{
				ID:         "ta1_test",
				Type:       "image",
				Filename:   "ball.jpg",
				CapturedAt: time.Date(2026, 5, 13, 1, 2, 3, 0, time.UTC),
			}},
			Total:         1,
			TotalAccuracy: "exact",
		})
	}))
	defer server.Close()

	client := New(server.URL, server.Client())
	page, err := client.Search(context.Background(), "access", SearchRequest{
		Collection: SearchCollectionRequest{
			Kind:  "search",
			Query: &SearchQuery{Text: "soccer", Mode: "auto"},
		},
		Page: SearchPageRequest{Index: 0, Size: 20},
	})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if capturedAuth != "Bearer access" {
		t.Fatalf("Authorization = %q", capturedAuth)
	}
	if len(page.Items) != 1 || page.Items[0].Filename != "ball.jpg" {
		t.Fatalf("Search() page = %+v", page)
	}
}
