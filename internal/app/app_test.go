package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rsahara/timich-mcp/internal/agentclient"
	"github.com/rsahara/timich-mcp/internal/state"
)

func TestBuildSearchRequestMapsThinWrapper(t *testing.T) {
	request, err := BuildSearchRequest(SearchAssetsInput{
		Text:      "サッカー",
		Mode:      "filename",
		MediaType: "image",
		CapturedAt: &CapturedAtInput{
			From: "2026-05-01T00:00:00Z",
			To:   "2026-05-13T00:00:00Z",
		},
		Sort:     "capturedAtDesc",
		Page:     2,
		PageSize: 250,
	})
	if err != nil {
		t.Fatalf("BuildSearchRequest() error = %v", err)
	}
	if request.Collection.Kind != "search" {
		t.Fatalf("collection kind = %q", request.Collection.Kind)
	}
	if request.Collection.Query == nil || request.Collection.Query.Text != "サッカー" || request.Collection.Query.Mode != "filename" {
		t.Fatalf("query = %+v", request.Collection.Query)
	}
	if got := request.Collection.Filters.MediaTypes; len(got) != 1 || got[0] != "image" {
		t.Fatalf("mediaTypes = %v", got)
	}
	if request.Page.Index != 2 || request.Page.Size != 200 {
		t.Fatalf("page = %+v", request.Page)
	}
	if request.Collection.Sort == nil || request.Collection.Sort.Field != "capturedAt" || request.Collection.Sort.Direction != "desc" {
		t.Fatalf("sort = %+v", request.Collection.Sort)
	}
}

func TestEnsureSessionRefreshesWhenRefreshTokenNearExpiry(t *testing.T) {
	now := time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC)
	refreshCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/session/refresh" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		refreshCalled = true
		_ = json.NewEncoder(w).Encode(agentclient.SessionBundle{
			AccessToken:           "new-access",
			RefreshToken:          "new-refresh",
			AgentID:               "agent",
			AgentName:             "Agent",
			DeviceID:              "device",
			BaseURL:               serverURL(r),
			AccessMode:            "full",
			AccessTokenExpiresAt:  now.Add(12 * time.Hour),
			RefreshTokenExpiresAt: now.Add(30 * 24 * time.Hour),
		})
	}))
	defer server.Close()

	store := state.NewStore(t.TempDir())
	if err := store.Save(state.File{
		AgentBaseURL:          server.URL,
		AccessToken:           "old-access",
		RefreshToken:          "old-refresh",
		AccessTokenExpiresAt:  now.Add(6 * time.Hour),
		RefreshTokenExpiresAt: now.Add(7 * 24 * time.Hour),
		DeviceName:            "Timich MCP on test",
		PairedAt:              now.Add(-time.Hour),
		UpdatedAt:             now.Add(-time.Hour),
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	timich := New(store, server.Client(), filepath.Join(t.TempDir(), "previews"))
	timich.Now = func() time.Time { return now }

	session, err := timich.EnsureSession(context.Background())
	if err != nil {
		t.Fatalf("EnsureSession() error = %v", err)
	}
	if !refreshCalled {
		t.Fatal("refresh was not called")
	}
	if session.AccessToken != "new-access" || session.RefreshToken != "new-refresh" {
		t.Fatalf("session = %+v", session)
	}
}

func TestGetAssetPreviewWritesHashedFile(t *testing.T) {
	now := time.Date(2026, 5, 13, 12, 0, 0, 0, time.UTC)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/assets/ta1_test/preview" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("jpeg"))
	}))
	defer server.Close()

	store := state.NewStore(t.TempDir())
	if err := store.Save(state.File{
		AgentBaseURL:          server.URL,
		AccessToken:           "access",
		RefreshToken:          "refresh",
		AccessTokenExpiresAt:  now.Add(time.Hour),
		RefreshTokenExpiresAt: now.Add(30 * 24 * time.Hour),
		DeviceName:            "Timich MCP on test",
		PairedAt:              now,
		UpdatedAt:             now,
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	previewDir := filepath.Join(t.TempDir(), "previews")
	timich := New(store, server.Client(), previewDir)
	timich.Now = func() time.Time { return now }

	output, err := timich.GetAssetPreview(context.Background(), PreviewInput{AssetID: "ta1_test", Filename: "photo.jpg"})
	if err != nil {
		t.Fatalf("GetAssetPreview() error = %v", err)
	}
	if filepath.Dir(output.Path) != previewDir || filepath.Ext(output.Path) != ".jpg" {
		t.Fatalf("preview path = %s", output.Path)
	}
	raw, err := os.ReadFile(output.Path)
	if err != nil {
		t.Fatalf("read preview: %v", err)
	}
	if string(raw) != "jpeg" || output.ContentType != "image/jpeg" || output.Filename != "photo.jpg" {
		t.Fatalf("output = %+v body=%q", output, raw)
	}
}

func serverURL(r *http.Request) string {
	return "http://" + r.Host
}
