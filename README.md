# modelgen

A code generator for type-safe struct mappings in Go.

## Installation

```bash
go get github.com/matt0792/modelgen@latest
```

## Usage

### Overview

Create a model generator, register a mapping, and generate the code:
```go
gen := modelgen.New("models") // "models" is the output package that will be created

gen.Map(&api.User{}) // api.User represents an imported struct type

gen.Generate("modelgen") // "modelgen" is the output dir
```

Now we can do things like: 
```go
externalUser := fetchFromAPI()

// Convert external model to internal model
user := (%models.User{}).From(&externalUser)

// Work with your internal model
user.Name = "Updated Name"

// Convert back (see limitations below)
updated := user.To()
```

### Practical usage

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

## Status

**This project is incomplete and under active development**

### Known Issues

- **`To()` method is broken for nested struct mappings** - currently generates incorrect conversion code for structs containing other structs
- Limited error handling in generated code

## License

MIT
