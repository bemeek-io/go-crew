# Contributing to go-crew

Thanks for your interest in improving go-crew!

## What We Need

- Live validation of documented-but-unverified operations (see the validation
  status note in [docs/api.md](docs/api.md)) and corrections where the real
  schema differs.
- Typed wrappers for more of the API surface (bills, rules, family, wallets).
- Bug reports with the failing GraphQL query and (redacted) response.

## Getting Started

1. Fork the repository and clone your fork.
2. Create a feature branch: `git checkout -b my-change`.
3. Make your changes, keeping the public API surface small.
4. Add or update tests — every exported method needs coverage against the
   fake server in `client_test.go`.
5. Run the checks below.
6. If you have a Crew account, run the live smoke tests.
7. Open a pull request describing what changed and how you verified it.

## Testing

```bash
go build ./...
go vet ./...
go test -race ./...
golangci-lint run
```

Live tests hit the real Crew API and are gated behind a build tag:

```bash
CREW_TOKEN=<bearer token> go test -tags live -run TestLive -v
```

## Code Style

- `gofmt` formatting; `go vet` and `golangci-lint` must pass.
- Flat single package `crew`; unexported internals.
- Functional options (`With*`), each documenting its default inline.
- Use `context.Context` on anything that hits the network.
- Return errors rather than panicking; prefix messages with `crew:` and wrap
  with `%w`. Sentinel errors live in `errors.go`.
- Add JSON struct tags to new model fields; query only fields the SDK models.
- Never introduce automatic retries around money-moving mutations —
  `initiateTransfer` has no idempotency mechanism.

## API Discovery Tips

Crew's GraphQL reference lives at `https://docs.trycrew.com/`. Field names
there don't always match the live schema — when in doubt, probe with a small
query via `Client.Execute` and confirm before adding typed wrappers.
