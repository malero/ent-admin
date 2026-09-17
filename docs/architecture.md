# Ent Admin architecture

Status: phase-one implementation contract, prepared during SDD Step 2.

## Purpose and scope

Ent Admin is a reusable Go package that turns selected Ent schema information
into a small, read-only, server-rendered administrative interface. Phase one
proves the boundaries needed for a useful foundation:

- Ent remains the source of truth.
- Ent-supported generation APIs inspect the graph at compile time.
- Generated code contains descriptors and narrow read adapters, not HTML.
- A generic runtime implements `http.Handler` and consumes the generated
  registry.
- Templ renders ordinary HTML; VSN adds one progressive enhancement path.
- The host application owns mounting, authentication, and middleware.

CRUD, deletion, search, advanced filters, authentication, RBAC, audit logs,
admin-specific persistence, dashboards, relationship editors, and a general
plugin system remain outside phase one.

## Target package layout

The repository is organized by responsibility rather than by generated entity:

```text
github.com/malero/ent-admin/
├── contract.go             # public descriptors, records, and Reader contract
├── config.go               # public Config and net/http Middleware contract
├── app.go                  # generic HTTP runtime (Step 4)
├── entc/                   # public Ent extension (Step 3)
├── internal/ui/            # generic Templ components (Step 5)
├── example/                # reproducible Ent + SQLite example (Step 7)
├── docs/architecture.md
└── docs/vsn-feedback.md
```

Generated Ent output stays in the consuming application's normal Ent target
package. The extension writes one clearly generated Go file there; it does not
create a parallel package of generated pages.

## Compile-time generation boundary

The public `entc` extension is passed to Ent's supported generation workflow:

```go
entc.Generate(
    "./schema",
    &gen.Config{Target: "./ent"},
    entadminentc.Extension(),
)
```

The extension consumes `*entc/gen.Graph`, `*gen.Type`, and `*gen.Field` values
provided by Ent. It never parses generated Go source. The graph is converted
to deterministic output in schema order, with stable formatting and a
generated-file header.

The generated target package exposes a function shaped like:

```go
func AdminRegistry(client *Client) entadmin.Registry
```

That function binds concrete, per-entity readers to the caller's Ent client.
The root package therefore has no dependency on a user's generated `ent`
package, while the generated package can use concrete query builders and
getters without reflection.

## Generated/runtime contract

The root package exposes four deliberately small concepts:

```go
type Registry struct { Entities []Entity }

type Entity struct {
    Name, Label, PluralLabel, Route string
    ID      Field
    Fields  []Field
    Reader  Reader
}

type Reader interface {
    List(context.Context, ListOptions) (ListResult, error)
    Get(context.Context, string) (Record, error)
}
```

`Entity.Fields` is ordered metadata. `Record.Values` is keyed by field name,
and `Record.ID` is a string so URL construction does not leak an Ent ID type
into the runtime. The reader is read-only and is bound to a concrete client at
registry construction time. This makes the runtime generic without making its
hot path depend on reflection or an untyped client parameter.

The phase-one generator supports only field families that the generic UI can
format predictably. Unsupported or ambiguous schema features fail generation
with an actionable error rather than silently producing misleading pages.
Generated code contains no Templ components, route handlers, or page-specific
markup.

## Runtime and HTTP integration

`entadmin.New(Config)` will return an `*App` that implements
`http.Handler`:

```go
admin, err := entadmin.New(entadmin.Config{
    Registry: ent.AdminRegistry(entClient),
    BasePath: "/control",
    Middleware: []entadmin.Middleware{requireStaff},
})
if err != nil {
    return err
}

mux.Handle("/control/", admin)
```

The runtime uses only `net/http`, URL/path utilities, and the generated
registry. It does not import Chi, Gorilla, Echo, Gin, or another router. A
consumer mounts the handler at the same path supplied as `BasePath`; router
introspection and a required `StripPrefix` are intentionally unnecessary.

### Base-path semantics

`BasePath` is normalized once during construction:

- `""` and `"/"` mean the root path `/`.
- Other values must become a single leading-slash path without a trailing
  slash, so `/admin/` becomes `/admin`.
- A value containing a query, fragment, or malformed path is rejected.
- The handler owns all internal URL construction. Entity routes, record links,
  form actions, assets, and VSN requests are joined under the normalized base
  path; no string literal `/admin` is used by the UI.
- Requests outside the configured base path receive `404`.

This contract keeps the handler predictable when mounted in a standard mux or
behind a different compatible router.

### Middleware and host-owned security

`Middleware` is an alias of `func(http.Handler) http.Handler`. The first item
is the outermost wrapper, making it possible for a host application to place
authentication, request IDs, CSRF policy, logging, or rate limits around the
entire admin surface. Ent Admin does not create users, passwords, sessions,
OAuth flows, or RBAC tables.

Authorization is intentionally a future, narrow decision point rather than a
phase-one subsystem. A later contract may receive a request, entity, and
action and return an allow/deny result, but phase one relies on host middleware
for access control.

## UI layering

The UI is generic and lives below the runtime boundary:

1. The runtime resolves a route and invokes a generated `Reader`.
2. The runtime maps the result to a small view model.
3. Internal Templ components render the layout, index, list, and optional
   detail page.
4. The same handlers return complete HTML for ordinary browser requests.
5. One list refresh interaction may return an HTML fragment when VSN sends its
   verified legacy XHR marker; the browser can still follow the ordinary link
   and receive a complete page when JavaScript is unavailable.

Generated packages do not know about templates. This preserves one renderer,
keeps generated output deterministic, and leaves future themes/components
inside the runtime/UI layer.

## VSN decision and compatibility constraint

The development environment contains VSN `0.1.124` under the local
`node_modules` tree. Its inspectable API registers `vsn-xhr` and `vsn-on`
attributes and sends `X-Requested-With: XMLHttpRequest` for enhanced requests.
The local sources and sibling bundled asset did not expose the newer
`Engine`, `autoMount`, `vsn-get`, `vsn-target`, or `vsn-swap` names described in
some adjacent documentation. Phase one therefore targets the verified
legacy API, serves/pins the exact asset used by the example, and records
development feedback in `docs/vsn-feedback.md` during Step 6.

The enhancement is progressive: the server has a full-page HTML fallback,
and the VSN request is an optimization over the same route rather than the
only way to navigate.

## Optional future persistence

No admin-specific tables or settings are required by the phase-one runtime.
The generated registry is derived from the Ent graph and the consuming
application's existing client. Future preferences, saved views, audit data,
or user-to-admin configuration may use an opt-in persistence package with its
own schema and migration lifecycle. That package must remain optional so a
basic mount never requires an additional database dependency.

## Alternatives considered

### Generated HTML per entity

Rejected. It duplicates templates, makes visual changes require regeneration,
and couples the Ent graph to HTTP/UI details. Generated descriptors plus
readers keep the generated surface small.

### Runtime reflection over Ent entities

Rejected for phase one. It would require a stable runtime shape that Ent does
not promise across generated packages and would make errors appear at request
time. Narrow generated adapters provide compile-time type checking while the
runtime remains generic.

### Passing `any`/`interface{}` client values into readers

Rejected. Binding the concrete client when `AdminRegistry(client)` is called
keeps the untyped boundary out of every request and lets generated readers use
ordinary Ent query builders.

### Router-specific adapters

Rejected. A standard `http.Handler` composes with the host's chosen router and
does not force a dependency for a small administrative surface.

### JavaScript-first navigation

Rejected. Server-rendered HTML is the source of truth, and VSN remains a
progressive enhancement with a direct-link fallback.

## Phase-two seams

The following seams are intentionally left open without implementing the
features behind them:

- authorization decisions beyond host middleware;
- field display/ordering annotations and custom formatters;
- filtering and sorting in `ListOptions`;
- relationship summaries and editors;
- CRUD command adapters separate from the read-only `Reader`;
- optional persistence for settings/audit data;
- themes and extension points for UI components.

Any addition should preserve the phase-one rule that schema generation,
generic runtime behavior, and UI rendering remain separate responsibilities.
