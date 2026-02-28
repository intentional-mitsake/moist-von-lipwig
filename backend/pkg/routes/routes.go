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
	//create a fileserver so that static files can be served
	//every time a fiel is requested, server looks for it in the templates folder
	fs := http.FileServer(http.Dir("./templates"))
	//if the request arives with '/static/' , thsi will remvoe the '/static/' part
	//and search for the remaining part in fs-->./templates
	mux.Handle("/static/", http.StripPrefix("/static/", fs))
	//have to do all the fileserver thing cuz when the indexHandler is called
	//it reads the index.html, when it sees style.css is needed it cant find it wihtout hte above setup
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/db", dbHandler)
	return mux
}

var tmpl = template.Must(template.ParseFiles("templates/index.html"))

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl.Execute(w, nil)
}

func dbHandler(w http.ResponseWriter, r *http.Request) {
	DB := db.OpenDB()
	logger.Info("Connected to database", "db", DB)
}
