package services

import (
	"database/sql"
	"fmt"
	"moist-von-lipwig/pkg/config"
	"moist-von-lipwig/pkg/database"
	"net"
	"strings"
	"time"

	"github.com/robfig/cron"
)

//check for delivery dates from the DB every 3 days(decided daily now, perf impact wont be much)
//this will vastly reduce the load on the server
//there is a risk of some posts being missed which are posted after the last check
//but have a delivery date before the next check
//for that reason, every time a post is made, if its delivery is before the next check,
// add it to the DB AND to the cache
//the cache will be used to check if any posts need to be delivered on the day
//send email if time has come and change is_delivered to true

func CronJobs(db *sql.DB) *cron.Cron {
	c := cron.New()
	var schedule []config.Delivery
	err := c.AddFunc("@every 20s", func() { //30s only for debugging-->should be 3 days in prod
		schedule = CheckDeliveryDates(db)
		//fmt.Println(schedule)
		//fmt.Println(time.Duration(time.Now()))
		//fmt.Println(time.Now().Add(3 * 24 * time.Hour))
		ScheduleDelivery(c, db, schedule)
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

func ScheduleDelivery(c *cron.Cron, db *sql.DB, schedule []config.Delivery) {
	//cron format: "sec min hr dom mon dow" there are others but we will use this
	//eg: "10 30 12 11 13 *" -> 10th sec, 30th min, 12th hr, 11th day, 13th month, *(skips the day of the week)
	//i wiil convert the delivery date to cron format here to prepare a specific schedule
	scheduledDates := make(map[string][]string) //[cronFormat]postID
	emailMap := make(map[string]string)         //[postID]email
	for _, post := range schedule {
		p := post
		//mt.Println(day)
		//fmt.Println(month)
		scheduledDate := fmt.Sprintf("50 57 10 %d %d *", //sec min hr dom mon dow--> 0 0 0 day month *
			//precision down to the seconds doesnt matter
			p.Delivery.Day(),
			p.Delivery.Month(),
		)
		emailMap[p.PostID] = p.Email //each post has an email and is unique
		//IF DELIVERY DATE IS PAST
		if isDeliveryPast(p.Delivery) {
			//deliver immediately if delivery date is past
			go func() { //have to ues local var for ids cuz risks of race condition
				//can use emailmap cuz it will be the same as isnt changed in loop iterations
				email := emailMap[p.PostID] //get the email of this postID
				go SendEmail(email)         //mutliple posts are sent at same schedule so run parallel to reduce load
				database.ChangeDeliveryStatus(db, []string{p.PostID})
			}()
		} else { //only add to map if delivery date is not past
			//for efficiency have a list of scheduled dates to group same day deliveries
			if scheduledDates[scheduledDate] == nil { //if the date is not in the map, add it
				scheduledDates[scheduledDate] = []string{p.PostID}
			} else { //if the date is in the map, append this postID to existing list of postIDs
				scheduledDates[scheduledDate] = append(scheduledDates[scheduledDate], p.PostID)
			}
		}

	}
	fmt.Println(scheduledDates)

	//most efficient will be to have one cron job for all the same dates
	for cronShedule, postIDs := range scheduledDates {
		//have to declare new variables inside the loop cuz
		//addFunc is not run here; its only scheduled here, by the time it runs, the loop will be over
		//and cuz thel oop is over it uses the last postIDs only
		currentSchedule := cronShedule
		currentPostIDs := postIDs
		c.AddFunc(currentSchedule, func() {
			for _, postID := range currentPostIDs {
				email := emailMap[postID] //get the email of this postID
				go SendEmail(email)       //mutliple posts are sent at same schedule so run parallel to reduce load
			}
			//doesnt need to be called for each post as they are all scheduled for the same date
			database.ChangeDeliveryStatus(db, currentPostIDs) //this way even delivery dates we missed previously will be marked as delivered
		})
	}

}

func isDeliveryPast(schedule time.Time) bool {
	now := time.Now()
	return schedule.Before(now) //returns true if schedule is before now
	//false if schedule is after cuurent day
}

func SendEmail(email string) {
	fmt.Println(email)
}
