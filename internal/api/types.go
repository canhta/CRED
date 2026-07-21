package api

// The types in this file are the console's wire contract. tygo generates the
// TypeScript mirror from them, so the JSON tags here are the field names the
// React app sees. Keep them clean and stable.

// ErrorResponse is the body of every non-2xx JSON response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// HealthResponse reports liveness and the resolved caller identity, so the
// console can show who it is acting as before any data loads.
type HealthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Principal string `json:"principal"`
}

// Scope narrows where a claim applies.
type Scope struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// Source locates the evidence a claim rests on within a repository.
type Source struct {
	Kind       string `json:"kind"`
	Repo       string `json:"repo"`
	Path       string `json:"path"`
	LineStart  int    `json:"line_start"`
	LineEnd    int    `json:"line_end"`
	SymbolPath string `json:"symbol_path"`
}

// ClaimListQuery is the query string of GET /api/claims.
type ClaimListQuery struct {
	Status     string `json:"status" form:"status"`
	ScopeKind  string `json:"scope_kind" form:"scope_kind"`
	ScopeValue string `json:"scope_value" form:"scope_value"`
	Limit      int    `json:"limit" form:"limit"`
	Offset     int    `json:"offset" form:"offset"`
}

// ClaimListItem is one row in the claims browser.
type ClaimListItem struct {
	ID            string  `json:"id"`
	Statement     string  `json:"statement"`
	Kind          string  `json:"kind"`
	Scope         Scope   `json:"scope"`
	Status        string  `json:"status"`
	Source        *Source `json:"source"`
	ContributedBy string  `json:"contributed_by"`
	RecordedAt    string  `json:"recorded_at"`
	ValidFrom     string  `json:"valid_from"`
	ValidUntil    string  `json:"valid_until"`
	SupersededAt  string  `json:"superseded_at"`
}

// ClaimListResponse is the body of GET /api/claims.
type ClaimListResponse struct {
	Claims []ClaimListItem `json:"claims"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
	Count  int             `json:"count"`
}

// EvidenceItem is one piece of evidence under a claim, with its anchor state.
type EvidenceItem struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Repo       string `json:"repo"`
	Path       string `json:"path"`
	LineStart  int    `json:"line_start"`
	LineEnd    int    `json:"line_end"`
	SymbolPath string `json:"symbol_path"`
	Anchor     string `json:"anchor"`
}

// ClaimDetail is the body of GET /api/claims/:id.
type ClaimDetail struct {
	ID            string         `json:"id"`
	Statement     string         `json:"statement"`
	Kind          string         `json:"kind"`
	Scope         Scope          `json:"scope"`
	Status        string         `json:"status"`
	Confidence    float64        `json:"confidence"`
	ContributedBy string         `json:"contributed_by"`
	SourceRepo    string         `json:"source_repo"`
	RecordedAt    string         `json:"recorded_at"`
	ValidFrom     string         `json:"valid_from"`
	ValidUntil    string         `json:"valid_until"`
	SupersededAt  string         `json:"superseded_at"`
	ExpiredReason string         `json:"expired_reason"`
	Evidence      []EvidenceItem `json:"evidence"`
}
