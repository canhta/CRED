// Package embed turns text into vectors, in-process and in pure Go.
//
// The read path requires no API key, no provider choice, and no vector-store
// decision. Nothing in this category ships config-free except the systems that
// never call a model, and this package is the reason CRED can. That position
// is lost the first time a remote provider is added "just for now".
package embed

import "context"

// Embedder produces unit-norm vectors.
//
// This interface exists from the first commit, and that is deliberate rather
// than decorative. The pure-Go forward pass is 9-16x slower than ONNX Runtime
// and the gap widens with sequence length; the accepted answer is a build-tagged
// ONNX Runtime variant behind this interface, never a rewrite. An
// implementation added later would arrive as a refactor of every call site.
type Embedder interface {
	// Embed returns one unit-norm vector per input text, in order.
	Embed(ctx context.Context, texts []string) ([][]float32, error)

	// ModelName is written to embedding_model_id's row and filtered on at
	// read time. A model column that reads ignore is worthless.
	ModelName() string

	// Dimensions is the vector width.
	Dimensions() int

	// Close releases the model.
	Close() error
}
