package example

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	entadmin "github.com/malero/ent-admin"
)

func TestOpenAndSeedSQLite(t *testing.T) {
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "example.db") + "?_fk=1"
	client, err := Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err := Seed(ctx, client); err != nil {
		t.Fatal(err)
	}
	if err := Seed(ctx, client); err != nil {
		t.Fatal(err)
	}

	books, err := client.Book.Query().All(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(books) != 3 {
		t.Fatalf("seeded books = %d, want 3", len(books))
	}
	for _, book := range books {
		if book.Title == "" || book.Author == "" {
			t.Fatalf("seeded book has incomplete data: %#v", book)
		}
	}
}

func TestNewHandlerMountsWithHostMiddleware(t *testing.T) {
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "example.db") + "?_fk=1"
	client, err := Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := Seed(ctx, client); err != nil {
		t.Fatal(err)
	}

	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Example-Middleware", "present")
			next.ServeHTTP(w, r)
		})
	}
	admin, err := NewHandler(client, "/control", middleware)
	if err != nil {
		t.Fatal(err)
	}
	if got := MountPath(admin); got != "/control/" {
		t.Fatalf("MountPath = %q, want /control/", got)
	}

	mux := http.NewServeMux()
	mux.Handle(MountPath(admin), admin)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/control/books", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", res.Code, http.StatusOK, res.Body.String())
	}
	if got := res.Header().Get("X-Example-Middleware"); got != "present" {
		t.Fatalf("middleware header = %q, want present", got)
	}
	for _, want := range []string{"The Go Programming Language", "Learning Go", "The Go Workshop", "/control/assets/vsn.min.js"} {
		if !strings.Contains(res.Body.String(), want) {
			t.Errorf("response does not contain %q: %q", want, res.Body.String())
		}
	}
}

func TestNewHandlerRejectsNilClient(t *testing.T) {
	if _, err := NewHandler(nil, "/control"); err == nil {
		t.Fatal("NewHandler unexpectedly accepted a nil client")
	}
}

func TestMountPathForRoot(t *testing.T) {
	admin, err := entadmin.New(entadmin.Config{
		Registry: entadmin.Registry{Entities: []entadmin.Entity{{
			Name:        "Book",
			Label:       "Book",
			PluralLabel: "Books",
			Route:       "books",
			ID:          entadmin.Field{Name: "id", Label: "Id", Kind: entadmin.FieldKindInt},
			Reader:      &testReader{},
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := MountPath(admin); got != "/" {
		t.Fatalf("MountPath(root) = %q, want /", got)
	}
}

// testReader keeps the root-only mount test independent from the generated
// example database client.
type testReader struct{}

func (*testReader) List(context.Context, entadmin.ListOptions) (entadmin.ListResult, error) {
	return entadmin.ListResult{}, nil
}

func (*testReader) Get(context.Context, string) (entadmin.Record, error) {
	return entadmin.Record{}, entadmin.ErrNotFound
}
