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
	}

	if v, ok := os.LookupEnv("CRED_ALLOW_MODEL_DOWNLOAD"); ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf(
				"CRED_ALLOW_MODEL_DOWNLOAD=%q is not a boolean", v)
		}
		c.AllowModelDownload = b
	}
	return c, nil
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
