package sql

import "embed"

// content holds our static web server content.
//
//go:embed schema/*.sql
var MigrationsFS embed.FS
