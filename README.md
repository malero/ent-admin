# Ent Admin

Ent Admin is a small, read-only administrative UI for Go applications that
use [Ent](https://entgo.io/). It generates UI-neutral entity metadata and
strongly typed read adapters from the Ent graph, then renders the result with
standard `net/http`, Templ, and progressive VSN enhancements.

Phase one is intentionally a foundation rather than a complete admin suite:
it covers generated metadata, a router-independent handler, an entity index,
read-only list/detail pages, configurable mounting, host middleware, and a
reproducible SQLite example. CRUD, authentication, RBAC, search, advanced
filters, audit logs, dashboards, and relationship editors are future work.

## Requirements

- Go 1.25 or newer. The module currently targets Go 1.25.0 because the Ent
  generation toolchain selected for development requires it.
- A C compiler when running the SQLite example, which uses
  `github.com/mattn/go-sqlite3`.

## Quick start

The checked-in example uses a `Book` Ent schema and SQLite. From the repository
root, regenerate the example Ent package when the schema or generator changes:

```sh
go run ./example/cmd/entgen
```

Start the example from its directory:

```sh
cd example
go run ./cmd/admin
```

Open [http://127.0.0.1:8080/control/](http://127.0.0.1:8080/control/). The
first run creates `ent-admin.db`, migrates the application schema, and seeds
three deterministic books. The example is read-only and does not create any
Ent Admin-specific tables. See [example/README.md](example/README.md) for
flags and isolated test details.

## Mounting in an application

The generated package exposes `AdminRegistry`, which binds the generated
readers to the consuming application's concrete Ent client:

The example below omits the surrounding imports and host-defined
`requireStaff` implementation.

```go
client, err := ent.Open(dialect.SQLite, "file:app.db?_fk=1")
if err != nil {
	return err
}
defer client.Close()

admin, err := entadmin.New(entadmin.Config{
	Registry: ent.AdminRegistry(client),
	BasePath: "/control",
	Middleware: []entadmin.Middleware{
		requireStaff,
	},
})
if err != nil {
	return err
}

mux := http.NewServeMux()
mux.Handle("/control/", admin)
```

`requireStaff` is owned by the host application and has the standard
`func(http.Handler) http.Handler` shape. Ent Admin does not create users,
sessions, authentication, or authorization tables. The handler owns its
internal URLs, so links and assets remain under `BasePath`; a consuming router
does not need a prefix-stripping adapter.

## Generation and tests

The Ent extension uses Ent's supported graph and template APIs. It does not
parse generated Go source and does not generate HTML. The generated output is
marked as generated and lives beside the consuming application's normal Ent
package.

Run the complete phase-one checks from the repository root:

```sh
go run -mod=readonly ./example/cmd/entgen
/Users/mattroberts/go/bin/templ generate
go test ./...
go vet ./...
```

The `templ` binary path above reflects the development environment; the
checked-in generated `internal/ui/page_templ.go` means users do not need Templ
to build the example from a clean checkout. The normal `go test ./...` command
also exercises the isolated SQLite example tests.

## VSN and progressive rendering

The list page includes a VSN-powered detail preview. Each preview control also
has a normal detail `href`, so the same feature works as a complete server-
rendered page when JavaScript is unavailable. The pinned VSN browser artifact
is served from the configured base path; no CDN is required.

The observed integration details and friction are recorded in
[docs/vsn-feedback.md](docs/vsn-feedback.md). The broader package boundaries
and trade-offs are in [docs/architecture.md](docs/architecture.md).

## Scope boundary

Phase one deliberately stops at a read-only foundation. It does not promise
CRUD, delete or bulk actions, filtering/search, authentication/RBAC, audit
logging, persistent admin settings, dashboards, uploads, relationship editors,
or support for every Ent field and edge shape. The end-of-pass decisions,
verification results, limitations, and phase-two recommendations are in
[docs/phase-one-report.md](docs/phase-one-report.md).
