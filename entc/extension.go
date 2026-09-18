// Package entadminentc integrates Ent Admin with Ent's code generator.
package entadminentc

import (
	"embed"
	"fmt"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"entgo.io/ent/schema/field"
)

//go:embed template/admin.tmpl
var templateFS embed.FS

// Extension returns the Ent Admin extension to pass to entc.Extensions.
func Extension() entc.Extension {
	return &extension{}
}

type extension struct {
	entc.DefaultExtension
}

func (*extension) Hooks() []gen.Hook {
	return []gen.Hook{validateGraph}
}

func (*extension) Templates() []*gen.Template {
	return []*gen.Template{
		gen.MustParse(gen.NewTemplate("entadmin").Funcs(template.FuncMap{
			"adminFieldKind":  fieldKind,
			"adminFieldValue": fieldValue,
			"adminIDParser":   idParser,
			"adminLabel":      humanLabel,
			"adminNodes":      orderedNodes,
		}).ParseFS(templateFS, "template/admin.tmpl")),
	}
}

func validateGraph(next gen.Generator) gen.Generator {
	return gen.GenerateFunc(func(g *gen.Graph) error {
		for _, node := range g.Nodes {
			if node.IsView() {
				return fmt.Errorf("ent-admin: entity %q is a view; views are not supported in phase one", node.Name)
			}
			if !node.HasOneFieldID() {
				return fmt.Errorf("ent-admin: entity %q must have one supported ID field", node.Name)
			}
			if err := validateID(node); err != nil {
				return err
			}
			if _, err := fieldKind(node.ID); err != nil {
				return fmt.Errorf("ent-admin: entity %q: %w", node.Name, err)
			}
			for _, f := range node.Fields {
				if _, err := fieldKind(f); err != nil {
					return fmt.Errorf("ent-admin: entity %q field %q: %w", node.Name, f.Name, err)
				}
			}
			if err := validateRoute(node.Table()); err != nil {
				return fmt.Errorf("ent-admin: entity %q: %w", node.Name, err)
			}
		}
		return next.Generate(g)
	})
}

func validateID(node *gen.Type) error {
	if node == nil || node.ID == nil {
		return fmt.Errorf("entity has no supported ID field")
	}
	switch {
	case node.ID.IsString(), node.ID.IsInt(), node.ID.IsInt64():
		return nil
	default:
		return fmt.Errorf("ent-admin: entity %q ID type %q cannot be parsed from a URL in phase one", node.Name, node.ID.Type)
	}
}

func validateRoute(route string) error {
	if route == "" || strings.ContainsAny(route, "/?#") {
		return fmt.Errorf("route %q must be a non-empty single path segment", route)
	}
	return nil
}

func fieldKind(f *gen.Field) (string, error) {
	if f == nil || f.Type == nil {
		return "", fmt.Errorf("field has no type information")
	}
	switch {
	case f.IsString():
		return "entadmin.FieldKindString", nil
	case f.IsInt():
		return "entadmin.FieldKindInt", nil
	case f.Type.Type == field.TypeInt8 || f.Type.Type == field.TypeInt16 || f.Type.Type == field.TypeInt32:
		return "entadmin.FieldKindInt", nil
	case f.IsInt64():
		return "entadmin.FieldKindInt64", nil
	case f.Type.Type == field.TypeUint || f.Type.Type == field.TypeUint8 || f.Type.Type == field.TypeUint16 || f.Type.Type == field.TypeUint32:
		return "entadmin.FieldKindUint", nil
	case f.Type.Type == field.TypeUint64:
		return "entadmin.FieldKindUint64", nil
	case f.Type.Type == field.TypeFloat32 || f.Type.Type == field.TypeFloat64:
		return "entadmin.FieldKindFloat64", nil
	case f.IsBool():
		return "entadmin.FieldKindBool", nil
	case f.IsTime():
		return "entadmin.FieldKindTime", nil
	case f.IsEnum():
		return "entadmin.FieldKindEnum", nil
	case f.IsUUID():
		return "entadmin.FieldKindUUID", nil
	case f.IsJSON():
		return "entadmin.FieldKindJSON", nil
	default:
		return "", fmt.Errorf("type %q is not supported by the phase-one generator", f.Type)
	}
}

func fieldValue(f *gen.Field, receiver string) string {
	ref := fmt.Sprintf("%s.%s", receiver, f.StructField())
	if f.Sensitive() {
		return "nil"
	}
	if f.NillableValue() {
		return fmt.Sprintf("func() any { if %s == nil { return nil }; return *%s }()", ref, ref)
	}
	return ref
}

func idParser(node *gen.Type) (string, error) {
	if err := validateID(node); err != nil {
		return "", err
	}
	idType := node.ID.Type.String()
	name := "entAdmin" + node.Name + "ID"
	switch {
	case node.ID.IsString():
		return fmt.Sprintf(`func %s(value string) (%s, error) {
	return %s(value), nil
}
`, name, idType, idType), nil
	case node.ID.IsInt():
		return fmt.Sprintf(`func %s(value string) (%s, error) {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return %s(0), fmt.Errorf("invalid %s id %%q: %%w", value, err)
	}
	return %s(parsed), nil
}
`, name, idType, idType, node.Name, idType), nil
	case node.ID.IsInt64():
		return fmt.Sprintf(`func %s(value string) (%s, error) {
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return %s(0), fmt.Errorf("invalid %s id %%q: %%w", value, err)
	}
	return %s(parsed), nil
}
`, name, idType, idType, node.Name, idType), nil
	default:
		return "", fmt.Errorf("entity %q ID type %q cannot be parsed", node.Name, node.ID.Type)
	}
}

func humanLabel(value string) string {
	words := strings.FieldsFunc(value, func(r rune) bool {
		return r == '_' || r == '-' || unicode.IsSpace(r)
	})
	for i, word := range words {
		if word == "" {
			continue
		}
		runes := []rune(strings.ToLower(word))
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

func orderedNodes(nodes []*gen.Type) []*gen.Type {
	ordered := append([]*gen.Type(nil), nodes...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Name < ordered[j].Name
	})
	return ordered
}
