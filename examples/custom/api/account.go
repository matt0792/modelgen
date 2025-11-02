package api

// Account represents an external API account model
type Account struct {
	ID          int
	Name        string
	Username    string
	Email       string
	LegacyField string
}
