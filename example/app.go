// Package example contains the small consumer-facing application used by the
// Ent Admin phase-one example.
package example

import (
	"context"
	"fmt"
	"net/http"

	"entgo.io/ent/dialect"
	"github.com/malero/ent-admin"
	"github.com/malero/ent-admin/example/ent"
	_ "github.com/mattn/go-sqlite3"
)

// Open opens a SQLite database and creates the example schema if necessary.
// The database contains only application-owned Book data; Ent Admin does not
// add any tables.
func Open(ctx context.Context, dataSourceName string) (*ent.Client, error) {
	client, err := ent.Open(dialect.SQLite, dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("open example database: %w", err)
	}
	if err := client.Schema.Create(ctx); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("create example schema: %w", err)
	}
	return client, nil
}

// Seed inserts the deterministic example records when the database is empty.
// It is intentionally idempotent so restarting the example does not duplicate
// records or modify existing application data.
func Seed(ctx context.Context, client *ent.Client) error {
	if client == nil {
		return fmt.Errorf("seed example database: nil client")
	}
	count, err := client.Book.Query().Count(ctx)
	if err != nil {
		return fmt.Errorf("count example books: %w", err)
	}
	if count > 0 {
		return nil
	}

	_, err = client.Book.CreateBulk(
		client.Book.Create().
			SetTitle("The Go Programming Language").
			SetAuthor("Alan A. A. Donovan and Brian W. Kernighan").
			SetPublished(true),
		client.Book.Create().
			SetTitle("Learning Go").
			SetAuthor("Jon Bodner").
			SetPublished(true),
		client.Book.Create().
			SetTitle("The Go Workshop").
			SetAuthor("Ethan Jones").
			SetPublished(false),
	).Save(ctx)
	if err != nil {
		return fmt.Errorf("seed example books: %w", err)
	}
	return nil
}

// NewHandler creates the generic admin handler for the example application.
// Middleware is supplied by the host application to demonstrate the standard
// net/http integration boundary.
func NewHandler(client *ent.Client, basePath string, middleware ...entadmin.Middleware) (*entadmin.App, error) {
	if client == nil {
		return nil, fmt.Errorf("create example admin: nil client")
	}
	return entadmin.New(entadmin.Config{
		Registry:   ent.AdminRegistry(client),
		BasePath:   basePath,
		Middleware: middleware,
	})
}

// MountPath returns the path pattern used to mount an admin handler on a
// standard net/http ServeMux.
func MountPath(admin *entadmin.App) string {
	if admin == nil || admin.BasePath() == "/" {
		return "/"
	}
	return admin.BasePath() + "/"
}

// Compile-time checks keep the example's public integration shape explicit.
var _ http.Handler = (*entadmin.App)(nil)
