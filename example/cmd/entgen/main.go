package main

import (
	"log"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"

	entadminentc "github.com/malero/ent-admin/entc"
)

func main() {
	if err := entc.Generate(
		"./example/ent/schema",
		&gen.Config{Target: "./example/ent"},
		entc.Extensions(entadminentc.Extension()),
	); err != nil {
		log.Fatal(err)
	}
}
