package main

import (
	"log"

	"github.com/matt0792/modelgen"
	"github.com/matt0792/modelgen/examples/everything/api"
)

func main() {
	gen := modelgen.New("models")

	// Custom mapping for Organization with field renaming and omission
	err := gen.Register(&api.Organization{}).
		MapField("ID", "OrgId").
		Omit("LegacyOrgCode"). // Exclude deprecated field
		Build()
	if err != nil {
		log.Fatal(err)
	}

	// Custom mapping for Account
	err = gen.Register(&api.Account{}).
		MapField("ID", "AccountId").
		Omit("Name"). // Exclude name from local model
		Build()
	if err != nil {
		log.Fatal(err)
	}

	// Simple mapping for Post
	gen.Map(&api.Post{})

	// Custom mapping for UserSettings
	err = gen.Register(&api.UserSettings{}).
		Omit("PrivateField"). // Exclude sensitive field
		Build()
	if err != nil {
		log.Fatal(err)
	}

	// Generate the code
	if err := gen.Generate("models"); err != nil {
		log.Fatal(err)
	}
}
