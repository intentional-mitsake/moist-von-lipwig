package routes

import (
	"database/sql"
	"fmt"
	"html/template"
	"moist-von-lipwig/pkg/config"
	"moist-von-lipwig/pkg/database"
	lg "moist-von-lipwig/pkg/log"
	"moist-von-lipwig/pkg/models"
	"moist-von-lipwig/pkg/services"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

var logger = lg.CreateLogger()

// every route handler that needs to access the database will be a method of htis struct
type DBConfig struct {
	DBObj *sql.DB
}

func CreateRouter(db *sql.DB) http.Handler {
	mux := http.NewServeMux()
	dbCnfg := DBConfig{DBObj: db}
	//create a fileserver so that static files can be served
	//every time a fiel is requested, server looks for it in the templates folder
	fs := http.FileServer(http.Dir("./templates"))
	//if the request arives with '/static/' , thsi will remvoe the '/static/' part
	//and search for the remaining part in fs-->./templates
	staticHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".css") {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		}
		fs.ServeHTTP(w, r)
	})
	mux.Handle("/static/", http.StripPrefix("/static/", staticHandler))
	//have to do all the fileserver thing cuz when the indexHandler is called
	//it reads the index.html, when it sees style.css is needed it cant find it wihtout hte above setup
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/db", dbCnfg.dbHandler)
	mux.HandleFunc("/post-letter", dbCnfg.postHandler)
	mux.HandleFunc("/access-post", dbCnfg.accessHandler)
	return mux
}

var tmpl = template.Must(template.ParseFiles("templates/index.html"))
var courierpg = template.Must(template.ParseFiles("templates/von-courier.html"))

func indexHandler(w http.ResponseWriter, r *http.Request) {
	//to allow only get requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tmpl.Execute(w, nil)
}

func (d *DBConfig) dbHandler(w http.ResponseWriter, r *http.Request) {
	if d.DBObj == nil {
		http.Error(w, "Database not initialized", http.StatusInternalServerError)
		return
	}

}

func (d *DBConfig) postHandler(w http.ResponseWriter, r *http.Request) {
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
	message := r.FormValue("message")
	email := r.FormValue("email")
	waybilIDs := r.Form["waybill-ids"] //map of all the access ids cux its an array
	keys := r.Form["key"]              //map of all the keys cux its an array
	//files := r.MultipartForm.File["files"] //map of all the files cux its an array
	//imgs := r.MultipartForm.File["images"] //map of all the images cux its an array
	logger.Info("Post request received")
	now, time := services.Schedule()
	//logger.Info("Delivery Time: ", time)
	attachmentsPath, err := services.HandleFiles("attachments", r)
	if err != nil {
		http.Error(w, "Failed to handle files", http.StatusBadRequest)
		//logged it in the HandleFiles func already
	}
	imguploadsPath, err := services.HandleFiles("images", r)
	if err != nil {
		http.Error(w, "Failed to handle files", http.StatusBadRequest)
	}
	hm, err := services.HashIns(message, d.DBObj)
	if err != nil {
		http.Error(w, "Failed to hash message", http.StatusBadRequest)
	}

	//fmt.Println(uploadsPath)
	new_post := models.Post{
		PostID:      uuid.New().String(),
		AccessPairs: []config.AccessPair{},
		Email:       email,
		Message:     hm,
		Attachments: attachmentsPath,
		Images:      imguploadsPath,
		CreatedAt:   now,
		Delivery:    time,
		IsDelivered: false,
	}
	for i, _ := range waybilIDs {
		hk, err := services.HashIns(keys[i], d.DBObj)
		if err != nil {
			http.Error(w, "Failed to hash key", http.StatusBadRequest)
		}
		new_post.AccessPairs = append(new_post.AccessPairs, config.AccessPair{
			WaybillID: waybilIDs[i],
			Key:       hk,
		})
	}
	logger.Info("Post: ",
		"Post ID: ", new_post.PostID,
		"Access Pairs: ", len(new_post.AccessPairs),
		"Message: ", new_post.Message,
		"Created At: ", new_post.CreatedAt,
		"Delivery: ", new_post.Delivery,
		"Is Delivered: ", new_post.IsDelivered,
	)
	err = database.InsertPost(d.DBObj, &new_post)
	if err != nil {
		http.Error(w, "Failed to insert post", http.StatusBadRequest)
		return
	}
	logger.Info("Post inserted successfully")
}

type data struct {
	Show        bool
	IsDelivered bool
	Response    string
	Delivery    time.Time
	Post        models.Post
}

func (d *DBConfig) accessHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		//need to allow get requests to return back to the main page
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		//http.Error(w, "Failed to parse form", http.StatusBadRequest)
		logger.Error("Failed to parse form", "error", err)
		return
	}
	waybill := r.FormValue("waybill")
	key := r.FormValue("key")
	logger.Info("Waybill request received", "waybill", waybill, "key", key)
	ap := config.AccessPair{
		WaybillID: waybill,
		Key:       key,
	}
	post, isDelivered, res, dt, err := database.CheckDeliveryStatus(d.DBObj, ap)
	if err != nil {
		//http.Error(w, "Failed to check delivery status", http.StatusBadRequest)
		logger.Error("Failed to check delivery status", "error", err)
	}
	var dd data
	switch res {
	case 1: //waybill not found
		//http.Error(w, "Waybill not found", http.StatusBadRequest)
		//logger.Info("Waybill not found")
		dd.Show = false
		dd.Response = "Waybill not found"
	case 2: //failed to check delivery status
		//http.Error(w, "Failed to check delivery status", http.StatusBadRequest)
		//logger.Error("Failed to check delivery status", "error", err)
		dd.Show = false
		dd.Response = "Failed to check delivery status"
	case 3: //key not matching
		//http.Error(w, "Key not matching", http.StatusBadRequest)
		//logger.Error("Key not matching", "error", err)
		dd.Show = false
		dd.Response = "Key not matching"
	case 4: //match found
		dd.Show = true
		dd.IsDelivered = isDelivered
		dd.Response = fmt.Sprintf("Delivery Status: %t", dd.IsDelivered)
		dd.Delivery = dt
		if dd.IsDelivered {
			//only show the post if its delivered
			dd.Post = post
		}
		logger.Info("Delivery Status: ", dd.IsDelivered, "Delivery: ", dd.Delivery)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = courierpg.Execute(w, dd)
	if err != nil {
		logger.Error("Template loading failed", "error", err)
	}
}
