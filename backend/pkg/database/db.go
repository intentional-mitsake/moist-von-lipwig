package database

import (
	"database/sql"
	"log/slog"
	"moist-von-lipwig/pkg/config"
	"moist-von-lipwig/pkg/models"
	"os"

	_ "github.com/lib/pq"
)

var logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))

func OpenDB() (*sql.DB, error) {
	dbCnfg := config.LoadDBConfig()
	db, err := sql.Open(dbCnfg.DBDriver, dbCnfg.DBSource)
	if err != nil {
		logger.Error("Failed to open database", "error", err)
		return nil, err
	}
	//the above code doesnt really see if the creds are valid or the db conn is alive
	//it just validates that the format is right
	//need to ping to test if the connection is alive
	p := db.Ping()
	if p != nil {
		logger.Error("Failed to ping database", "error", p)
		return nil, p
	}
	return db, nil
}

func CloseDB(db *sql.DB) error {
	err := db.Close()
	if err != nil {
		logger.Error("Failed to close database", "error", err)
		return err
	}
	return nil
}

func CreateTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS posts (
			post_id SERIAL PRIMARY KEY,
			access_pairs JSONB,
			email TEXT,
			message TEXT,
			attachments TEXT[],
			images TEXT[],
			created_at TIMESTAMP,
			delivery TIMESTAMP,
			is_delivered BOOLEAN
		);
	`)
	if err != nil {
		logger.Error("Failed to create table", "error", err)
		return err
	}
	return nil
}

func InsertPost(db *sql.DB, post *models.Post) error {
	_, err := db.Exec(`
	 INSERT INTO posts (access_pairs, email, message, attachments, images, created_at, delivery, is_delivered)
	 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, post.AccessPairs, post.Email, post.Message, post.Attachments, post.Images, post.CreatedAt, post.Delivery, post.IsDelivered)
	if err != nil {
		logger.Error("Failed to insert post", "error", err)
		return err
	}
	return nil
}
