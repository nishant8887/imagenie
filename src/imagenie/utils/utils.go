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
