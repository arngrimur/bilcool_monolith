package testdb

import (
	"embed"
)

// OptionsFunc configures an option.
type OptionsFunc func(mate *DBMate)

// WithProjectRoot sets the strategy for finding the project root. Defaults to
// GoModule.
func WithProjectRoot(pr ProjectRoot) OptionsFunc {
	return func(dbMate *DBMate) {
		dbMate.ProjectRoot = pr
	}
}

func WithEmbeddedFs(fs embed.FS) OptionsFunc {
	return func(dbMate *DBMate) {
		dbMate.Fs = fs
		dbMate.UseFs = true
	}
}
