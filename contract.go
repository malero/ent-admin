// Package entadmin provides a small, read-only administrative UI runtime for
// metadata and readers generated from Ent schemas.
package entadmin

import (
	"context"
	"errors"
)

// FieldKind identifies the phase-one value families that the generic runtime
// knows how to format. Generators may expose additional kinds in later phases.
type FieldKind string

const (
	FieldKindString  FieldKind = "string"
	FieldKindInt     FieldKind = "int"
	FieldKindInt64   FieldKind = "int64"
	FieldKindUint    FieldKind = "uint"
	FieldKindUint64  FieldKind = "uint64"
	FieldKindFloat64 FieldKind = "float64"
	FieldKindBool    FieldKind = "bool"
	FieldKindTime    FieldKind = "time"
	FieldKindEnum    FieldKind = "enum"
	FieldKindUUID    FieldKind = "uuid"
	FieldKindJSON    FieldKind = "json"
)

// Field describes one schema field that can be displayed by the generic UI.
// Fields are kept in schema order by generated registries. The entity ID is
// described separately by Entity.ID and is not required to appear in Fields.
type Field struct {
	Name       string
	Label      string
	Kind       FieldKind
	Optional   bool
	Nillable   bool
	Immutable  bool
	Sensitive  bool
	EnumValues []string
}

// Entity is the generated, UI-neutral description of one Ent entity.
type Entity struct {
	Name        string
	Label       string
	PluralLabel string
	Route       string
	ID          Field
	Fields      []Field
	Reader      Reader
}

// Registry is the complete generated admin surface supplied to the runtime.
type Registry struct {
	Entities []Entity
}

// Find returns the entity registered at route. Route matching is exact; the
// runtime owns URL normalization and passes the canonical route segment.
func (r Registry) Find(route string) (Entity, bool) {
	for _, entity := range r.Entities {
		if entity.Route == route {
			return entity, true
		}
	}
	return Entity{}, false
}

// Record is a generic read result. Values are keyed by Field.Name so the UI
// can render records using the descriptor without generated HTML.
type Record struct {
	ID     string
	Values map[string]any
}

// ListOptions intentionally contains only pagination in phase one. Filtering,
// sorting, and search are future contracts rather than implicit behavior.
type ListOptions struct {
	Offset int
	Limit  int
}

// ListResult contains the records returned for one page.
type ListResult struct {
	Records []Record
}

// Reader is the narrow per-entity read-only adapter generated alongside the
// descriptor. Generated implementations bind a concrete Ent client when the
// registry is constructed, so the runtime never accepts an untyped client.
type Reader interface {
	List(context.Context, ListOptions) (ListResult, error)
	Get(context.Context, string) (Record, error)
}

// ErrNotFound is returned by generated readers when an Ent record does not
// exist. The runtime maps it to an HTTP 404 response.
var ErrNotFound = errors.New("ent-admin: record not found")
