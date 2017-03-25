package utils

import (
	log "github.com/Sirupsen/logrus"
	"net/http"
	"os"
	"path"
)

type noDirFileHandler struct {
	basePath   string
	fileserver http.Handler
}

func (self *noDirFileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	upath := path.Clean(self.basePath + "/" + r.URL.Path)
	if f, err := os.Stat(upath); err == nil && !f.IsDir() {
		log.Info("Serving file: ", upath)
		self.fileserver.ServeHTTP(w, r)
		return
	}
	http.NotFound(w, r)
}

func NoDirFileServer(basePath string) http.Handler {
	return &noDirFileHandler{basePath, http.FileServer(http.Dir(basePath))}
}

func DeleteFile(file_name string) error {
	if _, err := os.Stat(file_name); err == nil {
		log.Info("Deleting file: ", file_name)
		err := os.Remove(file_name)
		if err != nil {
			log.Error("Error in deleting file: ", file_name, err)
			return err
		}
	}
	return nil
}
