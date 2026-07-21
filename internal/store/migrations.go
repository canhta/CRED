// Package store carries the schema. The only package that may import a
// database driver is internal/store/pg; this one holds the migrations and
// embeds them so the single binary carries its own schema.
package store

import "embed"

// Migrations are embedded so `cred migrate` needs nothing on disk beside the
// binary.
//
//go:embed migrations/*.sql
var Migrations embed.FS

// MigrationsDir is the path inside Migrations that goose is pointed at.
const MigrationsDir = "migrations"
