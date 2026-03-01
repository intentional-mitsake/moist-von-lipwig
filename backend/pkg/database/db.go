package database

import (
	"database/sql"
	"log/slog"
	"moist-von-lipwig/pkg/config"
	"moist-von-lipwig/pkg/models"
	"os"
	"time"

	"encoding/json"

	"github.com/lib/pq"
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
	//exec gives *sql.Result, error
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS posts (
			post_id TEXT PRIMARY KEY,
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
	//fmt.Println("this is happening")
	//fmt.Println(res)
	return nil
}

func InsertPost(db *sql.DB, post *models.Post) error {
	//we have in the db access pairs as a JSONB. Jonb sllows us to store arrays, mpas etc in a single column otherwise not posssible
	//access pairs here tho is a slice and is not converted automatically to JSONB
	//marshla returns json of the slice
	jsonB, err := json.Marshal(post.AccessPairs)
	if err != nil {
		logger.Error("Failed to marshal access pairs", "error", err)
		return err
	}
	_, err = db.Exec( //banger of an error-->err was declared above, so if u use := here it gives error
		`INSERT INTO posts (post_id, access_pairs, email, message, attachments, images, created_at, delivery, is_delivered)
	    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);`,
		post.PostID,
		jsonB,
		post.Email,
		post.Message,
		pq.Array(post.Attachments),
		pq.Array(post.Images),
		post.CreatedAt,
		post.Delivery,
		post.IsDelivered,
	)
	if err != nil {
		logger.Error("Failed to insert post", "error", err)
		return err
	}
	return nil
}

func GetDeliveryDates(db *sql.DB) ([]config.Delivery, error) {
	rows, err := db.Query(
		`
		SELECT post_id, delivery, is_delivered FROM posts 
		WHERE delivery BETWEEN $1 AND $2;
		`,
		time.Now(),
		time.Now().Add(3*24*time.Hour),
	)
	if err != nil {
		logger.Error("Failed to get delivery dates", "error", err)
		return nil, err
	}
	var delivery []config.Delivery
	defer rows.Close() //closing the rows after we are done
	for rows.Next() {  //preps next row to rea
		var id string
		var date time.Time
		var isDelivered bool
		err := rows.Scan(&id, &date, &isDelivered)
		if err != nil {
			logger.Error("Error while scanning the rows", "error", err)
			return nil, err
		}
		delivery = append(delivery, config.Delivery{
			PostID:      id,
			Delivery:    date,
			IsDelivered: isDelivered,
		})
		if err = rows.Err(); err != nil {
			logger.Error("Error while scanning the rows", "error", err)
			return nil, err
		}
	}
	return delivery, nil
}
