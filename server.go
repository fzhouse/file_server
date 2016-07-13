package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
	"os"
)

func fileHandler(resp http.ResponseWriter, req *http.Request) {
	if req.Method == "PUT" {
		vars := mux.Vars(req)
		filename := vars["filename"]
		f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			log.Println(err)
		}
		defer f.Close()
		_, err = io.Copy(f, req.Body)
		if err != nil {
			log.Println(err)
		}
		log.Printf("file %s uploaded", filename)
	}
}

var port int

func init() {
	flag.IntVar(&port, "p", 8000, "Listen Port")
	flag.Parse()
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/file/{filename}", fileHandler)
	log.Printf("Listening on %d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), r))
}
