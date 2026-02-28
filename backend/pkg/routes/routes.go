package routes

import (
	"html/template"
	db "moist-von-lipwig/pkg/database"
	lg "moist-von-lipwig/pkg/log"
	"net/http"
)

var logger = lg.CreateLogger()

func CreateRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/db", dbHandler)
	return mux
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/index.html"))
	tmpl.Execute(w, nil)
}

func dbHandler(w http.ResponseWriter, r *http.Request) {
	DB := db.OpenDB()
	logger.Info("Connected to database", "db", DB)
}
