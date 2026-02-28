package models

type Post struct {
	PostID      int
	AccessIDs   []string
	Keys        []string
	Message     string
	Attachments []string
	Images      []string
}
