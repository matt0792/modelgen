package api

// Blog represents an external API blog model
type Blog struct {
	ID     int
	Title  string
	Author Author
	Posts  []Post
}

// Author represents a blog author
type Author struct {
	Name  string
	Email string
}

// Post represents a blog post
type Post struct {
	Title   string
	Content string
	Tags    []string
}
