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
	"imagenie/utils"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
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
	TmpPath    string
}

type ImagenieListener struct {
	db        *gorm.DB
	queHelper QueHelper
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

	deleteFile := func() {
		log.Info("Deleting file: ", new_file_name)
		err := os.Remove(new_file_name)
		if err != nil {
			log.Error("Error in removing file: ", new_file_name, err)
		}
	}

	file_id_str := fmt.Sprintf("%s", file_id)
	image := models.Image{FileId: file_id_str, Extension: extension}
	err = self.db.Create(&image).Error
	if err != nil {
		defer deleteFile()
		http.Error(w, "Error in saving file. (cannot create db entry)", http.StatusInternalServerError)
		return
	}

	data := ResizeImageArgs{ImageId: file_id_str, Path: new_file_name}
	err = self.queHelper.DoResize(data)
	if err != nil {
		defer deleteFile()
		http.Error(w, "Error in saving file. (cannot launch resize task)", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(200)
}
