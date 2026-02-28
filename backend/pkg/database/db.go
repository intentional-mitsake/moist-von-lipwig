package database

import (
	"database/sql"
	"log/slog"
	"moist-von-lipwig/pkg/config"
	"os"

	_ "github.com/lib/pq"
)

func OpenDB() *sql.DB {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	dbCnfg := config.LoadDBConfig()
	db, err := sql.Open(dbCnfg.DBDriver, dbCnfg.DBSource)
	if err != nil {
		logger.Error("Failed to open database", "error", err)
		os.Exit(1)
	}
	//the above code doesnt really see if the creds are valid or the db conn is alive
	//it just validates that the format is right
	//need to ping to test if the connection is alive
	p := db.Ping()
	if p != nil {
		logger.Error("Failed to ping database", "error", p)
		os.Exit(1)
	}
	return db
}
