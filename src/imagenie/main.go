package main

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/satori/go.uuid"
	"gopkg.in/yaml.v2"
	"imagenie/models"
	"imagenie/utils"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"
)

const (
	COOKIE_EXPIRY_IN_MINUTES = 2
	PAGE_SIZE                = 6
)

type ServiceSettings struct {
	Port       uint32
	DbHost     string
	DbUser     string
	DbName     string
	DbSslMode  string
	DbPassword string
	Workers    int
	TmpPath    string
	S3BaseUrl  string
	AwsConfig  AwsConfig
}

type ImagenieListener struct {
	db        *gorm.DB
	queHelper QueHelper
	awsHelper AwsHelper
	settings  ServiceSettings
}

func (self *ImagenieListener) Start() error {
	data, err := ioutil.ReadFile("config/settings.yml")
	if err != nil {
		return err
	}

	m := ServiceSettings{}
	err = yaml.Unmarshal(data, &m)
	if err != nil {
		return err
	}

	self.settings = m

	self.db, err = gorm.Open("postgres", fmt.Sprintf("host=%s user=%s dbname=%s sslmode=%s password=%s",
		self.settings.DbHost,
		self.settings.DbUser,
		self.settings.DbName,
		self.settings.DbSslMode,
		self.settings.DbPassword))

	if err != nil {
		return err
	}

	defer self.db.Close()

	models.MigrateAll(self.db)

	self.queHelper = QueHelper{}
	err = self.queHelper.Init(self)
	if err != nil {
		return err
	}

	defer self.queHelper.Shutdown()

	self.awsHelper = AwsHelper{}
	err = self.awsHelper.Init(self.settings.AwsConfig)
	if err != nil {
		return err
	}

	r := mux.NewRouter()
	r.HandleFunc("/", self.UserHome)
	r.HandleFunc("/upload", self.UploadFile).Methods("POST")

	r.HandleFunc("/user/create", self.UserCreate).Methods("POST")
	r.HandleFunc("/user/login", self.UserLogin).Methods("POST")
	r.HandleFunc("/user/logout", self.UserLogout).Methods("POST")

	r.HandleFunc("/images/{id:[0-9]+}", self.GetImagesForPage).Methods("GET")

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", utils.NoDirFileServer("static")))

	http.Handle("/", r)
	http.ListenAndServe(fmt.Sprintf(":%d", self.settings.Port), nil)
	return nil
}

func main() {
	server := ImagenieListener{}
	err := server.Start()
	if err != nil {
		log.Error("Error occured: ", err)
	}
}

func (self *ImagenieListener) UserHome(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "static/home.html")
}

func (self *ImagenieListener) SetUserCookie(w http.ResponseWriter, user models.User, expiry int) {
	expiration := time.Now().Add(time.Duration(expiry) * time.Minute)
	cookie := http.Cookie{Name: "user", Value: user.UserName, Path: "/", Expires: expiration}
	http.SetCookie(w, &cookie)
}

func (self *ImagenieListener) UserCreate(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	firstname := r.FormValue("firstname")
	lastname := r.FormValue("lastname")
	email := r.FormValue("email")
	password := r.FormValue("password")

	if username == "" || firstname == "" || lastname == "" || email == "" || password == "" {
		http.Error(w, "{}", http.StatusBadRequest)
		return
	}

	user := models.User{
		UserName:  username,
		FirstName: firstname,
		LastName:  lastname,
		Email:     email,
		Password:  password,
	}

	err := self.db.Create(&user).Error
	if err != nil {
		http.Error(w, "{}", http.StatusInternalServerError)
		return
	}

	self.SetUserCookie(w, user, COOKIE_EXPIRY_IN_MINUTES)
	w.WriteHeader(http.StatusOK)
}

func (self *ImagenieListener) UserLogin(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		http.Error(w, "{}", http.StatusBadRequest)
		return
	}

	var user models.User
	err := self.db.Table("users").Where("user_name = ?", username).First(&user).Error
	if err != nil {
		http.Error(w, "{}", http.StatusBadRequest)
		return
	}

	if user.Password != password {
		http.Error(w, "{}", http.StatusBadRequest)
		return
	}

	self.SetUserCookie(w, user, COOKIE_EXPIRY_IN_MINUTES)
	w.WriteHeader(http.StatusOK)
}

func (self *ImagenieListener) UserLogout(w http.ResponseWriter, r *http.Request) {
	user := models.User{}
	self.SetUserCookie(w, user, -1)
	w.WriteHeader(http.StatusOK)
}

func (self *ImagenieListener) UploadFile(w http.ResponseWriter, r *http.Request) {
	// Only authenticated request hence not seperating it into different function for now
	cookie, err := r.Cookie("user")
	if err != nil {
		http.Error(w, "Error no cookie found", http.StatusUnauthorized)
		return
	}

	var user models.User
	err = self.db.Table("users").Where("user_name = ?", cookie.Value).First(&user).Error
	if err != nil {
		http.Error(w, "Error invalid user", http.StatusUnauthorized)
		return
	}

	image_description := r.FormValue("image_description")
	file, header, err := r.FormFile("image_file")

	if err != nil {
		http.Error(w, "Error in uploading file", http.StatusBadRequest)
		return
	}

	defer file.Close()

	extension := filepath.Ext(header.Filename)
	if extension == "" {
		http.Error(w, "Only image files can be uploaded", http.StatusBadRequest)
		return
	}

	// More check on extension must be added

	file_id := uuid.NewV4()
	new_file_name := path.Clean(fmt.Sprintf("%s/%s-original%s", self.settings.TmpPath, file_id, extension))

	out, err := os.Create(new_file_name)
	if err != nil {
		http.Error(w, "Unable to create the file for writing.", http.StatusInternalServerError)
		return
	}

	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		http.Error(w, "Error in saving file.", http.StatusInternalServerError)
		return
	}

	// Code to make entry in the image table and lauch it's processing task

	file_id_str := fmt.Sprintf("%s", file_id)
	image := models.Image{FileId: file_id_str, Extension: extension, Description: image_description}
	err = self.db.Create(&image).Error
	if err != nil {
		defer utils.DeleteFile(new_file_name)
		http.Error(w, "Error in saving file. (cannot create db entry)", http.StatusInternalServerError)
		return
	}

	data := ImageArgs{ImageId: file_id_str, Extension: extension, Path: new_file_name}
	err = self.queHelper.DoProcess(data)
	if err != nil {
		defer utils.DeleteFile(new_file_name)
		http.Error(w, "Error in saving file. (cannot launch resize task)", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

type ImagesResponse struct {
	Page   uint32   `json:"page"`
	Images []RImage `json:"images"`
}

type RImage struct {
	Id          string `json:"id"`
	Description string `json:"description"`
	Original    string `json:"original"`
	Resized     string `json:"resized"`
}

func (self *ImagenieListener) GetImagesForPage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	page := vars["id"]

	page_no, err := strconv.ParseUint(page, 10, 32)
	if err != nil {
		http.Error(w, "{}", http.StatusBadRequest)
		return
	}

	if page_no <= 0 {
		http.Error(w, "{}", http.StatusBadRequest)
		return
	}

	var images []models.Image
	offset := (page_no - 1) * PAGE_SIZE
	err = self.db.Debug().Table("images").Offset(offset).Limit(PAGE_SIZE).Where("done = true").Find(&images).Error
	if err != nil {
		http.Error(w, "{}", http.StatusInternalServerError)
		return
	}

	r_images := make([]RImage, 0)
	for _, image := range images {
		original_file_name := path.Clean(fmt.Sprintf("/original/%s-original%s", image.FileId, image.Extension))
		resized_file_name := path.Clean(fmt.Sprintf("/resized/%s-resized%s", image.FileId, image.Extension))
		r_image := RImage{
			Id:          image.FileId,
			Description: image.Description,
			Original:    self.settings.S3BaseUrl + original_file_name,
			Resized:     self.settings.S3BaseUrl + resized_file_name,
		}
		r_images = append(r_images, r_image)
	}

	images_response := ImagesResponse{Page: uint32(page_no), Images: r_images}
	result, err := json.Marshal(images_response)
	if err != nil {
		http.Error(w, "{}", http.StatusInternalServerError)
		return
	}

	w.Write(result)
}
