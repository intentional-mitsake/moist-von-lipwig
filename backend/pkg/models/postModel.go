package models

import (
	"moist-von-lipwig/pkg/config"
	"time"
)

type Post struct {
	//`db:"post_id"`maps the column name in the database to the struct field
	PostID      string              `db:"post_id"`
	Sender      string              `db:"sender"`
	AccessPairs []config.AccessPair `db:"access_pairs"`
	Email       string              `db:"email"`
	Message     string              `db:"message"`
	Attachments []string            `db:"attachments"`
	Images      []string            `db:"images"`
	CreatedAt   time.Time           `db:"created_at"`
	Delivery    time.Time           `db:"delivery"`
	IsDelivered bool                `db:"is_delivered"`
}

//to do
/*
add a scheduled(bool) field to the table to check if the post was scheduled or not
initialize it to false, when a post is sheduled in services.ScheduleDelivery set it true
this var should be true before IsDelivered as that is only true once delivered
sceduled needs to be true once a cron job has been scheduled
this way when a GetDeliveryDates is called we will fetch WHERE IsDelivered = false AND scheduled = false
so that we only get unscheduled posts to prevent double scheduling
tehre is the problem of server crashing after the cron job has been scheduled but before it has been run
and as i havent figured out a workaround yet i will leave THIS for now
rate limiting needs to be added
*/
