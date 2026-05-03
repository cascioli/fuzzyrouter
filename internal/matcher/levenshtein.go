package matcher

// LevenshteinMatcher implements Matcher using normalized Levenshtein distance.
type LevenshteinMatcher struct {
	// candidates is the list of valid subdomains to match against.
	candidates []string
	// threshold is the minimum score [0,1] to accept a match.
	threshold float64
}

// NewLevenshtein creates a LevenshteinMatcher.
// threshold controls how strict matching is: 0.0 = accept anything, 1.0 = exact only.
func NewLevenshtein(candidates []string, threshold float64) *LevenshteinMatcher {
	return &LevenshteinMatcher{
		candidates: candidates,
		threshold:  threshold,
	}
}

// Match returns the candidate with highest similarity score above threshold.
func (m *LevenshteinMatcher) Match(input string) (string, float64) {
	var bestMatch string
	var bestScore float64

	for _, candidate := range m.candidates {
		score := similarity(input, candidate)
		if score > bestScore {
			bestScore = score
			bestMatch = candidate
		}
	}

	if bestScore < m.threshold {
		return "", 0
	}
	return bestMatch, bestScore
}

// similarity returns a normalized similarity score in [0.0, 1.0] between a and b.
// 1.0 means identical, 0.0 means completely different.
func similarity(a, b string) float64 {
	dist := levenshtein(a, b)
	maxLen := max(len(a), len(b))
	if maxLen == 0 {
		return 1.0
	}
	return 1.0 - float64(dist)/float64(maxLen)
}

// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	la, lb := len(ra), len(rb)

	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	// Use two rows to save memory: previous and current.
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)

	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[j] = min3(
				curr[j-1]+1,   // insertion
				prev[j]+1,     // deletion
				prev[j-1]+cost, // substitution
			)
		}
		prev, curr = curr, prev
	}

	return prev[lb]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
