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
	mux.HandleFunc("/post-letter", postHandler)
	return mux
}

var tmpl = template.Must(template.ParseFiles("templates/index.html"))

func indexHandler(w http.ResponseWriter, r *http.Request) {
	//to allow only get requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tmpl.Execute(w, nil)
}

func dbHandler(w http.ResponseWriter, r *http.Request) {
	DB := db.OpenDB()
	logger.Info("Connected to database", "db", DB)
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	//to allow only post requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	//multipart cuz theres files as well
	//parsing done to keep the size of the form small
	e := r.ParseMultipartForm(32 << 20)
	if e != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		logger.Error("Failed to parse form", "error", e)
		return
	}
	sender := r.FormValue("sender_email")
	message := r.FormValue("message")
	recipients := r.Form["recipients"]     //map of all the recipients cux its an array
	files := r.MultipartForm.File["files"] //map of all the files cux its an array
	imgs := r.MultipartForm.File["images"] //map of all the images cux its an array

	logger.Info("Post request received", "sender", sender, "recipients", recipients, "message", message, "files", files, "imgs", imgs)
}
