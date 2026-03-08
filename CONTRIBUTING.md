# Contributing to VectorPad

## Getting started

```bash
git clone https://github.com/ppiankov/vectorpad.git
cd vectorpad
make build
make test
```

## Development

- Go 1.24+
- `make build` - build binary
- `make test` - run tests with race detection
- `make coverage` - run tests with coverage report
- `make lint` - run golangci-lint
- `make fmt` - format code

## Code style

- Standard Go formatting (gofmt/goimports)
- Internal packages use short single-word names
- Comments explain "why" not "what"
- No magic numbers - name and document constants
- All errors must be checked

## Testing

- Tests are mandatory for new code
- Run `make test` before submitting (includes `-race`)
- Deterministic tests only - no flaky or probabilistic tests
- Test files live alongside source files

## Commits

- Conventional commits: `feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `chore:`
- One line, max 72 chars, imperative mood
- Say what changed, not every detail of how

## Pull requests

- One feature or fix per PR
- Include tests for new behavior
- Run `make lint` and `make test` before opening
- Keep PRs focused - avoid unrelated changes

## Architecture

VectorPad uses a minimal `cmd/vectorpad/main.go` that delegates to `internal/` packages. The TUI is built with Bubbletea. See the README for the full package map.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
