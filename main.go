package main

/*
  TODO: improve
  Make some more routing so we can see all the files and some pretty thumbnails.

  Basically, a route that makes an "ls" on the directory and returns some links to those files.
  These links can be constructed to point at the web application itself.
  For instance:


   - foo.png[/streamr/foo.png]
   - test.txt[/streamr/test.txt]

  Make some html template for this.

  TODO: Make those templates with go-bindata so we don't have to manage this on deploy.
  How far can we push this web application?
*/

import (
	"fmt"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"net/http"
	"path"
	"os"
	"os/user"
	"io"
	"encoding/json"
	"path/filepath"
)


// config configures directory to be looked upon.
type config struct {
	LookupDir string `json:"directory"`
}

func main() {
	// get a router, with a default home handler
	r := mux.NewRouter()
	r.HandleFunc("/", homeHandler)

	// read the configuration from the file in the directory
	conf := readConfig()

	// get the current user. If there is a problem, panic. TODO: refactor.
	currentUser, err := user.Current()
	if err != nil {
		panic(err)
	}

	// find a file by name, on the /streamr/{id} URL
	f := r.PathPrefix("/streamr").Subrouter()

	// inject to the stream handler function through a closure the configuration and the current user.
	f.HandleFunc("/{id}", streamHandler(conf, currentUser))
	f.HandleFunc("/", showFilesHandler(path.Join(currentUser.HomeDir, conf.LookupDir)))

	// start negroni
	n := negroni.Classic()
	n.UseHandler(r)
	n.Run(":3000")
}

// returns a default configuration if the file is fucked up
func readConfig() config {
	var conf config
	defaultConfig := config{LookupDir: "streamr"}
	j, err := os.Open("streamr.json")
	if err != nil {
		return defaultConfig
	}
	dec := json.NewDecoder(j)
	if err = dec.Decode(&conf); err != nil {
		return defaultConfig
	}
	return conf
}

// returns a http.HandlerFunc that handles the current user and configuration.
func streamHandler(conf config, current *user.User) http.HandlerFunc {
	return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		// extract the file name and look it up, on HomeDir/LookupDir/FileName.
		vars := mux.Vars(r)
		filePath := path.Join(current.HomeDir, conf.LookupDir, vars["id"])
		file, err := os.Open(filePath)
		if err != nil {
			// the file most likely doesn't exists.
			// TODO: handle better this kind of error.
			http.Error(w, http.StatusText(404), 404)
			fmt.Fprintf(w, err.Error())
			return
		}
		// copy from file to http.ResponseWriter (efficient)
		w.Header().Set("Content-Disposition", "attachment")
		io.Copy(w, file)
	})
}

func showFilesHandler(fullpath string) http.HandlerFunc {
	return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		err := filepath.Walk(fullpath, func(path string, info os.FileInfo, err error) error {
			fmt.Fprintf(w, "%s\n", path)
			return nil
		})
		if err != nil {
			fmt.Printf("[ERROR] %v", err)
		}
	})
}

// print something nice.
// TODO: templates!
func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello, home!")
}
