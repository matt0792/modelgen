package api

// Organization represents an external API organization model
type Organization struct {
	ID            int
	Name          string
	LegacyOrgCode string
	Accounts      []Account
}

// Account represents an external API account model
type Account struct {
	ID       int
	Name     string
	Username string
	Email    string
	Posts    []Post
	Settings UserSettings
}

// Post represents a user post
type Post struct {
	Title   string
	Content string
	Tags    []string
}

// UserSettings represents account settings
type UserSettings struct {
	Theme         string
	Notifications bool
	PrivateField  string
}
