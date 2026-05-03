package matcher

import (
	"testing"
)

var candidates = []string{"app", "admin", "api", "auth", "blog", "shop", "docs"}

func TestLevenshteinMatch(t *testing.T) {
	m := NewLevenshtein(candidates, 0.5)

	tests := []struct {
		input         string
		expectedMatch string
	}{
		{"atp", "app"},
		{"adnin", "admin"},
		{"apii", "api"},
		{"auht", "auth"},
		{"blg", "blog"},
		{"shp", "shop"},
		{"doc", "docs"},
		{"app", "app"}, // exact
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, score := m.Match(tt.input)
			if got != tt.expectedMatch {
				t.Errorf("Match(%q) = %q (score %.2f), want %q", tt.input, got, score, tt.expectedMatch)
			}
		})
	}
}

func TestLevenshteinNoMatch(t *testing.T) {
	m := NewLevenshtein(candidates, 0.8)

	// "xyz" should not match anything above 0.8 threshold
	got, score := m.Match("xyz")
	if got != "" {
		t.Errorf("Match(%q) = %q (score %.2f), expected no match", "xyz", got, score)
	}
}

func TestLevenshteinExact(t *testing.T) {
	m := NewLevenshtein(candidates, 0.5)
	got, score := m.Match("admin")
	if got != "admin" || score != 1.0 {
		t.Errorf("exact match: got %q score %.2f, want admin 1.0", got, score)
	}
}

func TestLevenshteinEmptyInput(t *testing.T) {
	m := NewLevenshtein(candidates, 0.5)
	got, _ := m.Match("")
	// Empty input should not match anything meaningfully
	_ = got // result is implementation-defined; just ensure no panic
}

func TestLevenshteinEmptyCandidates(t *testing.T) {
	m := NewLevenshtein([]string{}, 0.5)
	got, score := m.Match("app")
	if got != "" || score != 0 {
		t.Errorf("empty candidates: got %q score %.2f, want empty", got, score)
	}
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"abc", "abc", 0},
		{"kitten", "sitting", 3},
		{"atp", "app", 1},
		{"adnin", "admin", 1}, // a-d-[n→m]-i-n: single substitution
	}

	for _, tt := range tests {
		t.Run(tt.a+"→"+tt.b, func(t *testing.T) {
			got := levenshtein(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func BenchmarkLevenshteinMatch(b *testing.B) {
	m := NewLevenshtein(candidates, 0.5)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Match("adnin")
	}
}
