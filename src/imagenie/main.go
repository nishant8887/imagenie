package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/satori/go.uuid"
	"gopkg.in/yaml.v2"
	"imagenie/models"
	"imagenie/quehelper"
	"imagenie/utils"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

type ServiceSettings struct {
	Port       uint32
	DbHost     string
	DbUser     string
	DbName     string
	DbSslMode  string
	DbPassword string
	Workers    int
}

type ImagenieListener struct {
	db        *gorm.DB
	queHelper quehelper.QueHelper
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

	self.queHelper = quehelper.QueHelper{}
	err = self.queHelper.Init(self.settings.DbHost, self.settings.DbUser, self.settings.DbName, self.settings.DbPassword, self.settings.Workers)
	if err != nil {
		return err
	}

	defer self.queHelper.Shutdown()

	r := mux.NewRouter()
	r.HandleFunc("/home", self.Home)
	r.HandleFunc("/upload", self.UploadFile).Methods("POST")

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

func (self *ImagenieListener) Home(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello world!")
}

func (self *ImagenieListener) UploadFile(w http.ResponseWriter, r *http.Request) {

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

	file_id := uuid.NewV4()
	new_file_name := fmt.Sprintf("/tmp/%s-original%s", file_id, extension)

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

	w.WriteHeader(200)
}
