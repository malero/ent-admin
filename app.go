package entadmin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/malero/ent-admin/internal/ui"
	"github.com/malero/ent-admin/internal/view"
)

const defaultPageSize = 50

const vsnAssetPath = "/assets/vsn.min.js"

// App is the router-independent Ent Admin HTTP application.
type App struct {
	basePath string
	registry Registry
	handler  http.Handler
}

var _ http.Handler = (*App)(nil)

// New constructs an admin application and applies the configured middleware.
func New(cfg Config) (*App, error) {
	basePath, err := normalizeBasePath(cfg.BasePath)
	if err != nil {
		return nil, err
	}
	if err := validateRegistry(cfg.Registry); err != nil {
		return nil, err
	}
	for i, middleware := range cfg.Middleware {
		if middleware == nil {
			return nil, fmt.Errorf("ent-admin: middleware %d is nil", i)
		}
	}

	app := &App{
		basePath: basePath,
		registry: cfg.Registry,
	}
	var handler http.Handler = http.HandlerFunc(app.serveHTTP)
	for i := len(cfg.Middleware) - 1; i >= 0; i-- {
		handler = cfg.Middleware[i](handler)
		if handler == nil {
			return nil, fmt.Errorf("ent-admin: middleware %d returned a nil handler", i)
		}
	}
	app.handler = handler
	return app, nil
}

// BasePath returns the normalized mount path used by the application.
func (a *App) BasePath() string {
	return a.basePath
}

// ServeHTTP implements http.Handler.
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.handler.ServeHTTP(w, r)
}

type routeKind uint8

const (
	routeIndex routeKind = iota
	routeList
	routeDetail
	routeAsset
)

type routeMatch struct {
	kind   routeKind
	entity Entity
	id     string
}

func (a *App) serveHTTP(w http.ResponseWriter, r *http.Request) {
	match, ok := a.match(r.URL.Path)
	if !ok {
		a.renderError(w, r, http.StatusNotFound, "page not found")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		a.renderError(w, r, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	switch match.kind {
	case routeIndex:
		a.serveIndex(w, r)
	case routeList:
		a.serveList(w, r, match.entity)
	case routeDetail:
		a.serveDetail(w, r, match.entity, match.id)
	case routeAsset:
		a.serveVSNAsset(w)
	default:
		a.renderError(w, r, http.StatusNotFound, "page not found")
	}
}

func (a *App) serveIndex(w http.ResponseWriter, r *http.Request) {
	entities := make([]view.Entity, 0, len(a.registry.Entities))
	for _, entity := range a.registry.Entities {
		entities = append(entities, toViewEntity(entity))
	}
	a.renderPage(w, r.Context(), view.Page{
		Kind:        view.PageIndex,
		StatusCode:  http.StatusOK,
		BasePath:    a.basePath,
		CurrentPath: r.URL.Path,
		Title:       "Ent Admin",
		Entities:    entities,
	})
}

func (a *App) serveList(w http.ResponseWriter, r *http.Request, entity Entity) {
	result, err := entity.Reader.List(r.Context(), ListOptions{Limit: defaultPageSize})
	if err != nil {
		a.renderError(w, r, http.StatusInternalServerError, "unable to load records")
		return
	}

	entityView := toViewEntity(entity)
	a.renderPage(w, r.Context(), view.Page{
		Kind:        view.PageList,
		StatusCode:  http.StatusOK,
		BasePath:    a.basePath,
		CurrentPath: r.URL.Path,
		Title:       entity.PluralLabel,
		Entity:      &entityView,
		Records:     toViewRecords(result.Records),
	})
}

func (a *App) serveDetail(w http.ResponseWriter, r *http.Request, entity Entity, id string) {
	w.Header().Add("Vary", "HX-Request")
	record, err := entity.Reader.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			a.renderError(w, r, http.StatusNotFound, "record not found")
			return
		}
		a.renderError(w, r, http.StatusInternalServerError, "unable to load record")
		return
	}

	entityView := toViewEntity(entity)
	recordView := toViewRecord(record)
	page := view.Page{
		Kind:        view.PageDetail,
		StatusCode:  http.StatusOK,
		BasePath:    a.basePath,
		CurrentPath: r.URL.Path,
		Title:       entity.Label + " " + record.ID,
		Entity:      &entityView,
		Record:      &recordView,
	}
	if isVSNRequest(r) {
		a.renderDetailFragment(w, r.Context(), page)
		return
	}
	a.renderPage(w, r.Context(), page)
}

func (a *App) serveVSNAsset(w http.ResponseWriter) {
	if err := ui.ServeVSNScript(w); err != nil {
		http.Error(w, "ent-admin: serve VSN asset: "+err.Error(), http.StatusInternalServerError)
	}
}

func isVSNRequest(r *http.Request) bool {
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("HX-Request")), "true")
}

func (a *App) renderError(w http.ResponseWriter, r *http.Request, status int, message string) {
	a.renderPage(w, r.Context(), view.Page{
		Kind:        view.PageError,
		StatusCode:  status,
		BasePath:    a.basePath,
		CurrentPath: r.URL.Path,
		Title:       http.StatusText(status),
		Error:       message,
	})
}

func (a *App) renderPage(w http.ResponseWriter, ctx context.Context, page view.Page) {
	if err := ui.Render(ctx, w, page); err != nil {
		http.Error(w, "ent-admin: render page: "+err.Error(), http.StatusInternalServerError)
	}
}

func (a *App) renderDetailFragment(w http.ResponseWriter, ctx context.Context, page view.Page) {
	if err := ui.RenderDetailFragment(ctx, w, page); err != nil {
		http.Error(w, "ent-admin: render detail fragment: "+err.Error(), http.StatusInternalServerError)
	}
}

func (a *App) match(requestPath string) (routeMatch, bool) {
	relative, ok := a.relativePath(requestPath)
	if !ok {
		return routeMatch{}, false
	}
	if relative == "/" {
		return routeMatch{kind: routeIndex}, true
	}
	if relative == vsnAssetPath {
		return routeMatch{kind: routeAsset}, true
	}

	trimmed := strings.Trim(relative, "/")
	if trimmed == "" || strings.Contains(trimmed, "//") {
		return routeMatch{}, false
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) < 1 || len(parts) > 2 {
		return routeMatch{}, false
	}
	route, ok := decodeSegment(parts[0])
	if !ok {
		return routeMatch{}, false
	}
	entity, ok := a.registry.Find(route)
	if !ok {
		return routeMatch{}, false
	}
	if len(parts) == 1 {
		return routeMatch{kind: routeList, entity: entity}, true
	}
	id, ok := decodeSegment(parts[1])
	if !ok {
		return routeMatch{}, false
	}
	return routeMatch{kind: routeDetail, entity: entity, id: id}, true
}

func (a *App) relativePath(requestPath string) (string, bool) {
	if requestPath == "" || !strings.HasPrefix(requestPath, "/") {
		return "", false
	}
	if a.basePath == "/" {
		return requestPath, true
	}
	if requestPath == a.basePath || requestPath == a.basePath+"/" {
		return "/", true
	}
	prefix := a.basePath + "/"
	if !strings.HasPrefix(requestPath, prefix) {
		return "", false
	}
	relative := strings.TrimPrefix(requestPath, a.basePath)
	if strings.Contains(relative, "//") {
		return "", false
	}
	return relative, true
}

func decodeSegment(segment string) (string, bool) {
	decoded, err := url.PathUnescape(segment)
	if err != nil || decoded == "" || strings.ContainsAny(decoded, "/?#") {
		return "", false
	}
	return decoded, true
}

func normalizeBasePath(raw string) (string, error) {
	if raw == "" || raw == "/" {
		return "/", nil
	}
	if !strings.HasPrefix(raw, "/") {
		return "", fmt.Errorf("ent-admin: BasePath %q must start with '/'; got %q", raw, raw)
	}
	if strings.ContainsAny(raw, "?#") {
		return "", fmt.Errorf("ent-admin: BasePath %q must not contain a query or fragment", raw)
	}
	if strings.Contains(raw, "//") {
		return "", fmt.Errorf("ent-admin: BasePath %q contains an empty path segment", raw)
	}
	normalized := strings.TrimRight(raw, "/")
	if normalized == "" {
		return "/", nil
	}
	for _, segment := range strings.Split(strings.TrimPrefix(normalized, "/"), "/") {
		if segment == "." || segment == ".." || segment == "" {
			return "", fmt.Errorf("ent-admin: BasePath %q contains an invalid path segment", raw)
		}
		decoded, err := url.PathUnescape(segment)
		if err != nil {
			return "", fmt.Errorf("ent-admin: BasePath %q contains an invalid escape: %w", raw, err)
		}
		if decoded == "." || decoded == ".." || decoded == "" || strings.ContainsAny(decoded, "/?#") {
			return "", fmt.Errorf("ent-admin: BasePath %q contains an invalid escaped path segment", raw)
		}
	}
	return normalized, nil
}

func validateRegistry(registry Registry) error {
	routes := make(map[string]string, len(registry.Entities))
	for _, entity := range registry.Entities {
		if entity.Name == "" {
			return errors.New("ent-admin: registry entity name is empty")
		}
		if entity.Label == "" || entity.PluralLabel == "" {
			return fmt.Errorf("ent-admin: entity %q must define singular and plural labels", entity.Name)
		}
		if entity.Route == "" || strings.ContainsAny(entity.Route, "/?#") || entity.Route == "." || entity.Route == ".." {
			return fmt.Errorf("ent-admin: entity %q has invalid route %q", entity.Name, entity.Route)
		}
		if previous, exists := routes[entity.Route]; exists {
			return fmt.Errorf("ent-admin: entities %q and %q share route %q", previous, entity.Name, entity.Route)
		}
		routes[entity.Route] = entity.Name
		if nilReader(entity.Reader) {
			return fmt.Errorf("ent-admin: entity %q has no reader", entity.Name)
		}
		if entity.ID.Name == "" || entity.ID.Kind == "" {
			return fmt.Errorf("ent-admin: entity %q has an incomplete ID descriptor", entity.Name)
		}
		fields := make(map[string]struct{}, len(entity.Fields))
		for _, field := range entity.Fields {
			if field.Name == "" || field.Kind == "" {
				return fmt.Errorf("ent-admin: entity %q has an incomplete field descriptor", entity.Name)
			}
			if field.Name == entity.ID.Name {
				return fmt.Errorf("ent-admin: entity %q repeats its ID field %q", entity.Name, field.Name)
			}
			if _, exists := fields[field.Name]; exists {
				return fmt.Errorf("ent-admin: entity %q repeats field %q", entity.Name, field.Name)
			}
			fields[field.Name] = struct{}{}
		}
	}
	return nil
}

func nilReader(reader Reader) bool {
	if reader == nil {
		return true
	}
	value := reflect.ValueOf(reader)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func toViewEntity(entity Entity) view.Entity {
	fields := make([]view.Field, 0, len(entity.Fields))
	for _, field := range entity.Fields {
		fields = append(fields, view.Field{
			Name:      field.Name,
			Label:     field.Label,
			Kind:      string(field.Kind),
			Sensitive: field.Sensitive,
		})
	}
	return view.Entity{
		Name:        entity.Name,
		Label:       entity.Label,
		PluralLabel: entity.PluralLabel,
		Route:       entity.Route,
		Fields:      fields,
	}
}

func toViewRecords(records []Record) []view.Record {
	result := make([]view.Record, 0, len(records))
	for _, record := range records {
		result = append(result, toViewRecord(record))
	}
	return result
}

func toViewRecord(record Record) view.Record {
	return view.Record{ID: record.ID, Values: record.Values}
}
