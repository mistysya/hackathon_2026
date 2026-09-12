package backend

import _ "embed"

// SchemaSQL is the single, frozen SQLite schema used by the application.
//
//go:embed schema.sql
var SchemaSQL string
