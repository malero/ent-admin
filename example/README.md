# Ent Admin SQLite example

This directory is a small consumer application for the phase-one Ent Admin
foundation. It owns a single `Book` schema, uses SQLite for local persistence,
and mounts the generic admin handler at `/control`.

## Run it

From the repository root, regenerate the Ent package when the schema or
generator changes:

```sh
go run ./example/cmd/entgen
```

Then start the example from this directory:

```sh
cd example
go run ./cmd/admin
```

Open [http://127.0.0.1:8080/control/](http://127.0.0.1:8080/control/). The
first run creates `ent-admin.db` in the current directory and seeds three
deterministic books. Re-running the command is idempotent: existing books are
left unchanged.

The server logs each request through ordinary host-provided
`func(http.Handler) http.Handler` middleware. Use `-addr`, `-db`, or
`-base-path` to change the listen address, SQLite data source name, or mount
path. For example:

```sh
go run ./cmd/admin -addr 127.0.0.1:9090 -base-path /admin -db 'file:/tmp/ent-admin.db?_fk=1'
```

The admin UI is read-only. Its list rows include a VSN-powered detail preview
and retain a normal link fallback when JavaScript is unavailable.

## Test it

The example tests create an isolated temporary SQLite database, run the schema
migration, seed the records twice, and verify the mounted handler and host
middleware:

```sh
go test ./example/...
```

No Ent Admin-specific persistence tables are created; the database belongs to
the example application and contains only its `books` table.
