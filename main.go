package main

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
	"github.com/unrolled/render"
)


// config configures directory to be looked upon.
type config struct {
	LookupDir string `json:"directory"`
}

type HomeDisplay struct {
	User, Title, ServingDir string
	Files []string
}

func main() {
	// get a router, with a default home handler
	r := mux.NewRouter()

	// read the configuration from the file in the directory
	conf := readConfig()

	// get the current user. If there is a problem, panic. TODO: refactor.
	currentUser, err := user.Current()
	if err != nil {
		panic(err)
	}

	// the dir on which we are serving
	servingDir := path.Join(currentUser.HomeDir, conf.LookupDir)


	// find a file by name, on the /streamr/{id} URL
	f := r.PathPrefix("/streamr").Subrouter()

	// inject to the stream handler function through a closure the configuration and the current user.
	f.HandleFunc("/{id}", streamHandler(conf, currentUser)).Name("file")

	// construct the array of files which we can serve from.
	files := make([]string, 0)
	err = filepath.Walk(servingDir, func(path string, info os.FileInfo, err error) error {
		if err == nil {
			if servingDir != path {
				url, err := r.Get("file").URL("id", filepath.Base(path))
				if err != nil {
					return err
				}
				files = append(files, url.String())
			}
		}
		return nil
	})

	if err != nil {
		fmt.Println(err)
	}

	r.HandleFunc("/", homeHandler(render.New(), HomeDisplay{
		Title: "Streamr",
		User: currentUser.Name,
		ServingDir: servingDir,
		Files: files,
	}))


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

/*
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
*/

func homeHandler(renderer *render.Render, data HomeDisplay) http.HandlerFunc {
	return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		renderer.HTML(w, http.StatusOK, "home-template", data)
	})
}
