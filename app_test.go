package entadmin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

type testReader struct {
	listResult ListResult
	listErr    error
	records    map[string]Record
	getErr     error
	listOpts   []ListOptions
}

func (r *testReader) List(_ context.Context, opts ListOptions) (ListResult, error) {
	r.listOpts = append(r.listOpts, opts)
	return r.listResult, r.listErr
}

func (r *testReader) Get(_ context.Context, id string) (Record, error) {
	if r.getErr != nil {
		return Record{}, r.getErr
	}
	record, ok := r.records[id]
	if !ok {
		return Record{}, ErrNotFound
	}
	return record, nil
}

func testRegistry(reader *testReader) Registry {
	return Registry{Entities: []Entity{{
		Name:        "Book",
		Label:       "Book",
		PluralLabel: "Books",
		Route:       "books",
		ID:          Field{Name: "id", Kind: FieldKindInt},
		Fields:      []Field{{Name: "title", Label: "Title", Kind: FieldKindString}},
		Reader:      reader,
	}}}
}

func TestNewNormalizesBasePath(t *testing.T) {
	for raw, want := range map[string]string{
		"":          "/",
		"/":         "/",
		"/admin":    "/admin",
		"/admin/":   "/admin",
		"/control/": "/control",
	} {
		app, err := New(Config{BasePath: raw, Registry: testRegistry(&testReader{})})
		if err != nil {
			t.Fatalf("New(%q): %v", raw, err)
		}
		if got := app.BasePath(); got != want {
			t.Errorf("New(%q).BasePath() = %q, want %q", raw, got, want)
		}
	}
}

func TestNewRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{name: "missing leading slash", cfg: Config{BasePath: "admin", Registry: testRegistry(&testReader{})}},
		{name: "query in base path", cfg: Config{BasePath: "/admin?tab=books", Registry: testRegistry(&testReader{})}},
		{name: "nil middleware", cfg: Config{Registry: testRegistry(&testReader{}), Middleware: []Middleware{nil}}},
		{name: "duplicate route", cfg: Config{Registry: Registry{Entities: []Entity{
			{Name: "Book", Label: "Book", PluralLabel: "Books", Route: "books", ID: Field{Name: "id", Kind: FieldKindInt}, Reader: &testReader{}},
			{Name: "Novel", Label: "Novel", PluralLabel: "Novels", Route: "books", ID: Field{Name: "id", Kind: FieldKindInt}, Reader: &testReader{}},
		}}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := New(tt.cfg); err == nil {
				t.Fatal("New unexpectedly succeeded")
			}
		})
	}
}

func TestAppDispatchesIndexListAndDetailUnderBasePath(t *testing.T) {
	reader := &testReader{
		listResult: ListResult{Records: []Record{{ID: "42", Values: map[string]any{"title": "Go Patterns"}}}},
		records:    map[string]Record{"42": {ID: "42", Values: map[string]any{"title": "Go Patterns"}}},
	}
	app, err := New(Config{BasePath: "/control/", Registry: testRegistry(reader)})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path       string
		statusCode int
		contains   string
	}{
		{path: "/control", statusCode: http.StatusOK, contains: `href="/control/books">Books`},
		{path: "/control/books", statusCode: http.StatusOK, contains: `href="/control/books/42">42`},
		{path: "/control/books/42", statusCode: http.StatusOK, contains: `href="/control/books">Books`},
		{path: "/outside/books", statusCode: http.StatusNotFound, contains: "page not found"},
		{path: "/control/unknown", statusCode: http.StatusNotFound, contains: "page not found"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			res := httptest.NewRecorder()
			app.ServeHTTP(res, req)
			if res.Code != tt.statusCode {
				t.Fatalf("status = %d, want %d; body=%q", res.Code, tt.statusCode, res.Body.String())
			}
			if !strings.Contains(res.Body.String(), tt.contains) {
				t.Errorf("body %q does not contain %q", res.Body.String(), tt.contains)
			}
		})
	}
	if got := reader.listOpts; !reflect.DeepEqual(got, []ListOptions{{Limit: defaultPageSize}}) {
		t.Fatalf("list options = %#v", got)
	}

	mux := http.NewServeMux()
	mux.Handle("/control/", app)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/control/books", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("ServeMux-mounted app status = %d, want %d; body=%q", res.Code, http.StatusOK, res.Body.String())
	}
	if got := res.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("ServeMux-mounted app content type = %q, want HTML", got)
	}
}

func TestAppEscapesRenderedRecordValues(t *testing.T) {
	reader := &testReader{
		listResult: ListResult{Records: []Record{{ID: "1", Values: map[string]any{"title": "<script>alert(1)</script>"}}}},
	}
	app, err := New(Config{BasePath: "/control", Registry: testRegistry(reader)})
	if err != nil {
		t.Fatal(err)
	}

	res := httptest.NewRecorder()
	app.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/control/books", nil))
	body := res.Body.String()
	if strings.Contains(body, "<script>alert(1)</script>") {
		t.Fatalf("response rendered an unescaped value: %q", body)
	}
	if !strings.Contains(body, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Fatalf("response does not contain the escaped value: %q", body)
	}
}

func TestAppServesVSNPreviewAndScript(t *testing.T) {
	reader := &testReader{
		listResult: ListResult{Records: []Record{{ID: "42", Values: map[string]any{"title": "Go Patterns"}}}},
		records:    map[string]Record{"42": {ID: "42", Values: map[string]any{"title": "Go Patterns"}}},
	}
	app, err := New(Config{BasePath: "/control", Registry: testRegistry(reader)})
	if err != nil {
		t.Fatal(err)
	}

	listResponse := httptest.NewRecorder()
	app.ServeHTTP(listResponse, httptest.NewRequest(http.MethodGet, "/control/books", nil))
	listBody := listResponse.Body.String()
	for _, want := range []string{
		`src="/control/assets/vsn.min.js"`,
		`auto-mount`,
		`vsn-get="/control/books/42"`,
		`vsn-target="#record-preview"`,
		`vsn-swap="inner"`,
	} {
		if !strings.Contains(listBody, want) {
			t.Errorf("list response does not contain %q: %q", want, listBody)
		}
	}

	partialRequest := httptest.NewRequest(http.MethodGet, "/control/books/42", nil)
	partialRequest.Header.Set("HX-Request", "true")
	partialRequest.Header.Set("HX-Target", "record-preview")
	partialResponse := httptest.NewRecorder()
	app.ServeHTTP(partialResponse, partialRequest)
	partialBody := partialResponse.Body.String()
	if !strings.HasPrefix(partialBody, "<article>") {
		t.Fatalf("partial response = %q, want detail fragment", partialBody)
	}
	if strings.Contains(partialBody, "<!doctype html>") || !strings.Contains(partialBody, "Go Patterns") {
		t.Fatalf("partial response is not the expected detail fragment: %q", partialBody)
	}
	if got := partialResponse.Header().Get("Vary"); got != "HX-Request" {
		t.Fatalf("partial response Vary = %q, want HX-Request", got)
	}

	fullResponse := httptest.NewRecorder()
	app.ServeHTTP(fullResponse, httptest.NewRequest(http.MethodGet, "/control/books/42", nil))
	if !strings.HasPrefix(fullResponse.Body.String(), "<!doctype html>") {
		t.Fatalf("ordinary detail response = %q, want complete HTML document", fullResponse.Body.String())
	}
	if got := fullResponse.Header().Get("Vary"); got != "HX-Request" {
		t.Fatalf("full response Vary = %q, want HX-Request", got)
	}

	assetResponse := httptest.NewRecorder()
	app.ServeHTTP(assetResponse, httptest.NewRequest(http.MethodGet, "/control/assets/vsn.min.js", nil))
	if assetResponse.Code != http.StatusOK {
		t.Fatalf("VSN asset status = %d, want %d", assetResponse.Code, http.StatusOK)
	}
	if got := assetResponse.Header().Get("Content-Type"); got != "text/javascript; charset=utf-8" {
		t.Fatalf("VSN asset content type = %q", got)
	}
	if len(assetResponse.Body.Bytes()) < 100_000 {
		t.Fatalf("VSN asset is unexpectedly small: %d bytes", assetResponse.Body.Len())
	}

	outOfScope := httptest.NewRecorder()
	app.ServeHTTP(outOfScope, httptest.NewRequest(http.MethodGet, "/assets/vsn.min.js", nil))
	if outOfScope.Code != http.StatusNotFound {
		t.Fatalf("outside-base VSN asset status = %d, want %d", outOfScope.Code, http.StatusNotFound)
	}
}

func TestAppAppliesMiddlewareInDeclarationOrder(t *testing.T) {
	var events []string
	reader := &testReader{}
	middleware := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				events = append(events, name+" before")
				next.ServeHTTP(w, r)
				events = append(events, name+" after")
			})
		}
	}
	app, err := New(Config{
		Registry:   testRegistry(reader),
		Middleware: []Middleware{middleware("first"), middleware("second")},
	})
	if err != nil {
		t.Fatal(err)
	}

	res := httptest.NewRecorder()
	app.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/books", nil))
	want := []string{"first before", "second before", "second after", "first after"}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("middleware events = %#v, want %#v", events, want)
	}

	events = nil
	res = httptest.NewRecorder()
	app.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/missing", nil))
	if res.Code != http.StatusNotFound {
		t.Fatalf("missing route status = %d, want %d", res.Code, http.StatusNotFound)
	}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("middleware events for missing route = %#v, want %#v", events, want)
	}
}

func TestAppRejectsAmbiguousPaths(t *testing.T) {
	app, err := New(Config{BasePath: "/control", Registry: testRegistry(&testReader{})})
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{"/control//books", "/control/books//42", "/control/books/42/extra", "/control/books/%2F"} {
		if match, ok := app.match(path); ok {
			t.Errorf("match(%q) = %#v, true; want no match", path, match)
		}
	}
}

func TestAppMapsReaderErrorsAndMethods(t *testing.T) {
	tests := []struct {
		name       string
		reader     *testReader
		method     string
		path       string
		statusCode int
	}{
		{name: "not found", reader: &testReader{getErr: ErrNotFound}, method: http.MethodGet, path: "/books/7", statusCode: http.StatusNotFound},
		{name: "list failure", reader: &testReader{listErr: errors.New("database unavailable")}, method: http.MethodGet, path: "/books", statusCode: http.StatusInternalServerError},
		{name: "method not allowed", reader: &testReader{}, method: http.MethodPost, path: "/books", statusCode: http.StatusMethodNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, err := New(Config{Registry: testRegistry(tt.reader)})
			if err != nil {
				t.Fatal(err)
			}
			res := httptest.NewRecorder()
			app.ServeHTTP(res, httptest.NewRequest(tt.method, tt.path, nil))
			if res.Code != tt.statusCode {
				t.Fatalf("status = %d, want %d; body=%q", res.Code, tt.statusCode, res.Body.String())
			}
		})
	}
}
