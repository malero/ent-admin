# Ent Admin phase-one end-of-pass report

Status: implementation and verification report for SDD Step 8.

## Outcome

The phase-one foundation is implemented as a small, router-independent Go
package. Ent remains the source of truth; supported Ent generation APIs produce
deterministic metadata and concrete read adapters; the runtime exposes a
standard `http.Handler`; Templ renders the pages; VSN adds one progressive
detail-preview interaction; and the example proves a normal SQLite integration.

The implementation stops at the agreed read-only boundary. It does not imply
that CRUD, authentication, RBAC, search, audit logging, or other phase-two
features are complete.

## Delivered package surface

- `contract.go`: field, entity, registry, record, pagination, reader, and
  not-found contracts.
- `config.go`: `Config` and standard `net/http` middleware contract.
- `entc/`: Ent extension that inspects `*entc/gen.Graph` and emits generated
  metadata plus concrete read adapters.
- `app.go`: base-path-aware runtime implementing `http.Handler`, with index,
  list, detail, error, asset, method, and middleware behavior.
- `internal/ui/`: generated Templ components, safe display helpers, embedded
  VSN serving, complete-page rendering, and detail fragments.
- `example/`: generated Book Ent package, SQLite migration/seed helpers,
  runnable server, host middleware, and integration tests.
- `docs/`: architecture decisions, actual VSN feedback, and this report.

## Verification performed

The following checks passed from the repository root:

- `go mod tidy`
- `go run -mod=readonly ./example/cmd/entgen`
- `/Users/mattroberts/go/bin/templ generate` with `updates=0`
- `GOCACHE=/private/tmp/ent-admin-gocache GOMODCACHE=/Users/mattroberts/go/pkg/mod go test -count=1 ./...`
- `GOCACHE=/private/tmp/ent-admin-gocache GOMODCACHE=/Users/mattroberts/go/pkg/mod go vet ./...`
- `gofmt -d` over all Go sources
- `git diff --check`
- Upstream VSN request-contract tests: 6 tests passed

The live example was also started with a temporary SQLite database. The
mounted `/control/books` endpoint returned the seeded three-book list and
base-path VSN markup. A request to `/control/books/1` with
`HX-Request: true` returned the detail fragment with `Vary: HX-Request`.

The generator test exercises repeated output for byte-identical results,
checks generated Go parsing, verifies metadata/read paths, and rejects an
unsupported ID family. Runtime tests cover base paths, standard mux mounting,
middleware order, route/method/error behavior, escaping, VSN markup and asset
routing, full-page fallback, and partial responses. Example tests cover SQLite
migration, idempotent seeding, middleware mounting, and the executable's
consumer-facing integration boundary.

## Decisions and consequences

1. Generated descriptors and readers are the boundary between Ent and the
   generic runtime. This preserves compile-time type checking without coupling
   the root package to a consuming application's generated package.
2. The runtime is a standard `http.Handler` and constructs every internal URL
   from normalized `BasePath`. This keeps router choice and authentication in
   the host application.
3. Templ owns server-rendered HTML; generated code contains no page markup.
   Generated output remains deterministic and visual changes do not require
   schema regeneration.
4. VSN is isolated to list-page attributes, an embedded browser asset, and a
   small detail-fragment branch. Ordinary links and complete pages remain the
   fallback.
5. The example uses application-owned SQLite data only. No admin-specific
   persistence is needed for the phase-one runtime.

## Limitations

- The UI is read-only and intentionally has no CRUD, deletion, bulk action,
  filtering, sorting, search, or relationship editing.
- Host middleware is the only phase-one access-control mechanism; the package
  does not implement authentication or RBAC.
- The generator supports only field families with predictable generic display
  behavior and does not yet expose custom field annotations or formatters.
- The example uses the CGO-backed `github.com/mattn/go-sqlite3` driver.
- The VSN interaction has Go handler and live HTTP coverage but no committed
  browser automation harness.
- The example server has no graceful-shutdown command path because it is a
  deliberately small local demonstration.

## VSN feedback summary

The implementation uses VSN 1.0.15 from source revision `32c7e21`, with the
pinned `dist/index.min.js` artifact embedded under `internal/ui/assets/`. The
current API maps cleanly to a server-rendered fragment flow through
`vsn-get`, `vsn-target`, and `vsn-swap`, and its `HX-Request` header makes the
full-page/fragment distinction explicit.

The main friction was expressing modifier-bearing custom attributes such as
`vsn-get!trusted` in Templ. Plain `vsn-get` was sufficient for this fragment
and retained VSN's default sanitizer path. Detailed observations and
follow-up ideas are in [vsn-feedback.md](vsn-feedback.md).

## Phase-two recommendations

- Add a narrow authorization decision interface separate from host middleware.
- Add field display annotations, ordering, custom formatters, filtering, and
  sorting without changing generated HTML boundaries.
- Add opt-in CRUD command adapters and validation separate from the read-only
  `Reader`.
- Add relationship summaries/editors and richer Ent field support incrementally.
- Add optional persistence for settings/audit data only when a concrete use
  case requires it.
- Add a browser-level VSN/e2e harness and define the supported pinned-asset
  update workflow.

## Manual review items

- Confirm the public package names and generated `AdminRegistry` contract are
  the desired long-term API before publishing a first release.
- Review whether the CGO SQLite example is the right default or whether a
  pure-Go example driver should be offered later.
- Review the VSN asset vendoring and update/licensing process for open-source
  distribution.
- Review the HTML and accessibility details of the initial list/detail pages
  before treating them as a stable theme.
- Confirm that the explicit phase-one exclusions remain appropriate for the
  next SDD revision.
