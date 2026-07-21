// Package claim holds CRED's domain types: principals, claims, evidence, and
// the bi-temporal and access-control data they carry.
//
// This package is data only. The algebra over these types lives in
// internal/temporal and internal/acl, which are pure and which depguard
// forbids from importing a database driver. Keeping the types here and the
// decisions there is what stops access control from becoming a SQL predicate.
package claim

import "time"

// PrincipalID identifies a principal — a person, a team, an organization, or
// an agent.
//
// A standing check, run against CRED itself: grep the engine for the
// principal type; if it only appears in a client package, the retreat has
// already happened. This slice ships one principal, and the type still lives
// here, because retrofitting it later is the expensive move and every
// competitor that deferred it could not add it back.
type PrincipalID string

// PrincipalKind is the closed set of things that can hold a grant.
type PrincipalKind string

const (
	PrincipalUser  PrincipalKind = "user"
	PrincipalTeam  PrincipalKind = "team"
	PrincipalOrg   PrincipalKind = "org"
	PrincipalAgent PrincipalKind = "agent"
)

// Principal is an identity that recall is evaluated against.
type Principal struct {
	ID          PrincipalID
	Kind        PrincipalKind
	DisplayName string
}

// Grant is one entry in an access-control set.
//
// ExpiresAt is not optional in spirit: every ACL entry is required to carry a
// TTL, and a zero value means "no expiry", which internal/acl treats as valid
// forever. Stale permission data must deny rather than grant, so expiry is
// evaluated at recall against a caller-supplied clock, never cached.
type Grant struct {
	Principal PrincipalID
	ExpiresAt time.Time // zero means no expiry
}

// ACL is a set of grants. The empty ACL is reachable by nobody — treating it
// as public is the fail-open bug.
type ACL []Grant

// Kind determines a claim's validity semantics. Closed set.
type Kind string

const (
	KindConvention       Kind = "Convention"
	KindDecision         Kind = "Decision"
	KindConstraint       Kind = "Constraint"
	KindRejectedApproach Kind = "RejectedApproach"
	KindFailure          Kind = "Failure"
	KindReference        Kind = "Reference"
)

// ScopeKind is the granularity a claim applies at.
type ScopeKind string

const (
	ScopeOrg     ScopeKind = "organization"
	ScopeRepo    ScopeKind = "repository"
	ScopePath    ScopeKind = "path"
	ScopeService ScopeKind = "service"
)

// Scope narrows where a claim applies.
type Scope struct {
	Kind  ScopeKind
	Value string
}

// Interval is a half-open time interval [From, Until).
//
// Half-open throughout is what makes "no gaps and no overlaps" a single
// equality and eliminates the boundary off-by-one class entirely. A zero Until
// means open-ended.
type Interval struct {
	From  time.Time
	Until time.Time // zero means open-ended
}

// SourceKind describes what produced a piece of evidence.
type SourceKind string

const (
	SourceDocument    SourceKind = "document"
	SourceCode        SourceKind = "code"
	SourceAttestation SourceKind = "attestation"
)

// Evidence is what a claim rests on. A claim with no evidence cannot be
// written.
type Evidence struct {
	ID   string
	Kind SourceKind

	Repo      string
	Path      string
	LineStart int
	LineEnd   int

	// ExtractedText is the normalized source text, retained rather than
	// pointed at. A vectors-only store can never migrate to a new embedding
	// model, and that is decided at ingest, long before the first model swap.
	ExtractedText string

	// ContentSHA256 is the change detector. Re-seeding compares it and skips
	// unchanged chunks, which is what makes seeding idempotent. It is also tier 4
	// of the anchor ladder — the raw byte hash, diagnostic only.
	ContentSHA256 string

	// The semantic anchor, tiers 1–3. Computed at ingest by internal/anchor
	// from the whole source file, stored here, and re-resolved when the file
	// changes. Empty on attestations and on evidence written before anchoring
	// shipped — such rows are tier-4-only and re-anchoring leaves them untouched.
	// Hashes are hex.
	AnchorSymbolPath string // tier 1: heading path or symbol path
	AnchorNodeHash   string // tier 2: normalized enclosing-node hash
	AnchorWindowHash string // tier 3: normalized context-window hash

	AttestedBy PrincipalID
	AttestedAt time.Time

	Valid    Interval // when it was true in the world
	Recorded Interval // when the system knew it

	ACL ACL
}

// Claim is the atomic unit of knowledge: small, typed, independently
// expirable.
type Claim struct {
	ID        string
	Kind      Kind
	Statement string
	Scope     Scope

	Valid    Interval
	Recorded Interval

	SupersededBy string

	// SupersedeReason records why a claim was closed: a duplicate, a
	// contradiction, a human forget, a stale anchor, or a prune. Empty on a live
	// claim and on one that expired only because its valid interval elapsed.
	SupersedeReason string

	// Confidence is an explainable additive score, never an opaque posterior.
	Confidence float64

	SourceRepo       string
	ExtractedByModel string
	PromptVersion    string

	// ContributedBy is the principal a write is counted against. Empty on seeded
	// claims, which are exempt from the contribution quota by construction.
	ContributedBy PrincipalID

	ACL ACL

	// Evidence is populated by the store on load. An empty slice is a
	// bug, not a state.
	Evidence []Evidence
}
