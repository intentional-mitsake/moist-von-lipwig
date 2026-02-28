package services

import (
	"net"
	"strings"
)

//check for delivery dates from the DB every 3 days
//this will vastly reduce the load on the server
//there is a risk of some posts being missed which are posted after the last check
//but have a delivery date before the next check
//for that reason, every time a post is made, if its delivery is before the next check,
// add it to the DB AND to the cache
//the cache will be used to check if any posts need to be delivered on the day
//send email if time has come and change is_delivered to true

func CheckDeliveryDates() {}

func SendEmails() {}

func DomainExists(email string) (bool, error) {
	_, domain := splitEmail(email)
	mx, err := net.LookupMX(domain) //chechks if teh part after @ is a valid domain
	return err == nil && len(mx) > 0, err
}
func splitEmail(email string) (string, string) {
	parts := strings.Split(email, "@")
	return parts[0], parts[1]
}
