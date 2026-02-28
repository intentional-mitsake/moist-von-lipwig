package services

import (
	"database/sql"

	"golang.org/x/crypto/bcrypt"
)

// encrypt the messages, attachments, access pairs before storing in the db
const salt = 13

func HashIns(str string, db *sql.DB) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(str), salt)
	if err != nil {
		logger.Error("Error while hashing", "error", err)
		return "", err
	}
	return string(hash), nil
}
