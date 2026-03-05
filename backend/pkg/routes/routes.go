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

type Data struct {
	Show        bool
	IsDelivered bool
	Response    string
	Delivery    time.Time
	Post        models.Post
	Choices     []config.Choices
}

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
	us := http.FileServer(http.Dir("./uploads"))
	//if the request arives with '/static/' , thsi will remvoe the '/static/' part
	//and search for the remaining part in fs-->./templates
	staticHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".css") {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		}
		fs.ServeHTTP(w, r)
	})
	mux.Handle("/uploads", http.StripPrefix("/uploads/", us))
	mux.Handle("/static/", http.StripPrefix("/static/", staticHandler))
	//have to do all the fileserver thing cuz when the indexHandler is called
	//it reads the index.html, when it sees style.css is needed it cant find it wihtout hte above setup
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/db", dbCnfg.dbHandler)
	mux.HandleFunc("/post-letter", dbCnfg.postHandler)
	mux.HandleFunc("/access-post", dbCnfg.accessHandler)
	mux.HandleFunc("/access-id", dbCnfg.idHandler)
	return mux
}

var tmpl = template.Must(template.ParseFiles("templates/index.html"))
var courierpg = template.Must(template.ParseFiles("templates/von-courier.html"))
var posted = template.Must(template.ParseFiles("templates/posted.html"))

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
	if r.Method == http.MethodGet {
		//need to allow get requests to return back to the main page
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
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
	sender := r.FormValue("sender")
	//due to validatio funcs in js, this really cant happen thru html
	//only way to trigger wuld be to skip the site(js validation) and submit data thru smth like postman
	//still possible to send invalid stuff so at least check if its empty to save time
	if (message == "") || (email == "") || (waybilIDs == nil) || (keys == nil) || (sender == "") {
		http.Error(w, "Invalid Inputs", http.StatusBadRequest)
		logger.Error("Invalid Input", "Email", email, "Message", message, "Waybill IDs", waybilIDs, "Keys", keys, "Sender", sender)
		return
	}
	for i := range waybilIDs {
		waybilIDs[i] = strings.TrimSpace(waybilIDs[i]) //to remove whitespace from ends and begining
		keys[i] = strings.TrimSpace(keys[i])
	}
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
	//bcrypt is one way encryption
	//hm, err := services.AESEncrypt([]byte(message))

	//fmt.Println(uploadsPath)
	new_post := models.Post{
		PostID:      uuid.New().String(),
		Sender:      sender,
		AccessPairs: []config.AccessPair{},
		Email:       email,
		Message:     message,
		Attachments: attachmentsPath,
		Images:      imguploadsPath,
		CreatedAt:   now,
		Delivery:    time,
		IsDelivered: false,
	}
	for i, _ := range waybilIDs {
		hk, err := services.HashIns(keys[i])
		if err != nil {
			http.Error(w, "Failed to hash key", http.StatusInternalServerError)
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
		http.Error(w, "Failed to insert post", http.StatusInternalServerError)
		return
	}
	logger.Info("Post inserted successfully")
	postID := new_post.PostID
	w.WriteHeader(http.StatusCreated)
	posted.Execute(w, postID)
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
	//decided not to use postid here for various reasons
	//1. its long and hard to remember even tho its unique and can be used to identify the post much faster
	//2. access pairs seemed an interesting idea
	//the only prob with it is that if somehow two users have the same access pair, they will both be able to see only one posts status
	//for that postid would be golden as its unique for each post
	//one idea is to mix the postid with the waybill id
	waybill := strings.TrimSpace(r.FormValue("waybill"))
	key := strings.TrimSpace(r.FormValue("key"))
	if key == "" || waybill == "" {
		http.Error(w, "Invalid Inputs", http.StatusBadRequest)
		logger.Error("Invalid Input", "Waybill", waybill, "Key", key)
		return
	}
	logger.Info("Waybill request received", "waybill", waybill, "key", key)
	ap := config.AccessPair{
		Key:       key,
		WaybillID: waybill,
	}
	posts, res, err := database.CheckDeliveryStatus(d.DBObj, ap)
	if err != nil {
		//http.Error(w, "Failed to check delivery status", http.StatusInternalServerError)
		logger.Error("Failed to check delivery status", "error", err)
	}
	var dd Data //couldnt place in config due to circular dependency(models and config)
	switch res {
	case 1: //waybill not found
		//http.Error(w, "Waybill not found", http.StatusNotFound)
		//logger.Info("Waybill not found")
		dd.Show = false
		dd.Response = "Waybill not found"
		w.WriteHeader(http.StatusNotFound)
	case 2: //failed to check delivery status
		//http.Error(w, "Failed to check delivery status", http.StatusInternalServerError)
		//logger.Error("Failed to check delivery status", "error", err)
		dd.Show = false
		dd.Response = "Failed to check delivery status"
		w.WriteHeader(http.StatusInternalServerError)
	case 3: //key not matching
		//http.Error(w, "Key not matching", http.StatusUnauthorized)
		//logger.Error("Key not matching", "error", err)
		dd.Show = false
		dd.Response = "Key not matching"
		w.WriteHeader(http.StatusUnauthorized)
	case 4: //match found
		if len(posts) == 1 { //multiple access pairs with same waybill id and keys(highly unlikely but still i ran into it)
			dd.Show = true
			dd.IsDelivered = posts[0].IsDelivered
			var response string
			if dd.IsDelivered {
				response = "Delivered"
			} else {
				response = "Not Delivered Yet"
			}
			dd.Response = fmt.Sprintf("Delivery Status: %s", response)
			dd.Delivery = posts[0].Delivery
			if dd.IsDelivered {
				//only show the post if its delivered
				dd.Post = posts[0]
			}
			logger.Info("Delivery Status: ", dd.IsDelivered, "Delivery: ", dd.Delivery)
			w.WriteHeader(http.StatusFound) //only give found if only one pair found
		} else { //multiples
			dd.Choices = []config.Choices{}
			for _, post := range posts {
				//if we put the post itself in choices, somone can access a post not belonging to them
				//so we only send the created at dates to the html
				//user can choose which one is theirs
				dd.Choices = append(dd.Choices, config.Choices{
					CreatedAt: post.CreatedAt,
					PostID:    post.PostID})
			}
			dd.Response = "Multiple Access Pairs Found"
			logger.Info("Multiple Pairs Found: ", dd.Choices)
			w.WriteHeader(http.StatusMultipleChoices)
		}
	case 5:
		dd.Show = false
		dd.Response = "Failed to check delivery status"
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err = courierpg.Execute(w, dd)
	if err != nil {
		logger.Error("Template loading failed", "error", err)
		http.Error(w, "Template loading failed", http.StatusInternalServerError)
	}
}

func (d *DBConfig) idHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		//this is only get requests
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	postid := r.URL.Query().Get("id")
	logger.Info("Post ID request received", "postid", postid)
	post := database.GetPost(d.DBObj, postid)
	var dd Data
	dd.Show = true
	dd.IsDelivered = post.IsDelivered
	var response string
	if dd.IsDelivered {
		response = "Delivered"
	} else {
		response = "Not Delivered Yet"
	}
	dd.Response = fmt.Sprintf("Delivery Status: %s", response)
	dd.Delivery = post.Delivery
	//if dd.IsDelivered {
	//only show the post if its delivered
	//dd.Post = post
	//}
	//if there were multi access pairs with same waybill id and keys,
	//a risk is that one guy might be able to access the post not belonging to them
	//so in such cases we only show the created at dates,delivery date and delivery status
	//this way they know when it wiil reach them without them being able to access the post info of others
	//will block the message display on courier page for such cases, users can get theri deli dates from created date
	//another approach i thouught of was asking users the POST ID if this happende and only if this happend
	//but again that a unique number and it might be tedious to save it somewehr and find it again for this
	//so this is the one i prefer, if u have same pair as anohte user, u two will be able to see each others delivery data but nothig els
	//the postid ask approach has risk of lockngi the user out if they lost it
	logger.Info("Delivery Status: ", dd.IsDelivered, "Delivery: ", dd.Delivery)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := courierpg.Execute(w, dd)
	if err != nil {
		logger.Error("Template loading failed", "error", err)
		http.Error(w, "Template loading failed", http.StatusInternalServerError)
	}
}
