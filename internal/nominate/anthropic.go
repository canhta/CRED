package nominate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// The Anthropic adapter is hand-rolled against the raw Messages API rather than
// the official SDK, on purpose. go.md rule 9 prefers copying thirty lines over
// a module, and the accepted dependency set does not include an LLM SDK; the
// surface CRED needs is one POST with a JSON-schema output format and a stop
// reason back. The worker-ops spike also warns that the Stainless SDKs ship
// breaking changes in *minor* releases weekly — a standing tax a solo
// maintainer should not take on for this little API. If a second provider is
// ever added, it is a second Model, not a framework.
//
// Structured output uses output_config.format with a json_schema (the current
// parameter; the older top-level output_format is deprecated). No provider
// validates the schema server-side, so the result is still validated in Valid —
// this parameter is a hint, never a contract.

const (
	anthropicURL     = "https://api.anthropic.com/v1/messages"
	anthropicVersion = "2023-06-01"
	// DefaultModel is Anthropic's current most-capable model. Overridable via
	// CRED_LLM_MODEL for cost or availability reasons.
	DefaultModel = "claude-opus-4-8"
	// defaultMaxTokens bounds one nomination. Extraction output is a small JSON
	// array; a generous ceiling still leaves truncation (StopLength) as the
	// signal that the source was too large for one call.
	defaultMaxTokens = 2048
)

// AnthropicModel calls the Anthropic Messages API. It is the production Model;
// tests use a stub or the Fake nominator and never need a key.
type AnthropicModel struct {
	apiKey    string
	model     string
	maxTokens int
	http      *http.Client
}

// AnthropicOption configures an AnthropicModel.
type AnthropicOption func(*AnthropicModel)

// WithModel overrides the model id.
func WithModel(id string) AnthropicOption {
	return func(m *AnthropicModel) {
		if id != "" {
			m.model = id
		}
	}
}

// WithHTTPClient overrides the HTTP client (its Timeout is the request budget).
func WithHTTPClient(c *http.Client) AnthropicOption {
	return func(m *AnthropicModel) { m.http = c }
}

// NewAnthropicModel builds a Model. An empty key is an error rather than a
// deferred failure: the write path requires a key, and saying so at construction
// is clearer than a 401 buried in a River retry.
func NewAnthropicModel(apiKey string, opts ...AnthropicOption) (*AnthropicModel, error) {
	if apiKey == "" {
		return nil, ErrNoKey
	}
	m := &AnthropicModel{
		apiKey:    apiKey,
		model:     DefaultModel,
		maxTokens: defaultMaxTokens,
		http:      &http.Client{Timeout: 60 * time.Second},
	}
	for _, o := range opts {
		o(m)
	}
	return m, nil
}

type anthropicRequest struct {
	Model        string             `json:"model"`
	MaxTokens    int                `json:"max_tokens"`
	Messages     []anthropicMessage `json:"messages"`
	OutputConfig *outputConfig      `json:"output_config,omitempty"`
}

type outputConfig struct {
	Format outputFormat `json:"format"`
}

type outputFormat struct {
	Type   string          `json:"type"`
	Schema json.RawMessage `json:"schema"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// Generate implements Model.
func (m *AnthropicModel) Generate(ctx context.Context, prompt string, schema []byte) ([]byte, string, Usage, error) {
	body, err := json.Marshal(anthropicRequest{
		Model:     m.model,
		MaxTokens: m.maxTokens,
		Messages:  []anthropicMessage{{Role: "user", Content: prompt}},
		OutputConfig: &outputConfig{Format: outputFormat{
			Type:   "json_schema",
			Schema: schema,
		}},
	})
	if err != nil {
		return nil, "", Usage{}, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicURL, bytes.NewReader(body))
	if err != nil {
		return nil, "", Usage{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", m.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)

	resp, err := m.http.Do(req)
	if err != nil {
		return nil, "", Usage{}, fmt.Errorf("call anthropic: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var decoded anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, "", Usage{}, fmt.Errorf("decode response (status %d): %w", resp.StatusCode, err)
	}
	usage := Usage{InputTokens: decoded.Usage.InputTokens, OutputTokens: decoded.Usage.OutputTokens}

	if resp.StatusCode != http.StatusOK {
		if decoded.Error != nil {
			// The message may echo request content; it never echoes stored
			// claim text, so it is safe to surface here. Callers log identifiers
			// and counts, not this string, into any shared sink.
			return nil, "", usage, fmt.Errorf("anthropic %d %s: %s",
				resp.StatusCode, decoded.Error.Type, decoded.Error.Message)
		}
		return nil, "", usage, fmt.Errorf("anthropic returned status %d", resp.StatusCode)
	}

	// With output_config.format the model returns the JSON as a single text
	// block. Concatenate any text blocks, ignoring non-text (e.g. thinking).
	var out bytes.Buffer
	for _, c := range decoded.Content {
		if c.Type == "text" {
			out.WriteString(c.Text)
		}
	}
	return out.Bytes(), decoded.StopReason, usage, nil
}

var _ Model = (*AnthropicModel)(nil)
