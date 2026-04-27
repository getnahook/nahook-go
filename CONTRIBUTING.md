# Contributing to nahook-go

Thanks for considering a contribution! A few important things to know first.

## Source of truth

This repository is a **subtree-split mirror** of the Go SDK from our private monorepo `getnahook/nahook`. PRs filed directly here **cannot be merged** — the next subtree-push from the monorepo will force-overwrite this branch.

## What we welcome

- **Bug reports** — open a GitHub issue with: reproduction steps, SDK version, Go version (`go version`), OS, and `go env GOOS GOARCH`.
- **Feature requests** — open an issue describing the use case and the API surface you'd want.
- **Small code suggestions** — paste a snippet in an issue and describe intent; we'll port it into the monorepo and credit you in the resulting commit.
- **Substantial patches** — email `support@nahook.com` first; we'll either discuss read access to the monorepo or hand-port your change with credit.

## Local development

```bash
git clone https://github.com/getnahook/nahook-go
cd nahook-go
go build ./...
go test ./...    # ~100 tests across 6 packages
go vet ./...
```

`go.mod` declares `go 1.21`. SDK supports Go 1.21+.

### Code style

- `gofmt -l .` must be empty (CI enforces)
- `go vet ./...` must be clean (CI enforces)
- Idiomatic Go — match surrounding patterns

## License

By contributing, you agree your changes are released under the [MIT License](LICENSE).
