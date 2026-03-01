package main

import (
	"fmt"
	"log/slog"
	"moist-von-lipwig/pkg/database"
	"moist-von-lipwig/pkg/routes"
	"net/http"
	"os"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{}))
	port := os.Getenv("PORT")
	if port == "" {
		port = "8848"
	}
	addr := fmt.Sprint(":", port)
	db, err := database.OpenDB()
	if err != nil {
		logger.Error("Failed to open the DB connection", "error", err)
	}
	logger.Info("Connected to database", "db", db)
	router := routes.CreateRouter(db)
	//great thing about this create is that it creates the tables only if they dont exist
	err = database.CreateTables(db) //ignores this command if tables exist
	if err != nil {
		logger.Error("Failed to create tables", "error", err)
	}
	defer database.CloseDB(db)

	//CRON JOBS
	//	c := services.CronJobs(db)
	//defer c.Stop() //so that the cron jobs dont run forever and stop if the server crashes

	//listenandserve only returns error, thus unless the server crashes or we shut it, this wont be
	//displayed if its after the func
	logger.Info("Server starting", "address", addr)
	server := http.Server{
		Addr:    addr, //host:8848
		Handler: router,
	}
	if err := server.ListenAndServe(); err != nil {
		logger.Error("Server failed", "error", err)
		os.Exit(1)
	}
	fmt.Println("i just remembered noting is supposed to be hre")
}
