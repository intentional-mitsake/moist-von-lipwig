package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"moist-von-lipwig/pkg/config"
	"moist-von-lipwig/pkg/models"
	"os"
	"sync"
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
			sender TEXT,
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
	_, err = db.Exec(`
    CREATE INDEX IF NOT EXISTS apIndx ON posts USING GIN (access_pairs);
	`)
	if err != nil {
		logger.Error("Failed to created inexes", "error", err)
	}
	//fmt.Println("this is happening")
	//fmt.Println(res)
	logger.Info("Created tables")
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
		`INSERT INTO posts (post_id, sender, access_pairs, email, message, attachments, images, created_at, delivery, is_delivered)
	    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);`,
		post.PostID,
		post.Sender,
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
	//logger.Info("HERE")
	var delivery []config.Delivery
	defer rows.Close() //closing the rows after we are done
	for rows.Next() {  //preps next row to rea
		var id, email string
		var date time.Time
		var isDelivered bool
		err := rows.Scan(&id, &date, &isDelivered, &email)
		//logger.Info("Query", id, date, isDelivered, email)
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
		return
	}
	logger.Info("Delivery status changed", "postIDs", postIDs)
}

func CheckDeliveryStatus(db *sql.DB, accesspair config.AccessPair) (post []models.Post, res int, e error) {
	indx := `[{"WaybillID": "` + accesspair.WaybillID + `"}]`
	rows, err := db.Query(`
    SELECT post_id, sender, email, message, attachments, images, created_at, delivery, is_delivered, pair->>'Key'
    FROM posts, jsonb_array_elements(access_pairs) AS pair
    WHERE access_pairs @> $1
	AND pair->> 'WaybillID' = $2`, indx, accesspair.WaybillID)
	//using indexing sequential search can be removed here
	//didnt addd as much speed as i hoped
	//theres a noticable diff in speed when theres waybill and when theres no waybill
	// that was reduced by a lto after implementing async thru go routines
	//still theres a feel of slowness so tohugh indexing would help
	//realiseing now that unless i have to do a seq search of 10000000 rows, diff isnt much noticable from wihtout indexng
	//will keep it still
	//all in all its pretty fast now, go routines are reducing cpu time consumed by bcrypt and for loop for each row that matches
	//indexing is removing the need for sequential seach so much faster db access
	//logger.Info("Rows:", rows)
	if err != nil {

		logger.Error("Database query failed", "error", err)
		return []models.Post{}, 5, err
	}
	defer rows.Close()
	found := false
	var posts []models.Post
	var tempHashedPassword string
	var mu sync.Mutex
	var wg sync.WaitGroup
	for rows.Next() {
		//if theres no error AND there is a row that means at least one match
		found = true
		var post models.Post
		if err := rows.Scan(
			&post.PostID,
			&post.Sender,
			&post.Email,
			&post.Message,
			pq.Array(&post.Attachments),
			pq.Array(&post.Images),
			&post.CreatedAt,
			&post.Delivery,
			&post.IsDelivered,
			&tempHashedPassword,
		); err != nil {
			continue
		}
		logger.Info("Checking pair", "postID", post.PostID, "hash", tempHashedPassword)
		wg.Add(1) // incr wg counter
		go func(tempHashedPass string, key string) {
			defer wg.Done() //decr wg counter by 1
			//bcrypt compare itself costs a lot of cpu time so go routines needed to reduce that
			//the mre the salt the more cpu time it takes, but also takes longer for hackers to try brute force as compareing passwoirds takes more time
			err = bcrypt.CompareHashAndPassword([]byte(tempHashedPass), []byte(key))
			if err == nil {
				mu.Lock()
				// match found --> no error from query, at least one row, and no error from bcrypt(key matches)-->append
				posts = append(posts, post)
				//return post, tempIsDelivered, 4, tempDelivery, nil
				mu.Unlock()
			}
		}(tempHashedPassword, accesspair.Key)
	}
	wg.Wait() //blocks this func until wg counter is 0
	//basically each iteration counter is incr by 1 and when that iteration's go routine is done counter is decr by 1
	//**NOTE: counter isnt decr when the iteration is done, but when the go routine is done
	//the goroutines keep getting added throught the for loop, so the counter never reaches 0 untill all goroutines are done
	//this wait is called to hold this func until all goroutines are done
	if !found {
		logger.Error("Waybill not found", "error", err)
		return []models.Post{}, 1, err
	}
	if found {
		if len(posts) == 0 { //found rows but key did not matach so did not append to posts
			return []models.Post{}, 3, fmt.Errorf("invalid waybill or key")
		}
	}
	//if we get to this point-->if !found was false so we found at least one row
	//found is true means it found at least one row
	//so no ened to chcek for nil posts in routes.go, but better to have one so logged it here
	logger.Info("Posts: ", posts)
	return posts, 4, nil
}

func GetPost(db *sql.DB, postID string) (Post models.Post) {
	err := db.QueryRow(`
		SELECT post_id, sender, email, message, attachments, images, created_at, delivery, is_delivered
		FROM posts
		WHERE post_id = $1`, postID).Scan(
		&Post.PostID,
		&Post.Sender,
		&Post.Email,
		&Post.Message,
		pq.Array(&Post.Attachments),
		pq.Array(&Post.Images),
		&Post.CreatedAt,
		&Post.Delivery,
		&Post.IsDelivered,
	)
	if err != nil {
		logger.Error("Database query failed", "error", err)
		return models.Post{}
	}
	return Post
}
