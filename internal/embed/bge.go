package embed

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gomlx/gomlx/backends"
	_ "github.com/gomlx/gomlx/backends/simplego" // pure Go, CGO_ENABLED=0
	"github.com/gomlx/gomlx/pkg/core/graph"
	mlcontext "github.com/gomlx/gomlx/pkg/ml/context"
	"github.com/gomlx/onnx-gomlx/onnx"
	onnxparser "github.com/gomlx/onnx-gomlx/onnx/parser"

	"github.com/canhta/cred/internal/embed/wordpiece"
)

// ModelName is the only embedding model this slice ships.
const ModelName = "bge-small-en-v1.5"

// Dimensions is bge-small-en-v1.5's output width.
const Dimensions = 384

// modelURL and modelSHA256 pin the artifact. An unpinned model download is a
// supply-chain hole, and the hash is checked on every load, not only on
// download — a truncated download nearly became a fabricated upstream defect
// once already on this project.
const (
	modelURL    = "https://huggingface.co/BAAI/bge-small-en-v1.5/resolve/main/onnx/model.onnx"
	modelSHA256 = "828e1496d7fabb79cfa4dcd84fa38625c0d3d21da474a00f08db0f559940cf35"
	modelBytes  = 133093490
)

// sequenceBuckets bound how many distinct graph shapes are compiled. Every new
// input length would otherwise trigger a recompile, and a recompile on the
// recall path costs far more than the padding does.
var sequenceBuckets = []int{16, 32, 64, 128, 256, wordpiece.MaxSequenceLength}

// BGE is a pure-Go embedder over bge-small-en-v1.5.
//
// Safe for concurrent use: Embed serializes on the executor, which gomlx does
// not document as concurrency-safe.
type BGE struct {
	tok     *wordpiece.Tokenizer
	model   onnx.Model
	ctx     *mlcontext.Context
	backend backends.Backend
	exec    *mlcontext.Exec

	mu sync.Mutex
}

// NewBGE loads the model from modelPath.
func NewBGE(modelPath string) (*BGE, error) {
	if err := verifySHA256(modelPath, modelSHA256); err != nil {
		return nil, err
	}

	tok, err := wordpiece.New()
	if err != nil {
		return nil, fmt.Errorf("tokenizer: %w", err)
	}

	model, err := onnxparser.ParseFile(modelPath)
	if err != nil {
		return nil, fmt.Errorf("parse onnx model: %w", err)
	}

	mlCtx := mlcontext.New()
	if verr := model.VariablesToContext(mlCtx); verr != nil {
		return nil, fmt.Errorf("load model weights: %w", verr)
	}

	backend, err := backends.New()
	if err != nil {
		return nil, fmt.Errorf("gomlx backend: %w", err)
	}

	exec, err := mlcontext.NewExec(backend, mlCtx,
		func(ctx *mlcontext.Context, inputs []*graph.Node) *graph.Node {
			out := model.CallGraph(ctx, inputs[0].Graph(), map[string]*graph.Node{
				"input_ids":      inputs[0],
				"attention_mask": inputs[1],
				"token_type_ids": inputs[2],
			})
			// CLS pooling, then L2 normalization. Vectors are normalized at
			// write time so inner product is the distance and the norm is a
			// database CHECK.
			cls := graph.Squeeze(graph.Slice(out[0],
				graph.AxisRange(), graph.AxisRange(0, 1), graph.AxisRange()), 1)
			norm := graph.Sqrt(graph.ReduceAndKeep(
				graph.Mul(cls, cls), graph.ReduceSum, -1))
			return graph.Div(cls, norm)
		})
	if err != nil {
		return nil, fmt.Errorf("compile graph: %w", err)
	}

	return &BGE{tok: tok, model: model, ctx: mlCtx, backend: backend, exec: exec}, nil
}

// ModelName implements Embedder.
func (b *BGE) ModelName() string { return ModelName }

// Dimensions implements Embedder.
func (b *BGE) Dimensions() int { return Dimensions }

// Close implements Embedder.
func (b *BGE) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.model != nil {
		if err := b.model.Close(); err != nil {
			return err
		}
		b.model = nil
	}
	return nil
}

// Embed implements Embedder.
func (b *BGE) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	ids := make([][]int32, len(texts))
	longest := 1
	for i, t := range texts {
		ids[i] = b.tok.Encode(t)
		if len(ids[i]) > longest {
			longest = len(ids[i])
		}
	}
	width := bucketFor(longest)

	inputIDs := make([][]int64, len(texts))
	mask := make([][]int64, len(texts))
	types := make([][]int64, len(texts))
	for i, seq := range ids {
		inputIDs[i] = make([]int64, width)
		mask[i] = make([]int64, width)
		types[i] = make([]int64, width)
		for j, id := range seq {
			inputIDs[i][j] = int64(id)
			mask[i][j] = 1
		}
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	outs, err := b.exec.Exec(inputIDs, mask, types)
	if err != nil {
		return nil, fmt.Errorf("forward pass: %w", err)
	}
	defer func() {
		for _, o := range outs {
			// Finalization frees device buffers. A failure here leaks memory
			// rather than corrupting a result, so it is logged by the caller's
			// pressure, not returned.
			_ = o.FinalizeAll()
		}
	}()

	raw, ok := outs[0].Value().([][]float32)
	if !ok {
		return nil, fmt.Errorf("forward pass returned %T, want [][]float32", outs[0].Value())
	}
	vecs := make([][]float32, len(raw))
	for i, v := range raw {
		vecs[i] = renormalize(v)
	}
	return vecs, nil
}

func bucketFor(n int) int {
	for _, b := range sequenceBuckets {
		if n <= b {
			return b
		}
	}
	return wordpiece.MaxSequenceLength
}

// renormalize corrects float32 drift so the database norm CHECK is not a
// coin flip on the tolerance boundary.
func renormalize(v []float32) []float32 {
	var sum float64
	for _, f := range v {
		sum += float64(f) * float64(f)
	}
	n := math.Sqrt(sum)
	if n == 0 {
		return v
	}
	out := make([]float32, len(v))
	for i, f := range v {
		out[i] = float32(float64(f) / n)
	}
	return out
}

// ModelPath resolves the model file, downloading it if absent.
//
// The model is never embedded in the binary: it would land in the data
// segment, be resident, and be re-downloaded on every patch release.
func ModelPath(ctx context.Context, dir string, allowDownload bool) (string, error) {
	if dir == "" {
		cache, err := os.UserCacheDir()
		if err != nil {
			return "", fmt.Errorf("resolve cache dir: %w", err)
		}
		dir = filepath.Join(cache, "cred", "models", ModelName)
	}
	path := filepath.Join(dir, "model.onnx")

	if err := verifySHA256(path, modelSHA256); err == nil {
		return path, nil
	}
	if !allowDownload {
		return "", fmt.Errorf("model not present at %s and downloading is disabled", path)
	}
	if err := download(ctx, modelURL, path, modelSHA256); err != nil {
		return "", err
	}
	return path, nil
}

func verifySHA256(path, want string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hash %s: %w", path, err)
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		return fmt.Errorf("model at %s has hash %s, expected %s", path, got, want)
	}
	return nil
}

func download(ctx context.Context, url, dest, want string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("create model dir: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download model: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download model: %s", resp.Status)
	}

	tmp := dest + ".partial"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create %s: %w", tmp, err)
	}
	n, err := io.Copy(f, resp.Body)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return fmt.Errorf("write model: %w", err)
	}
	// Size-check before hashing so a truncated transfer reports as truncated
	// rather than as a corrupt artifact.
	if n != modelBytes {
		return fmt.Errorf("model download truncated: %d bytes, expected %d", n, modelBytes)
	}
	if err := verifySHA256(tmp, want); err != nil {
		return err
	}
	return os.Rename(tmp, dest)
}
