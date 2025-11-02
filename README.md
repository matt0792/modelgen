# modelgen

A code generator for type-safe struct mappings in Go.

## Installation

```bash
go get github.com/matt0792/modelgen@latest
```

## Usage

Create a generator file (e.g., `gen/main.go`):

```go
package main

import (
	"log"
	"github.com/matt0792/modelgen"
)

func main() {
	gen := modelgen.New()

    gen.Map(&api.Account{})

	if err := gen.Generate("./models/generated"); err != nil {
		log.Fatal(err)
	}
}
```

Run the generator:

```bash
go run gen/main.go
```

This creates `account.go` with mapping methods:

```go
externalAccount := fetchFromAPI()

// Convert external model to internal model
account := (%Account{}).From(&externalAccount)

// Work with your internal model
account.Name = "Updated Name"

// Convert back (note: see limitations below)
updated := account.To()
```

## Status

**This project is incomplete and under active development**

### Known Issues

- **`To()` method is broken for nested struct mappings** - currently generates incorrect conversion code for structs containing other structs
- Limited error handling in generated code

## Example

Given these structs:

```go
// External API model
package api

type Account struct {
	ID       int
	Name     string
	Username string
	Posts    []Post
}

type Post struct {
	Title   string
	Content string
}
```

And this generator file:

```go
func generate() {
    gen := modelgen.New("models")

	// Manual mapping setup
	err := gen.Register(&api.Account{}).
		MapField("ID", "ExternalId"). // Create a custom mapping (api.Account.ID -> models.Account.ExternalId)
		Omit("Name").                 // Omit a field from our local model
		Build()
	if err != nil {
		panic(err)
	}

	// Create a default mapping
	gen.Map(&api.Post{})

	// Generate the code
	err = gen.Generate("models")
	if err != nil {
		panic(err)
	}
}
```

Running the generator creates:

models/account.go

```go
package models

type Account struct {
	ExternalId int
	Username   string
	Posts      []Post
}

// From maps from an external struct to a local
//
// Usage: localAccount := (&Account{}).From(&externalAccount)
func (t *Account) From(src *api.Account) *Account {
	if src == nil {
		return nil
	}

	return &Account{
		ExternalId: src.ID,
		Username:   src.Username,
		Posts: func() []Post {
			if src.Posts == nil {
				return nil
			}
			result := make([]Post, len(src.Posts))
			for i, item := range src.Posts {
				converted := (&Post{}).From(&item)
				if converted != nil {
					result[i] = *converted
				}
			}
			return result
		}(),
	}
}

// Usage: externalAccount := Account.To()
func (t *Account) To() api.Account {
	return api.Account{
		ID:       t.ExternalId,
		Name:     "",
		Username: t.Username,
		Posts:    t.Posts,
	}
}
```

models/post.go

```go
package models

type Post struct {
	Title   string
	Content string
}

// From maps from an external struct to a local
//
// Usage: localPost := (&Post{}).From(&externalPost)
func (t *Post) From(src *api.Post) *Post {
	if src == nil {
		return nil
	}

	return &Post{
		Title:   src.Title,
		Content: src.Content,
	}
}

// Usage: externalPost := Post.To()
func (t *Post) To() api.Post {
	return api.Post{
		Title:   t.Title,
		Content: t.Content,
	}
}
```

## License

MIT
