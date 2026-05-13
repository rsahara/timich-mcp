package agentclient

import "time"

type SearchRequest struct {
	Collection SearchCollectionRequest `json:"collection"`
	Page       SearchPageRequest       `json:"page"`
}

type SearchCollectionRequest struct {
	Kind    string        `json:"kind"`
	Query   *SearchQuery  `json:"query,omitempty"`
	Filters SearchFilters `json:"filters,omitempty"`
	Sort    *SearchSort   `json:"sort,omitempty"`
}

type SearchQuery struct {
	Text string `json:"text,omitempty"`
	Mode string `json:"mode,omitempty"`
}

type SearchFilters struct {
	MediaTypes []string          `json:"mediaTypes,omitempty"`
	CapturedAt *SearchCapturedAt `json:"capturedAt,omitempty"`
}

type SearchCapturedAt struct {
	From *string `json:"from,omitempty"`
	To   *string `json:"to,omitempty"`
}

type SearchSort struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

type SearchPageRequest struct {
	Index int `json:"index"`
	Size  int `json:"size"`
}

type SearchPage struct {
	CollectionKey string            `json:"collectionKey"`
	Page          SearchPageRequest `json:"page"`
	Items         []Asset           `json:"items"`
	Total         int               `json:"total"`
	TotalAccuracy string            `json:"totalAccuracy"`
	NextPageIndex *int              `json:"nextPageIndex,omitempty"`
	Boundary      *SearchBoundary   `json:"boundary,omitempty"`
	Resolved      SearchResolved    `json:"resolved"`
}

type Asset struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Filename   string    `json:"filename"`
	CapturedAt time.Time `json:"capturedAt"`
	Duration   *string   `json:"duration,omitempty"`
}

type SearchBoundary struct {
	Kind string `json:"kind"`
}

type SearchResolved struct {
	CollectionKind string     `json:"collectionKind"`
	QueryMode      string     `json:"queryMode"`
	Sort           SearchSort `json:"sort"`
	TimelineLike   bool       `json:"timelineLike"`
}

type SearchCapabilities struct {
	QueryModes    []string                 `json:"queryModes"`
	Filters       SearchFilterCapabilities `json:"filters"`
	Sorts         []SearchSortCapability   `json:"sorts"`
	TotalAccuracy []string                 `json:"totalAccuracy"`
	Page          SearchPageCapabilities   `json:"page"`
}

type SearchFilterCapabilities struct {
	MediaTypes []string `json:"mediaTypes"`
	CapturedAt bool     `json:"capturedAt"`
}

type SearchSortCapability struct {
	Field      string   `json:"field"`
	Directions []string `json:"directions"`
}

type SearchPageCapabilities struct {
	MaxSize int `json:"maxSize"`
}

type PreviewResponse struct {
	Body        []byte
	ContentType string
}
