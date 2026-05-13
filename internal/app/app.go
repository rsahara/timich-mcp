package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rsahara/timich-mcp/internal/agentclient"
	"github.com/rsahara/timich-mcp/internal/state"
)

const (
	defaultPageSize             = 20
	maxPageSize                 = 200
	accessTokenRefreshLead      = 5 * time.Minute
	refreshTokenRefreshLead     = 14 * 24 * time.Hour
	previewMaxAge               = 24 * time.Hour
	defaultPreviewDirectoryName = "timich-mcp/previews"
)

type App struct {
	Store      *state.Store
	HTTPClient *http.Client
	PreviewDir string
	Now        func() time.Time
}

type Error struct {
	Code    string
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Code == "" {
		return e.Message
	}
	return e.Code + ": " + e.Message
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type PairResult struct {
	Paired  bool          `json:"paired"`
	Status  *StatusResult `json:"status,omitempty"`
	Warning string        `json:"warning,omitempty"`
}

type StatusResult struct {
	Paired                bool      `json:"paired"`
	AgentBaseURL          string    `json:"agentBaseURL,omitempty"`
	AgentID               string    `json:"agentId,omitempty"`
	AgentName             string    `json:"agentName,omitempty"`
	DeviceID              string    `json:"deviceId,omitempty"`
	DeviceName            string    `json:"deviceName,omitempty"`
	AccessMode            string    `json:"accessMode,omitempty"`
	AccessTokenExpiresAt  time.Time `json:"accessTokenExpiresAt,omitempty"`
	RefreshTokenExpiresAt time.Time `json:"refreshTokenExpiresAt,omitempty"`
	CapabilitiesOK        bool      `json:"capabilitiesOK"`
	SearchOK              bool      `json:"searchOK"`
	SearchTotal           int       `json:"searchTotal,omitempty"`
	SearchTotalAccuracy   string    `json:"searchTotalAccuracy,omitempty"`
	SearchItemCount       int       `json:"searchItemCount,omitempty"`
}

type CapabilitiesOutput struct {
	Paired       bool                           `json:"paired"`
	AgentBaseURL string                         `json:"agentBaseURL"`
	Capabilities agentclient.SearchCapabilities `json:"capabilities"`
}

type SearchAssetsInput struct {
	Text       string           `json:"text,omitempty" jsonschema:"Optional search text. When present, Timich searches assets; when omitted, Timich browses the timeline."`
	Mode       string           `json:"mode,omitempty" jsonschema:"Search mode for text queries: auto, semantic, or filename. Defaults to auto."`
	MediaType  string           `json:"mediaType,omitempty" jsonschema:"Optional media type filter: image or video."`
	CapturedAt *CapturedAtInput `json:"capturedAt,omitempty" jsonschema:"Optional UTC captured time filter. Use RFC3339 timestamps ending in Z."`
	Sort       string           `json:"sort,omitempty" jsonschema:"Optional sort: default, capturedAtDesc, or relevanceDesc. The default lets Timich Agent choose the backend default."`
	Page       int              `json:"page,omitempty" jsonschema:"Zero-based page index. Defaults to 0."`
	PageSize   int              `json:"pageSize,omitempty" jsonschema:"Page size. Defaults to 20 and is capped at 200."`
}

type CapturedAtInput struct {
	From string `json:"from,omitempty" jsonschema:"Inclusive lower captured-at bound as a UTC RFC3339 timestamp ending in Z."`
	To   string `json:"to,omitempty" jsonschema:"Exclusive upper captured-at bound as a UTC RFC3339 timestamp ending in Z."`
}

type SearchAssetsOutput struct {
	CollectionKey string                        `json:"collectionKey"`
	Total         int                           `json:"total"`
	TotalAccuracy string                        `json:"totalAccuracy"`
	Page          agentclient.SearchPageRequest `json:"page"`
	NextPageIndex *int                          `json:"nextPageIndex,omitempty"`
	Boundary      *agentclient.SearchBoundary   `json:"boundary,omitempty"`
	Resolved      agentclient.SearchResolved    `json:"resolved"`
	Items         []agentclient.Asset           `json:"items"`
}

type PreviewInput struct {
	AssetID  string `json:"assetId" jsonschema:"Timich asset id returned by search_assets."`
	Filename string `json:"filename,omitempty" jsonschema:"Optional original filename from the search result, returned for caller convenience."`
}

type PreviewOutput struct {
	Path        string `json:"path"`
	ContentType string `json:"contentType"`
	Filename    string `json:"filename,omitempty"`
	AssetID     string `json:"assetId"`
}

func New(store *state.Store, httpClient *http.Client, previewDir string) *App {
	if store == nil {
		store = state.NewStore("")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	if previewDir == "" {
		previewDir = DefaultPreviewDir()
	}
	return &App{
		Store:      store,
		HTTPClient: httpClient,
		PreviewDir: previewDir,
		Now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func DefaultPreviewDir() string {
	return filepath.Join(os.TempDir(), defaultPreviewDirectoryName)
}

func DefaultDeviceName() string {
	host, err := os.Hostname()
	if err != nil || strings.TrimSpace(host) == "" {
		return "Timich MCP"
	}
	return "Timich MCP on " + strings.TrimSpace(host)
}

func (a *App) Pair(ctx context.Context, agentURL string, pairingCode string, deviceName string) (PairResult, error) {
	normalizedURL, err := agentclient.NormalizeBaseURL(agentURL)
	if err != nil {
		return PairResult{}, userError("invalid_input", err.Error(), err)
	}
	pairingCode = strings.TrimSpace(pairingCode)
	if pairingCode == "" {
		return PairResult{}, userError("invalid_input", "pairing code is required", nil)
	}
	deviceName = strings.TrimSpace(deviceName)
	if deviceName == "" {
		deviceName = DefaultDeviceName()
	}

	client := a.client(normalizedURL)
	bundle, err := client.Pair(ctx, pairingCode, deviceName)
	if err != nil {
		return PairResult{}, mapAgentError(err)
	}
	now := a.now()
	if err := a.Store.Save(stateFromBundle(bundle, normalizedURL, deviceName, now)); err != nil {
		return PairResult{}, err
	}

	status, err := a.Status(ctx)
	if err != nil {
		return PairResult{
			Paired:  true,
			Warning: ErrorMessage(err),
		}, nil
	}
	return PairResult{Paired: true, Status: &status}, nil
}

func (a *App) Status(ctx context.Context) (StatusResult, error) {
	session, err := a.EnsureSession(ctx)
	if err != nil {
		return StatusResult{}, err
	}
	client := a.client(session.AgentBaseURL)
	if _, err := client.Capabilities(ctx, session.AccessToken); err != nil {
		if agentclient.IsUnauthorized(err) {
			session, err = a.forceRefresh(ctx, session)
			if err != nil {
				return StatusResult{}, err
			}
			if _, err = client.Capabilities(ctx, session.AccessToken); err != nil {
				return StatusResult{}, mapAgentError(err)
			}
		} else {
			return StatusResult{}, mapAgentError(err)
		}
	}
	page, err := client.Search(ctx, session.AccessToken, agentclient.SearchRequest{
		Collection: agentclient.SearchCollectionRequest{Kind: "timeline"},
		Page:       agentclient.SearchPageRequest{Index: 0, Size: 1},
	})
	if err != nil {
		if !agentclient.IsUnauthorized(err) {
			return StatusResult{}, mapAgentError(err)
		}
		session, err = a.forceRefresh(ctx, session)
		if err != nil {
			return StatusResult{}, err
		}
		page, err = client.Search(ctx, session.AccessToken, agentclient.SearchRequest{
			Collection: agentclient.SearchCollectionRequest{Kind: "timeline"},
			Page:       agentclient.SearchPageRequest{Index: 0, Size: 1},
		})
		if err != nil {
			return StatusResult{}, mapAgentError(err)
		}
	}
	return StatusResult{
		Paired:                true,
		AgentBaseURL:          session.AgentBaseURL,
		AgentID:               session.AgentID,
		AgentName:             session.AgentName,
		DeviceID:              session.DeviceID,
		DeviceName:            session.DeviceName,
		AccessMode:            session.AccessMode,
		AccessTokenExpiresAt:  session.AccessTokenExpiresAt,
		RefreshTokenExpiresAt: session.RefreshTokenExpiresAt,
		CapabilitiesOK:        true,
		SearchOK:              true,
		SearchTotal:           page.Total,
		SearchTotalAccuracy:   page.TotalAccuracy,
		SearchItemCount:       len(page.Items),
	}, nil
}

func (a *App) GetSearchCapabilities(ctx context.Context) (CapabilitiesOutput, error) {
	session, err := a.EnsureSession(ctx)
	if err != nil {
		return CapabilitiesOutput{}, err
	}
	client := a.client(session.AgentBaseURL)
	capabilities, err := client.Capabilities(ctx, session.AccessToken)
	if err != nil {
		if !agentclient.IsUnauthorized(err) {
			return CapabilitiesOutput{}, mapAgentError(err)
		}
		session, err = a.forceRefresh(ctx, session)
		if err != nil {
			return CapabilitiesOutput{}, err
		}
		capabilities, err = client.Capabilities(ctx, session.AccessToken)
		if err != nil {
			return CapabilitiesOutput{}, mapAgentError(err)
		}
	}
	return CapabilitiesOutput{
		Paired:       true,
		AgentBaseURL: session.AgentBaseURL,
		Capabilities: capabilities,
	}, nil
}

func (a *App) SearchAssets(ctx context.Context, input SearchAssetsInput) (SearchAssetsOutput, error) {
	request, err := BuildSearchRequest(input)
	if err != nil {
		return SearchAssetsOutput{}, err
	}
	session, err := a.EnsureSession(ctx)
	if err != nil {
		return SearchAssetsOutput{}, err
	}
	client := a.client(session.AgentBaseURL)
	page, err := client.Search(ctx, session.AccessToken, request)
	if err != nil {
		if !agentclient.IsUnauthorized(err) {
			return SearchAssetsOutput{}, mapAgentError(err)
		}
		session, err = a.forceRefresh(ctx, session)
		if err != nil {
			return SearchAssetsOutput{}, err
		}
		page, err = client.Search(ctx, session.AccessToken, request)
		if err != nil {
			return SearchAssetsOutput{}, mapAgentError(err)
		}
	}
	return SearchAssetsOutput{
		CollectionKey: page.CollectionKey,
		Total:         page.Total,
		TotalAccuracy: page.TotalAccuracy,
		Page:          page.Page,
		NextPageIndex: page.NextPageIndex,
		Boundary:      page.Boundary,
		Resolved:      page.Resolved,
		Items:         page.Items,
	}, nil
}

func BuildSearchRequest(input SearchAssetsInput) (agentclient.SearchRequest, error) {
	text := strings.TrimSpace(input.Text)
	mode := strings.TrimSpace(input.Mode)
	if mode == "" {
		mode = "auto"
	}
	if mode != "auto" && mode != "semantic" && mode != "filename" {
		return agentclient.SearchRequest{}, userError("invalid_input", "mode must be auto, semantic, or filename", nil)
	}
	if input.Page < 0 {
		return agentclient.SearchRequest{}, userError("invalid_input", "page must be zero or greater", nil)
	}
	pageSize := input.PageSize
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	if pageSize < 1 {
		return agentclient.SearchRequest{}, userError("invalid_input", "pageSize must be greater than zero", nil)
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	collection := agentclient.SearchCollectionRequest{Kind: "timeline"}
	if text != "" {
		collection.Kind = "search"
		collection.Query = &agentclient.SearchQuery{Text: text, Mode: mode}
	}

	mediaType := strings.ToLower(strings.TrimSpace(input.MediaType))
	if mediaType != "" {
		if mediaType != "image" && mediaType != "video" {
			return agentclient.SearchRequest{}, userError("invalid_input", "mediaType must be image or video", nil)
		}
		collection.Filters.MediaTypes = []string{mediaType}
	}
	if input.CapturedAt != nil {
		capturedAt, err := buildCapturedAtFilter(*input.CapturedAt)
		if err != nil {
			return agentclient.SearchRequest{}, err
		}
		collection.Filters.CapturedAt = capturedAt
	}
	sort := strings.TrimSpace(input.Sort)
	if sort == "" {
		sort = "default"
	}
	switch sort {
	case "default":
	case "capturedAtDesc":
		collection.Sort = &agentclient.SearchSort{Field: "capturedAt", Direction: "desc"}
	case "relevanceDesc":
		collection.Sort = &agentclient.SearchSort{Field: "relevance", Direction: "desc"}
	default:
		return agentclient.SearchRequest{}, userError("invalid_input", "sort must be default, capturedAtDesc, or relevanceDesc", nil)
	}

	return agentclient.SearchRequest{
		Collection: collection,
		Page:       agentclient.SearchPageRequest{Index: input.Page, Size: pageSize},
	}, nil
}

func (a *App) GetAssetPreview(ctx context.Context, input PreviewInput) (PreviewOutput, error) {
	assetID := strings.TrimSpace(input.AssetID)
	if assetID == "" {
		return PreviewOutput{}, userError("invalid_input", "assetId is required", nil)
	}
	_ = a.CleanupOldPreviews()

	session, err := a.EnsureSession(ctx)
	if err != nil {
		return PreviewOutput{}, err
	}
	client := a.client(session.AgentBaseURL)
	preview, err := client.Preview(ctx, session.AccessToken, assetID)
	if err != nil {
		if !agentclient.IsUnauthorized(err) {
			return PreviewOutput{}, mapAgentError(err)
		}
		session, err = a.forceRefresh(ctx, session)
		if err != nil {
			return PreviewOutput{}, err
		}
		preview, err = client.Preview(ctx, session.AccessToken, assetID)
		if err != nil {
			return PreviewOutput{}, mapAgentError(err)
		}
	}
	path, err := a.writePreviewFile(assetID, preview.ContentType, preview.Body)
	if err != nil {
		return PreviewOutput{}, err
	}
	return PreviewOutput{
		Path:        path,
		ContentType: preview.ContentType,
		Filename:    strings.TrimSpace(input.Filename),
		AssetID:     assetID,
	}, nil
}

func (a *App) Logout() error {
	var errs []error
	if err := a.Store.Delete(); err != nil {
		errs = append(errs, err)
	}
	if err := os.RemoveAll(a.PreviewDir); err != nil {
		errs = append(errs, fmt.Errorf("remove preview directory: %w", err))
	}
	return errors.Join(errs...)
}

func (a *App) CleanupOldPreviews() error {
	cutoff := a.now().Add(-previewMaxAge)
	return filepath.WalkDir(a.PreviewDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		if entry == nil || entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(path)
		}
		return nil
	})
}

func (a *App) EnsureSession(ctx context.Context) (state.File, error) {
	session, err := a.Store.Load()
	if err != nil {
		if errors.Is(err, state.ErrNotPaired) {
			return state.File{}, userError("not_paired", "Run `timich-mcp pair --agent-url http://HOST:8082 --pairing-code CODE` first.", err)
		}
		return state.File{}, err
	}
	normalizedURL, err := agentclient.NormalizeBaseURL(session.AgentBaseURL)
	if err != nil {
		return state.File{}, userError("invalid_state", "The saved agent URL is not valid. Pair again to replace local state.", err)
	}
	session.AgentBaseURL = normalizedURL
	if shouldRefresh(session, a.now) {
		return a.forceRefresh(ctx, session)
	}
	return session, nil
}

func (a *App) forceRefresh(ctx context.Context, session state.File) (state.File, error) {
	if session.RefreshToken == "" {
		return state.File{}, userError("not_authenticated", "No refresh token is saved. Pair again.", nil)
	}
	client := a.client(session.AgentBaseURL)
	bundle, err := client.Refresh(ctx, session.RefreshToken)
	if err != nil {
		if agentclient.IsUnauthorized(err) {
			return state.File{}, userError("not_authenticated", "The saved session is no longer valid. Pair again.", err)
		}
		return state.File{}, mapAgentError(err)
	}
	now := a.now()
	next := stateFromBundle(bundle, session.AgentBaseURL, session.DeviceName, now)
	if !session.PairedAt.IsZero() {
		next.PairedAt = session.PairedAt
	}
	if next.DeviceName == "" {
		next.DeviceName = session.DeviceName
	}
	if err := a.Store.Save(next); err != nil {
		return state.File{}, err
	}
	return next, nil
}

func buildCapturedAtFilter(input CapturedAtInput) (*agentclient.SearchCapturedAt, error) {
	from := strings.TrimSpace(input.From)
	to := strings.TrimSpace(input.To)
	if from == "" && to == "" {
		return nil, nil
	}
	if from != "" {
		if err := validateUTCTime(from); err != nil {
			return nil, err
		}
	}
	if to != "" {
		if err := validateUTCTime(to); err != nil {
			return nil, err
		}
	}
	if from != "" && to != "" {
		fromTime, _ := time.Parse(time.RFC3339Nano, from)
		toTime, _ := time.Parse(time.RFC3339Nano, to)
		if !fromTime.Before(toTime) {
			return nil, userError("invalid_input", "capturedAt.from must be before capturedAt.to", nil)
		}
	}
	filter := &agentclient.SearchCapturedAt{}
	if from != "" {
		filter.From = &from
	}
	if to != "" {
		filter.To = &to
	}
	return filter, nil
}

func validateUTCTime(value string) error {
	if !strings.HasSuffix(value, "Z") {
		return userError("invalid_input", "capturedAt timestamps must be UTC RFC3339 values ending in Z", nil)
	}
	if _, err := time.Parse(time.RFC3339Nano, value); err != nil {
		return userError("invalid_input", "capturedAt timestamps must be valid RFC3339 values", err)
	}
	return nil
}

func shouldRefresh(session state.File, now func() time.Time) bool {
	current := now()
	if session.AccessToken == "" || !current.Add(accessTokenRefreshLead).Before(session.AccessTokenExpiresAt) {
		return true
	}
	if session.RefreshTokenExpiresAt.IsZero() {
		return false
	}
	return !current.Add(refreshTokenRefreshLead).Before(session.RefreshTokenExpiresAt)
}

func stateFromBundle(bundle agentclient.SessionBundle, agentBaseURL string, deviceName string, now time.Time) state.File {
	if deviceName == "" {
		deviceName = DefaultDeviceName()
	}
	return state.File{
		AgentBaseURL:          agentBaseURL,
		AccessToken:           bundle.AccessToken,
		RefreshToken:          bundle.RefreshToken,
		AccessTokenExpiresAt:  bundle.AccessTokenExpiresAt.UTC(),
		RefreshTokenExpiresAt: bundle.RefreshTokenExpiresAt.UTC(),
		DeviceName:            deviceName,
		AgentID:               bundle.AgentID,
		AgentName:             bundle.AgentName,
		DeviceID:              bundle.DeviceID,
		AccessMode:            bundle.AccessMode,
		PairedAt:              now.UTC(),
		UpdatedAt:             now.UTC(),
	}
}

func (a *App) writePreviewFile(assetID string, contentType string, body []byte) (string, error) {
	if err := os.MkdirAll(a.PreviewDir, 0o700); err != nil {
		return "", fmt.Errorf("create preview directory: %w", err)
	}
	if err := os.Chmod(a.PreviewDir, 0o700); err != nil {
		return "", fmt.Errorf("protect preview directory: %w", err)
	}
	sum := sha256.Sum256([]byte(assetID))
	filename := hex.EncodeToString(sum[:8]) + extensionForContentType(contentType)
	path := filepath.Join(a.PreviewDir, filename)
	tmpPath := path + fmt.Sprintf(".%d.tmp", a.now().UnixNano())
	if err := os.WriteFile(tmpPath, body, 0o600); err != nil {
		return "", fmt.Errorf("write preview file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("protect preview file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("replace preview file: %w", err)
	}
	return path, nil
}

func extensionForContentType(contentType string) string {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/webp":
		return ".webp"
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	default:
		return ".bin"
	}
}

func (a *App) client(agentBaseURL string) *agentclient.Client {
	return agentclient.New(agentBaseURL, a.HTTPClient)
}

func (a *App) now() time.Time {
	if a.Now == nil {
		return time.Now().UTC()
	}
	return a.Now().UTC()
}

func userError(code string, message string, err error) error {
	return &Error{Code: code, Message: message, Err: err}
}

func mapAgentError(err error) error {
	if err == nil {
		return nil
	}
	var status *agentclient.StatusError
	if errors.As(err, &status) {
		code := status.Code
		if code == "" {
			code = fmt.Sprintf("agent_http_%d", status.StatusCode)
		}
		message := status.Message
		if message == "" {
			message = status.Error()
		}
		if status.StatusCode == http.StatusUnauthorized {
			code = "not_authenticated"
			message = "The saved session is no longer valid. Pair again."
		}
		return userError(code, message, err)
	}
	return err
}

func ErrorInfo(err error) (string, string) {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Code, appErr.Message
	}
	return "internal_error", err.Error()
}

func ErrorMessage(err error) string {
	code, message := ErrorInfo(err)
	if code == "" {
		return message
	}
	return code + ": " + message
}
