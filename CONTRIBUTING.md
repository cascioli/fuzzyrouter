# Contributing to FuzzyRouter

## Before You Start

- Read `CLAUDE.md` for architecture constraints, coding standards, and naming rules.
- Open an issue before starting non-trivial work — aligns scope early.

## Development Setup

```bash
git clone https://github.com/simlimone/fuzzyrouter.git
cd fuzzyrouter
go mod download
go test ./...
```

No additional tooling required. Single external dependency: `gopkg.in/yaml.v3`.

## Making Changes

### Rules

- Zero business logic in `cmd/`. All logic lives in `internal/`.
- Dependency direction is strict: `main → config, matcher, server`. No cycles.
- `server` must not import `config`. `matcher` must not import anything internal.
- No new external dependencies beyond `gopkg.in/yaml.v3`.
- No frameworks (`gin`, `echo`), no logger libraries (`zap`, `logrus`).

### Adding a New Matcher

1. Create `internal/matcher/<name>.go`
2. Implement `matcher.Matcher` interface (`Match(input string) (string, float64)`)
3. Export constructor: `New<Name>(candidates []string, threshold float64)`
4. Add `<name>_test.go` with the same test matrix as `levenshtein_test.go`
5. Add `BenchmarkLevenshteinMatch` equivalent
6. Wire in `main.go` behind a config field

### Adding a Config Field

1. Add field to `Config` struct with YAML tag
2. Handle ENV override in `overlayEnv`
3. Add validation in `validate`
4. Update `config.example.yaml`
5. Update the environment variables table in `README.md`

## Testing

```bash
go test ./...
go vet ./...
```

- Table-driven tests with a `name` field on every case.
- No mocks. No test-only interfaces. Test against real implementations.
- Benchmarks required for core algorithm changes.
- Discard logs in tests: `slog.New(slog.NewTextHandler(io.Discard, nil))`.

## Pull Request Checklist

- [ ] `go test ./...` passes
- [ ] `go vet ./...` clean
- [ ] New config fields reflected in `config.example.yaml` and `README.md`
- [ ] No new external dependencies
- [ ] GoDoc on all exported types and functions

## Commit Style

Conventional Commits. Subject ≤ 50 chars.

```
feat: add jaro-winkler matcher
fix: handle empty subdomain in extractSubdomain
docs: update ENV table in README
test: add benchmark for levenshtein with large candidate list
```
