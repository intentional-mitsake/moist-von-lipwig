package models

import "time"

type AccessPair struct {
	WaybillID string
	Key       string
}

type Post struct {
	//`db:"post_id"`maps the column name in the database to the struct field
	PostID      string       `db:"post_id"`
	AccessPairs []AccessPair `db:"access_pairs"`
	Email       string       `db:"email"`
	Message     string       `db:"message"`
	Attachments []string     `db:"attachments"`
	Images      []string     `db:"images"`
	CreatedAt   time.Time    `db:"created_at"`
	Delivery    time.Time    `db:"delivery"`
	IsDelivered bool         `db:"is_delivered"`
}
