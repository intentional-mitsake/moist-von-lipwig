package services

import (
	"database/sql"
	"fmt"
	"moist-von-lipwig/pkg/config"
	"moist-von-lipwig/pkg/database"
	"net"
	"strings"

	"github.com/robfig/cron"
)

//check for delivery dates from the DB every 3 days
//this will vastly reduce the load on the server
//there is a risk of some posts being missed which are posted after the last check
//but have a delivery date before the next check
//for that reason, every time a post is made, if its delivery is before the next check,
// add it to the DB AND to the cache
//the cache will be used to check if any posts need to be delivered on the day
//send email if time has come and change is_delivered to true

func CronJobs(db *sql.DB) *cron.Cron {
	c := cron.New()
	err := c.AddFunc("@every 30s", func() {
		schedule := CheckDeliveryDates(db)
		fmt.Println(schedule)
		//fmt.Println(time.Duration(time.Now()))
		//fmt.Println(time.Now().Add(3 * 24 * time.Hour))
	})
	if err != nil {
		logger.Error("Error running cron job", "error", err)
	}
	c.Start()
	logger.Info("Cron starting", "cron", c)
	return c
}

func CheckDeliveryDates(db *sql.DB) []config.Delivery {
	schedule, _ := database.GetDeliveryDates(db)
	//already logged the error and no need to send that to the user
	return schedule
}

func DomainExists(email string) (bool, error) {
	_, domain := splitEmail(email)
	mx, err := net.LookupMX(domain) //chechks if teh part after @ is a valid domain
	return err == nil && len(mx) > 0, err
}
func splitEmail(email string) (string, string) {
	parts := strings.Split(email, "@")
	return parts[0], parts[1]
}
