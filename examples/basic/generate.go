package main

import (
	"log"

	"github.com/matt0792/modelgen"
	"github.com/matt0792/modelgen/examples/basic/api"
)

func main() {
	gen := modelgen.New("models")

	// Create a default mapping with all fields
	gen.Map(&api.User{})

	// Generate the code
	if err := gen.Generate("models"); err != nil {
		log.Fatal(err)
	}
}
