package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"gopkg.in/yaml.v2"
	"imagenie/utils"
	"io/ioutil"
	"net/http"
)

type ServiceSettings struct {
	Port       uint32
	DbHost     string
	DbUser     string
	DbName     string
	DbSslMode  string
	DbPassword string
}

type ImagenieListener struct {
	db       *gorm.DB
	settings ServiceSettings
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

	r := mux.NewRouter()
	r.HandleFunc("/user/", self.Home)
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
