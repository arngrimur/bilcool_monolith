package testdb

// OptionsFunc configures an option.
type OptionsFunc func(mate *DBMate)

// WithProjectRoot sets the strategy for finding the project root. Defaults to
// GoModule.
func WithProjectRoot(pr ProjectRoot) OptionsFunc {
	return func(o *DBMate) {
		o.ProjectRoot = pr
	}
}
