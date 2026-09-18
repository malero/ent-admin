package entadminentc

import (
	"bytes"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"entgo.io/ent/schema/field"
)

func TestExtensionGeneratesDeterministicRegistry(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller returned no source path")
	}
	repoRoot := filepath.Dir(filepath.Dir(sourceFile))
	schemaPath := filepath.Join(repoRoot, "example", "ent", "schema")

	targets := []string{t.TempDir(), t.TempDir()}
	outputs := make([][]byte, 0, len(targets))
	for _, target := range targets {
		err := entc.Generate(
			schemaPath,
			&gen.Config{
				Target:  target,
				Package: "github.com/malero/ent-admin/example/ent",
			},
			entc.Extensions(Extension()),
		)
		if err != nil {
			t.Fatalf("generate Ent Admin output: %v", err)
		}

		path := filepath.Join(target, "entadmin.go")
		output, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read generated output: %v", err)
		}
		if _, err := parser.ParseFile(token.NewFileSet(), path, output, parser.ParseComments); err != nil {
			t.Fatalf("parse generated output: %v", err)
		}
		outputs = append(outputs, output)
	}

	if !bytes.Equal(outputs[0], outputs[1]) {
		t.Fatal("repeated generation produced different entadmin.go output")
	}

	generated := string(outputs[0])
	for _, fragment := range []string{
		`Name:        "Book"`,
		`Route:       "books"`,
		`FieldKindInt`,
		`FieldKindString`,
		`Reader: &entAdminBookReader`,
		`r.client.Book.Query()`,
		`r.client.Book.Get(ctx, parsed)`,
	} {
		if !strings.Contains(generated, fragment) {
			t.Errorf("generated output is missing %q", fragment)
		}
	}
}

func TestExtensionRejectsUnsupportedID(t *testing.T) {
	node := &gen.Type{
		Name: "Widget",
		ID:   &gen.Field{Type: &field.TypeInfo{Type: field.TypeBool}},
	}
	if err := validateID(node); err == nil {
		t.Fatal("validateID unexpectedly accepted a boolean ID")
	}
}
