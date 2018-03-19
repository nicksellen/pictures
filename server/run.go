package server

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	bleveHttp "github.com/blevesearch/bleve/http"
	"github.com/gorilla/mux"
	"github.com/julienschmidt/httprouter"
	"github.com/nicksellen/pictures/index"
)

func getStaticDir() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return filepath.Join(filepath.Dir(ex), "static")
}

// Run indexes things
func Run() error {
	index, err := index.OpenIndexReadOnly("db.bleve")

	// index, err := index.OpenIndex("db.bleve")
	if err != nil {
		return err
	}

	bindAddr := "localhost:7080"

	router := httprouter.New()

	bleveHttp.RegisterIndexName("pictures", index)
	router.Handler("POST", "/api/search", bleveHttp.NewSearchHandler("pictures"))
	router.Handler("GET", "/api/fields", bleveHttp.NewListFieldsHandler("pictures"))

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("frontend/dist")))
	mux.Handle("/api/", router)

	mux.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("/tmp/pictures-foo/thumbnails/1024x786"))))

	log.Printf("listening on %v", bindAddr)
	return http.ListenAndServe(bindAddr, mux)
}

func muxVariableLookup(req *http.Request, name string) string {
	return mux.Vars(req)[name]
}

func docIDLookup(req *http.Request) string {
	return muxVariableLookup(req, "docID")
}
