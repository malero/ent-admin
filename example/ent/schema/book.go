package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// Book is the minimal example entity used to prove generation and read-only
// metadata/adapters.
type Book struct {
	ent.Schema
}

func (Book) Fields() []ent.Field {
	return []ent.Field{
		field.String("title"),
		field.String("author"),
		field.Bool("published"),
	}
}
