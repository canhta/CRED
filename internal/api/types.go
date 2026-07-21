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
	Status           string `json:"status"`
	Version          string `json:"version"`
	Principal        string `json:"principal"`
	RegistrationOpen bool   `json:"registration_open"`
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

// UsageQuery is the query string of GET /api/usage.
type UsageQuery struct {
	Scopes int `json:"scopes" form:"scopes"`
}

// LimitStatus is one limit's window state: what has been used, the
// configured ceiling, and the remaining headroom before the control binds.
// Remaining reuses internal/limit.Decision's own sentinel (-1 means
// unlimited) and Ceiling <= 0 means disabled, rather than the API inventing
// a second "off"/"unlimited" convention on top of the one internal/limit
// already has — the frontend formats both, once, the same way
// `cred usage` already does in the terminal.
type LimitStatus struct {
	Window    string `json:"window"`
	Used      int    `json:"used"`
	Ceiling   int    `json:"ceiling"`
	Remaining int    `json:"remaining"`
	Allowed   bool   `json:"allowed"`
	Reason    string `json:"reason"`
}

// ScopeCost is one scope's inference cost since the report's cutoff — "which
// teams actually use this", the same report `cred usage` prints.
type ScopeCost struct {
	Scope        Scope `json:"scope"`
	Calls        int   `json:"calls"`
	InputTokens  int   `json:"input_tokens"`
	OutputTokens int   `json:"output_tokens"`
}

// ScopeGrowth is one scope's live-claim count against the growth ceiling, and
// how many claims the next prune pass would close.
type ScopeGrowth struct {
	Scope     Scope `json:"scope"`
	Live      int   `json:"live"`
	Ceiling   int   `json:"ceiling"`
	NextPrune int   `json:"next_prune"`
}

// UsageResponse is the body of GET /api/usage: the calling principal's limit
// headroom, its denied-contribution count, and the org-wide cost/growth
// report — the same counters and the same internal/limit decisions
// `cred usage` prints, so the console never shows a number the enforcement
// path didn't also compute.
type UsageResponse struct {
	Principal          string        `json:"principal"`
	Contribution       LimitStatus   `json:"contribution"`
	Cost               LimitStatus   `json:"cost"`
	InputTokensUsed    int           `json:"input_tokens_used"`
	InputTokensCeiling int           `json:"input_tokens_ceiling"`
	Recall             LimitStatus   `json:"recall"`
	DeniedWindow       string        `json:"denied_window"`
	Denied             int           `json:"denied"`
	CostByScope        []ScopeCost   `json:"cost_by_scope"`
	ScopeGrowth        []ScopeGrowth `json:"scope_growth"`
}

// RegisterRequest is the body of POST /api/auth/register.
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

// LoginRequest is the body of POST /api/auth/login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse is the body of a successful register or login.
type AuthResponse struct {
	Principal string `json:"principal"`
	Role      string `json:"role"`
}
