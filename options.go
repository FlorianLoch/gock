package pgock

// Options carries per-request matching options.
type Options struct {
	// DisableRegexpHost, when true, treats the host passed to (*Transport).New
	// as a plain string rather than a regular expression.
	DisableRegexpHost bool
}
