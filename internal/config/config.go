// Package config resolves configuration from the environment.
//
// Every variable is prefixed CRED_, except DATABASE_URL, which is a de-facto
// standard users already expect. Every one has a working default, because a
// .env.example you must copy first is a step, and steps cost users.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/canhta/cred/internal/limit"
)

// DefaultDatabaseURL matches the compose file, so `cred` run beside
// `docker compose up` needs no configuration at all. The credential in it is
// the compose file's development credential, published on purpose so that the
// first run needs no configuration: it is not a secret, and there is nothing
// behind it but an empty local database.
const DefaultDatabaseURL = "postgres://cred:cred@127.0.0.1:5433/cred?sslmode=disable" //nolint:gosec // G101: documented development credential, not a secret

// Config is the resolved configuration.
type Config struct {
	// DatabaseURL is the one datastore. L7: relational, vector and full-text
	// all live here, and no second datastore may be added without removing
	// another.
	DatabaseURL string

	// ModelDir holds the ONNX model. Empty means the user cache directory.
	ModelDir string

	// AllowModelDownload gates the first-run fetch. Air-gapped deployments set
	// it false and bake the model into the image layer instead.
	AllowModelDownload bool

	// Principal is the identity recall is evaluated against. One principal in
	// this slice; the field exists because the alternative is threading it in
	// later through every call site.
	Principal string

	// LogLevel is debug, info, warn or error.
	LogLevel string

	// LogFormat is json or text. Text is the default for a CLI a human reads;
	// the server switches to json.
	LogFormat string

	// AutoCapture gates the automatic write path. It defaults to ON, matching
	// the shipped Mem0 pattern (D-017): the trigger enqueues a nomination job,
	// which a worker extracts off the turn. Opt out with CRED_AUTO_CAPTURE=false.
	// This gates only automatic *nomination*; explicit `remember` is unaffected,
	// and reads never touch it.
	AutoCapture bool

	// LLMAPIKey is the key the automatic write path needs to call the model. It
	// is deliberately absent from the read path and the explicit attestation
	// path, both of which require no key. Empty is valid everywhere except the
	// curate worker, which fails loudly if it must nominate without one.
	LLMAPIKey string

	// LLMModel is the model id the nominator uses. Defaults to the current
	// most-capable Anthropic model; overridable for cost or availability.
	LLMModel string

	// LLMBaseURL selects the provider dialect. Empty means Anthropic (Claude).
	// Any value routes to the OpenAI-compatible adapter, which covers OpenAI,
	// DeepSeek, and self-hosted servers (vLLM, Ollama, LM Studio) alike — they
	// all speak /chat/completions. Examples: https://api.deepseek.com,
	// https://api.openai.com/v1, http://localhost:11434/v1.
	LLMBaseURL string

	// Limits is the usage-and-limits policy (PRD 8). It ships with working
	// defaults (limit.Defaults), so the four controls are on out of the box with
	// no configuration — a limit that has to be configured to exist is off on
	// first run. Each ceiling is overridable through a CRED_* variable; a
	// non-positive override disables that one control.
	Limits limit.Config
}

// Load resolves configuration from the environment.
func Load() (Config, error) {
	c := Config{
		DatabaseURL:        env("DATABASE_URL", DefaultDatabaseURL),
		ModelDir:           env("CRED_MODEL_DIR", ""),
		AllowModelDownload: true,
		Principal:          env("CRED_PRINCIPAL", "local"),
		LogLevel:           env("CRED_LOG_LEVEL", "info"),
		LogFormat:          env("CRED_LOG_FORMAT", "text"),
		AutoCapture:        true,
		// One key variable for every provider. A provider-specific fallback
		// (ANTHROPIC_API_KEY) would privilege one dialect over the others now
		// that OpenAI, DeepSeek, and self-hosted are equal citizens.
		LLMAPIKey:  env("CRED_LLM_API_KEY", ""),
		LLMModel:   env("CRED_LLM_MODEL", ""),
		LLMBaseURL: env("CRED_LLM_BASE_URL", ""),
	}

	if v, ok := os.LookupEnv("CRED_ALLOW_MODEL_DOWNLOAD"); ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf(
				"CRED_ALLOW_MODEL_DOWNLOAD=%q is not a boolean", v)
		}
		c.AllowModelDownload = b
	}
	if v, ok := os.LookupEnv("CRED_AUTO_CAPTURE"); ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf("CRED_AUTO_CAPTURE=%q is not a boolean", v)
		}
		c.AutoCapture = b
	}

	c.Limits = limit.Defaults()
	if err := applyLimitOverrides(&c.Limits); err != nil {
		return Config{}, err
	}
	return c, nil
}

// applyLimitOverrides lets an operator retune any one section-8 ceiling without
// a config file. Every one is optional; absent leaves the default in place.
func applyLimitOverrides(l *limit.Config) error {
	for _, o := range []struct {
		key string
		dst *int
	}{
		{"CRED_CONTRIBUTION_QUOTA", &l.ContributionQuota},
		{"CRED_COST_MAX_CALLS", &l.MaxInferenceCalls},
		{"CRED_COST_MAX_TOKENS", &l.MaxInputTokens},
		{"CRED_RECALL_RATE", &l.RecallRate},
		{"CRED_RECALL_MAX_CLAIMS", &l.MaxPackageClaims},
		{"CRED_SCOPE_CLAIM_CEILING", &l.ScopeClaimCeiling},
	} {
		if v, ok := os.LookupEnv(o.key); ok {
			n, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("%s=%q is not an integer", o.key, v)
			}
			*o.dst = n
		}
	}
	for _, o := range []struct {
		key string
		dst *time.Duration
	}{
		{"CRED_CONTRIBUTION_WINDOW", &l.ContributionWindow},
		{"CRED_COST_WINDOW", &l.CostWindow},
		{"CRED_RECALL_WINDOW", &l.RecallWindow},
	} {
		if v, ok := os.LookupEnv(o.key); ok {
			d, err := time.ParseDuration(v)
			if err != nil {
				return fmt.Errorf("%s=%q is not a duration", o.key, v)
			}
			*o.dst = d
		}
	}
	return nil
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
