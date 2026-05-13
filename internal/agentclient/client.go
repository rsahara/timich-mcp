package agentclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxPreviewBytes = 10 << 20

type Client struct {
	BaseURL string
	HTTP    *http.Client
}

type SessionBundle struct {
	AccessToken            string    `json:"accessToken"`
	RefreshToken           string    `json:"refreshToken"`
	AgentID                string    `json:"agentId"`
	AgentName              string    `json:"agentName"`
	DeviceID               string    `json:"deviceId"`
	BaseURL                string    `json:"baseURL"`
	AccessMode             string    `json:"accessMode"`
	AccessTokenExpiresAt   time.Time `json:"accessTokenExpiresAt"`
	RefreshTokenExpiresAt  time.Time `json:"refreshTokenExpiresAt"`
	CertificateFingerprint string    `json:"certificateFingerprint,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type StatusError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *StatusError) Error() string {
	if e == nil {
		return ""
	}
	if e.Code == "" && e.Message == "" {
		return fmt.Sprintf("agent request failed with status %d", e.StatusCode)
	}
	if e.Message == "" {
		return fmt.Sprintf("%s (status %d)", e.Code, e.StatusCode)
	}
	if e.Code == "" {
		return fmt.Sprintf("%s (status %d)", e.Message, e.StatusCode)
	}
	return fmt.Sprintf("%s: %s (status %d)", e.Code, e.Message, e.StatusCode)
}

func IsUnauthorized(err error) bool {
	var status *StatusError
	return errors.As(err, &status) && status.StatusCode == http.StatusUnauthorized
}

func NormalizeBaseURL(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("agent URL is required")
	}
	if !strings.Contains(value, "://") {
		value = "http://" + value
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("parse agent URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("agent URL scheme must be http or https")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("agent URL host is required")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/"), nil
}

func New(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    httpClient,
	}
}

func (c *Client) Pair(ctx context.Context, pairingCode string, deviceName string) (SessionBundle, error) {
	var out SessionBundle
	err := c.postJSON(ctx, "/v1/pairing/redeem", "", map[string]string{
		"pairingCode": pairingCode,
		"deviceName":  deviceName,
	}, &out)
	return out, err
}

func (c *Client) Refresh(ctx context.Context, refreshToken string) (SessionBundle, error) {
	var out SessionBundle
	err := c.postJSON(ctx, "/v1/session/refresh", "", map[string]string{
		"refreshToken": refreshToken,
	}, &out)
	return out, err
}

func (c *Client) Capabilities(ctx context.Context, accessToken string) (SearchCapabilities, error) {
	var out SearchCapabilities
	err := c.doJSON(ctx, http.MethodGet, "/v1/assets/search/capabilities", accessToken, nil, &out)
	return out, err
}

func (c *Client) Search(ctx context.Context, accessToken string, search SearchRequest) (SearchPage, error) {
	var out SearchPage
	err := c.postJSON(ctx, "/v1/assets/search", accessToken, search, &out)
	return out, err
}

func (c *Client) Preview(ctx context.Context, accessToken string, assetID string) (PreviewResponse, error) {
	endpoint := "/v1/assets/" + url.PathEscape(assetID) + "/preview"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+endpoint, nil)
	if err != nil {
		return PreviewResponse{}, err
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)

	response, err := c.HTTP.Do(request)
	if err != nil {
		return PreviewResponse{}, fmt.Errorf("perform preview request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return PreviewResponse{}, decodeStatusError(response)
	}
	body, err := readAtMost(response.Body, maxPreviewBytes)
	if err != nil {
		return PreviewResponse{}, fmt.Errorf("read preview response: %w", err)
	}
	return PreviewResponse{
		Body:        body,
		ContentType: mediaTypeOnly(response.Header.Get("Content-Type")),
	}, nil
}

func (c *Client) postJSON(ctx context.Context, path string, accessToken string, in any, out any) error {
	return c.doJSON(ctx, http.MethodPost, path, accessToken, in, out)
}

func (c *Client) doJSON(ctx context.Context, method string, path string, accessToken string, in any, out any) error {
	var body io.Reader
	if in != nil {
		raw, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		body = bytes.NewReader(raw)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if accessToken != "" {
		request.Header.Set("Authorization", "Bearer "+accessToken)
	}
	response, err := c.HTTP.Do(request)
	if err != nil {
		return fmt.Errorf("perform request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return decodeStatusError(response)
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(response.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func decodeStatusError(response *http.Response) error {
	var payload ErrorResponse
	limited := io.LimitReader(response.Body, 1<<20)
	_ = json.NewDecoder(limited).Decode(&payload)
	if payload.Message == "" {
		payload.Message = http.StatusText(response.StatusCode)
	}
	return &StatusError{
		StatusCode: response.StatusCode,
		Code:       payload.Error,
		Message:    payload.Message,
	}
}

func readAtMost(reader io.Reader, maxBytes int64) ([]byte, error) {
	limited := io.LimitReader(reader, maxBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("response exceeded %d bytes", maxBytes)
	}
	return body, nil
}

func mediaTypeOnly(value string) string {
	if i := strings.Index(value, ";"); i >= 0 {
		value = value[:i]
	}
	return strings.ToLower(strings.TrimSpace(value))
}
