package services

import (
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
)

func Schedule() (time.Time, time.Time, string) { //now, DT time.Time {
	now := time.Now() //get current time
	//fmt.Println(now)
	//10 * 365 = 3650 days--> 24 * 3650 = 87600 hours
	randTime := rand.IntN(87600) //theres a limit to postgres' date field
	//plus i dont really want it to be too far in the future
	//fmt.Println(time.Duration(randTime) * time.Hour)
	deliveryTime := now.Add(time.Duration(randTime) * time.Hour) //add random time to current tim
	//fmt.Println(deliveryTime)
	//generate a postID here as well
	id := uuid.New()
	postID := id.String()
	return now, deliveryTime, postID
}
