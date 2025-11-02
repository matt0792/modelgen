package main

import (
	"log"

	"github.com/matt0792/modelgen"
	"github.com/matt0792/modelgen/examples/custom/api"
)

func main() {
	gen := modelgen.New("models")

	// Custom mapping with field renaming and omission
	err := gen.Register(&api.Account{}).
		MapField("ID", "ExternalId"). // Rename ID to ExternalId
		Omit("Name").                 // Exclude Name field from local model
		Omit("LegacyField").          // Exclude deprecated field
		Build()
	if err != nil {
		log.Fatal(err)
	}

	// Generate the code
	if err := gen.Generate("models"); err != nil {
		log.Fatal(err)
	}
}
