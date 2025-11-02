package main

import (
	"log"

	"github.com/matt0792/modelgen"
	"github.com/matt0792/modelgen/examples/nested/api"
)

func main() {
	gen := modelgen.New("models")

	// Map nested structs - nested conversions are automatically handled
	gen.Map(&api.Blog{})
	gen.Map(&api.Author{})
	gen.Map(&api.Post{})

	// Generate the code
	if err := gen.Generate("models"); err != nil {
		log.Fatal(err)
	}
}
