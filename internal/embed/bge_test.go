package embed

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

// referenceVector is the first five components of the CLS-pooled, L2-normalized
// embedding of referenceText, as recorded in
// docs/research/spikes/go-embeddings-tokenizer.md where the same model was run
// under onnx-gomlx and cross-checked against ONNX Runtime at cosine 1.00000000.
//
// Pinning it here turns that spike result into a regression test: a change to
// the tokenizer, the pooling, or the normalization moves these numbers, and a
// silent move is exactly the failure shape this model choice was validated
// against.
var (
	referenceText   = "The quick brown fox jumps over the lazy dog."
	referenceVector = []float32{
		-0.10406199, -0.013690416, -0.009501907, 0.107154176, 0.010600818,
	}
)

// tolerance is float32 rounding. The spike measured a maximum per-element
// deviation of 1.4e-7 between this backend and ONNX Runtime.
const tolerance = 1e-6

func newTestEmbedder(t *testing.T) *BGE {
	t.Helper()
	path, err := ModelPath(t.Context(), "", false)
	if err != nil {
		t.Skipf("model not in the local cache; run `cred doctor` to fetch it: %v", err)
	}
	e, err := NewBGE(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, e.Close()) })
	return e
}

func TestEmbedMatchesTheRecordedReference(t *testing.T) {
	e := newTestEmbedder(t)
	vecs, err := e.Embed(t.Context(), []string{referenceText})
	require.NoError(t, err)
	require.Len(t, vecs, 1)
	require.Len(t, vecs[0], Dimensions)
	for i, want := range referenceVector {
		require.InDelta(t, want, vecs[0][i], tolerance, "component %d", i)
	}
}

func TestEmbedIsUnitNorm(t *testing.T) {
	e := newTestEmbedder(t)
	vecs, err := e.Embed(t.Context(), []string{
		referenceText,
		"SELECT * FROM claims WHERE embedding <=> $1 LIMIT 10",
		"",
	})
	require.NoError(t, err)
	require.Len(t, vecs, 3)
	for i, v := range vecs {
		var sum float64
		for _, f := range v {
			sum += float64(f) * float64(f)
		}
		// The database CHECK uses 1e-2; this is tighter so a drift shows up
		// here before it shows up as a constraint violation on write.
		require.InDelta(t, 1.0, math.Sqrt(sum), 1e-4, "vector %d is not unit norm", i)
	}
}

// TestBatchingDoesNotChangeVectors — padding to a shape bucket must not alter
// a result. If it does, the mask is being ignored and every batched write is
// subtly wrong while every single-item read looks fine.
func TestBatchingDoesNotChangeVectors(t *testing.T) {
	e := newTestEmbedder(t)
	alone, err := e.Embed(t.Context(), []string{referenceText})
	require.NoError(t, err)
	batched, err := e.Embed(t.Context(), []string{
		referenceText,
		"a much longer text that forces the batch into a wider shape bucket " +
			"than the reference sentence would have needed on its own, which " +
			"is the condition under which an ignored attention mask shows up",
	})
	require.NoError(t, err)
	for i := range alone[0] {
		require.InDelta(t, alone[0][i], batched[0][i], 1e-4, "component %d", i)
	}
}

func TestBucketFor(t *testing.T) {
	tests := []struct{ in, want int }{
		{1, 16}, {16, 16}, {17, 32}, {128, 128}, {129, 256}, {512, 512}, {9999, 512},
	}
	for _, tc := range tests {
		require.Equal(t, tc.want, bucketFor(tc.in), "bucketFor(%d)", tc.in)
	}
}

func BenchmarkEmbedQuery(b *testing.B) {
	path, err := ModelPath(b.Context(), "", false)
	if err != nil {
		b.Skip(err)
	}
	e, err := NewBGE(path)
	require.NoError(b, err)
	defer func() { require.NoError(b, e.Close()) }()

	// Warm the compiled graph so the benchmark measures the forward pass
	// rather than the one-off compile.
	_, err = e.Embed(b.Context(), []string{referenceText})
	require.NoError(b, err)

	b.ReportAllocs()
	for b.Loop() {
		if _, err := e.Embed(b.Context(), []string{referenceText}); err != nil {
			b.Fatal(err)
		}
	}
}
