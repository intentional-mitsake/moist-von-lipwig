package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"moist-von-lipwig/pkg/config"
	"moist-von-lipwig/pkg/models"
	"os"
	"time"

	"encoding/json"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
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
	rows, err := db.Query( //for efficiency only get the ones that are not delivered
		//changed to <=now+24hours cuz that way we will get all false state posts with delivery <= now+24
		//so anything missed will be covered as well
		`
		SELECT post_id, delivery, is_delivered, email FROM posts 
		WHERE delivery <= NOW() + INTERVAL '24 hours'
		AND is_delivered = false;
		`)
	if err != nil {
		logger.Error("Failed to get delivery dates", "error", err)
		return nil, err
	}
	var delivery []config.Delivery
	defer rows.Close() //closing the rows after we are done
	for rows.Next() {  //preps next row to rea
		var id, email string
		var date time.Time
		var isDelivered bool
		err := rows.Scan(&id, &date, &isDelivered, &email)
		if err != nil {
			logger.Error("Error while scanning the rows", "error", err)
			return nil, err
		}
		delivery = append(delivery, config.Delivery{
			PostID:      id,
			Delivery:    date,
			IsDelivered: isDelivered,
			Email:       email,
		})
		if err = rows.Err(); err != nil {
			logger.Error("Error while scanning the rows", "error", err)
			return nil, err
		}
	}
	return delivery, nil
}

func ChangeDeliveryStatus(db *sql.DB, postIDs []string) {
	_, err := db.Exec( //no need to change if already delivered
		`
		UPDATE posts
		SET is_delivered = true
		WHERE post_id = ANY($1)
		AND is_delivered = false;
		`,
		pq.Array(postIDs), //used interval to add 24 hours to get deliveries <= now AND 24hours from now as well
		//otherwise we miss deliveries to be made only hours later than the time CRONjob checked db
		//swriched to any(postIDs) so that instead of delivery time, it checks for postIDs
	)
	if err != nil {
		logger.Error("Failed to change delivery status", "error", err)
	}
}

func CheckDeliveryStatus(db *sql.DB, accesspair config.AccessPair) (post models.Post, status bool, res int, delivery time.Time, e error) {
	rows, err := db.Query(`
    SELECT post_id, delivery, is_delivered, pair->>'Key'
    FROM posts, jsonb_path_query(access_pairs, '$[*]') AS pair
    WHERE pair->>'WaybillID' = $1`, accesspair.WaybillID)
	if err != nil {
		logger.Error("Database query failed", "error", err)
		return models.Post{}, false, 2, time.Time{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var tempDelivery time.Time
		var tempIsDelivered bool
		var tempHashedPassword string
		var postID string

		if err := rows.Scan(&postID, &tempDelivery, &tempIsDelivered, &tempHashedPassword); err != nil {
			continue
		}
		err = bcrypt.CompareHashAndPassword([]byte(tempHashedPassword), []byte(accesspair.Key))
		if err == nil {
			// match found
			var post models.Post
			err = db.QueryRow(`
			SELECT post_id, email, message, attachments, images, created_at, delivery, is_delivered
			FROM posts
			WHERE post_id = $1`, postID).Scan(
				&post.PostID,
				&post.Email,
				&post.Message,
				pq.Array(&post.Attachments),
				pq.Array(&post.Images),
				&post.CreatedAt,
				&post.Delivery,
				&post.IsDelivered,
			)
			if err != nil {
				logger.Error("Database query failed", "error", err)
				return models.Post{}, false, 2, time.Time{}, err
			}
			return post, tempIsDelivered, 4, tempDelivery, nil
		}
	}

	//no match
	return models.Post{}, false, 3, time.Time{}, fmt.Errorf("invalid waybill or key")
}
