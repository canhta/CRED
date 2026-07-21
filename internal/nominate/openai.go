package nominate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// OpenAICompatModel calls any OpenAI-compatible /chat/completions endpoint —
// DeepSeek, OpenAI, Together, a local vLLM, anything speaking that dialect. A
// third provider is a third Model, not a framework.
//
// Two things differ from Anthropic. First, the schema is not a request
// parameter: JSON mode here (response_format {type: json_object}) forces
// syntactically valid JSON but enforces no schema, so the schema rides in the
// prompt and the local Valid check remains the real contract. Second,
// truncation is spelled "length" here and "max_tokens" at Anthropic; this Model
// normalizes to StopLength so the Extractor's single truncation check holds for
// every provider. Returning the raw "length" would let a truncated response — a
// valid JSON prefix that parses and passes — slip through as if complete.
type OpenAICompatModel struct {
	apiKey  string
	baseURL string
	model   string
	// maxTokens bounds the candidate array. A truncated response is discarded
	// (StopLength), so the ceiling is generous rather than tight.
	maxTokens int
	http      *http.Client
}

// OpenAIOption configures an OpenAICompatModel.
type OpenAIOption func(*OpenAICompatModel)

// WithOpenAIModel overrides the model id.
func WithOpenAIModel(id string) OpenAIOption {
	return func(m *OpenAICompatModel) {
		if id != "" {
			m.model = id
		}
	}
}

// WithOpenAIHTTPClient overrides the HTTP client (tests inject a fake).
func WithOpenAIHTTPClient(c *http.Client) OpenAIOption {
	return func(m *OpenAICompatModel) { m.http = c }
}

// NewOpenAICompatModel builds a Model against baseURL (e.g.
// "https://api.deepseek.com"). An empty key or base URL is an error rather than
// a model that fails only on first use.
func NewOpenAICompatModel(apiKey, baseURL, model string, opts ...OpenAIOption) (*OpenAICompatModel, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("nominate: OpenAI-compatible model requires an API key")
	}
	if baseURL == "" {
		return nil, fmt.Errorf("nominate: OpenAI-compatible model requires a base URL")
	}
	m := &OpenAICompatModel{
		apiKey:    apiKey,
		baseURL:   baseURL,
		model:     model,
		maxTokens: defaultMaxTokens,
		http:      &http.Client{Timeout: 60 * time.Second},
	}
	for _, o := range opts {
		o(m)
	}
	if m.model == "" {
		return nil, fmt.Errorf("nominate: OpenAI-compatible model requires a model id")
	}
	return m, nil
}

type openAIRequest struct {
	Model          string          `json:"model"`
	MaxTokens      int             `json:"max_tokens"`
	Messages       []openAIMessage `json:"messages"`
	ResponseFormat openAIRespFmt   `json:"response_format"`
	// Deterministic-as-possible: nomination is an extraction, not a
	// brainstorm. Temperature 0 is honored by DeepSeek and OpenAI alike.
	Temperature float64 `json:"temperature"`
}

type openAIRespFmt struct {
	Type string `json:"type"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// Generate implements Model.
func (m *OpenAICompatModel) Generate(ctx context.Context, prompt string, schema []byte) ([]byte, string, Usage, error) {
	// JSON mode enforces valid JSON but not the schema, and the docs warn that
	// without an explicit JSON instruction the model can emit unbounded
	// whitespace. So the schema and the instruction ride in the prompt.
	sys := fmt.Sprintf(
		"Respond with a single JSON object that conforms to this JSON Schema. "+
			"Output only the JSON object, no prose.\n\nSchema:\n%s", schema)

	body, err := json.Marshal(openAIRequest{
		Model:          m.model,
		MaxTokens:      m.maxTokens,
		ResponseFormat: openAIRespFmt{Type: "json_object"},
		Temperature:    0,
		Messages: []openAIMessage{
			{Role: "system", Content: sys},
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, "", Usage{}, fmt.Errorf("marshal request: %w", err)
	}

	url := m.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, "", Usage{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", "Bearer "+m.apiKey)

	resp, err := m.http.Do(req)
	if err != nil {
		return nil, "", Usage{}, fmt.Errorf("call model: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var decoded openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, "", Usage{}, fmt.Errorf("decode response (status %d): %w", resp.StatusCode, err)
	}
	usage := Usage{
		InputTokens:  decoded.Usage.PromptTokens,
		OutputTokens: decoded.Usage.CompletionTokens,
	}

	if resp.StatusCode != http.StatusOK {
		if decoded.Error != nil {
			return nil, "", usage, fmt.Errorf("model %d %s: %s",
				resp.StatusCode, decoded.Error.Type, decoded.Error.Message)
		}
		return nil, "", usage, fmt.Errorf("model returned status %d", resp.StatusCode)
	}
	if len(decoded.Choices) == 0 {
		return nil, "", usage, fmt.Errorf("model returned no choices")
	}

	choice := decoded.Choices[0]
	// Normalize truncation to the canonical StopLength so the Extractor's
	// provider-agnostic check catches it. "length" is OpenAI's spelling;
	// anything else passes through unchanged for diagnostics.
	stop := choice.FinishReason
	if stop == "length" {
		stop = StopLength
	}
	return []byte(choice.Message.Content), stop, usage, nil
}

var _ Model = (*OpenAICompatModel)(nil)
