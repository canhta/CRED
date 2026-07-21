package nominate_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/canhta/cred/internal/nominate"
	"github.com/stretchr/testify/require"
)

// The finish-reason normalization is the one correctness trap in the adapter:
// the Extractor gates truncation on a single canonical StopLength, and OpenAI
// spells truncation "length" while Anthropic spells it "max_tokens". If the
// adapter returned "length" verbatim, the gate would miss it and a truncated
// (valid-prefix) response would parse and pass — the exact L2 hole. This test
// pins that "length" is translated to StopLength.
func TestOpenAINormalizesTruncationReason(t *testing.T) {
	// The handler runs on another goroutine, so it records what it saw and the
	// main goroutine asserts (testifylint's go-require rule, and the correct
	// pattern — a failed require off the test goroutine does not stop the test).
	var gotAuth, gotFormat string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("authorization")
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		_ = json.Unmarshal(body, &req)
		if rf, ok := req["response_format"].(map[string]any); ok {
			gotFormat, _ = rf["type"].(string)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				"message":       map[string]any{"content": `{"candidates":[]}`},
				"finish_reason": "length",
			}},
			"usage": map[string]any{"prompt_tokens": 11, "completion_tokens": 3},
		})
	}))
	defer srv.Close()

	m, err := nominate.NewOpenAICompatModel("k", srv.URL, "deepseek-chat",
		nominate.WithOpenAIHTTPClient(srv.Client()))
	require.NoError(t, err)

	raw, stop, usage, err := m.Generate(context.Background(), "prompt", []byte(`{"type":"object"}`))
	require.NoError(t, err)
	require.Equal(t, "Bearer k", gotAuth)
	// JSON mode must be requested, and the schema rides in the prompt, not a
	// wire parameter — json_object enforces syntax, never the schema.
	require.Equal(t, "json_object", gotFormat)
	require.JSONEq(t, `{"candidates":[]}`, string(raw))
	require.Equal(t, nominate.StopLength, stop, "OpenAI 'length' must normalize to the canonical StopLength")
	require.Equal(t, 11, usage.InputTokens)
	require.Equal(t, 3, usage.OutputTokens)
}

// A non-200 with an error body surfaces the provider's message rather than a
// bare status, and still reports the tokens the failed call spent.
func TestOpenAISurfacesErrorBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"type": "authentication_error", "message": "bad key"},
		})
	}))
	defer srv.Close()

	m, err := nominate.NewOpenAICompatModel("k", srv.URL, "deepseek-chat",
		nominate.WithOpenAIHTTPClient(srv.Client()))
	require.NoError(t, err)

	_, _, _, err = m.Generate(context.Background(), "p", []byte(`{}`))
	require.Error(t, err)
	require.Contains(t, err.Error(), "bad key")
}

func TestOpenAIRequiresKeyBaseURLAndModel(t *testing.T) {
	_, err := nominate.NewOpenAICompatModel("", "https://x", "m")
	require.Error(t, err)
	_, err = nominate.NewOpenAICompatModel("k", "", "m")
	require.Error(t, err)
	_, err = nominate.NewOpenAICompatModel("k", "https://x", "")
	require.Error(t, err)
}
