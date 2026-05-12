// Package matcher provides fuzzy string matching for subdomain resolution.
package matcher

// Matcher finds the closest valid subdomain for a given input.
type Matcher interface {
	// Match returns the best-matching subdomain from the known list.
	// Returns the match and a confidence score in [0.0, 1.0].
	// Returns empty string and 0 if no suitable match is found.
	Match(input string) (match string, score float64)
}

