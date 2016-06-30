package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type BaseStat struct {
	Userid   string
	Datetime uint32
	Target   string
	Operator string
	Network  string
}

type PingResult struct {
	Seq   uint
	Ttl   uint
	Size  uint
	Delay float64
}

type PingStat struct {
	BaseStat
	Result []PingResult
}

type TrResult struct {
	Hop   uint
	Ip    string
	Delay []float64
}

type TrStat struct {
	BaseStat
	Result []TrResult
}

func fileHandler(resp http.ResponseWriter, req *http.Request) {
	if req.Method == "PUT" {
		vars := mux.Vars(req)
		filename := vars["filename"]
		f, err := os.Create(filename)
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

func pingHandler(resp http.ResponseWriter, req *http.Request) {
	var ps PingStat
	if req.Method == "POST" {
		data, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Println(err)
		}
		err = json.Unmarshal(data, &ps)
		if err != nil {
			log.Println(err)
		}
		filename := fmt.Sprintf("ping_%s-%d.log", ps.Userid, ps.Datetime)
		f, err := os.Create(filename)
		if err != nil {
			log.Println(err)
		}
		defer f.Close()
		_, err = f.Write([]byte(fmt.Sprintf("Userid: %s\nDatetime: %d\nOperator: %s\nNetwork: %s\nTarget: %s\n", ps.Userid, ps.Datetime, ps.Operator, ps.Network, ps.Target)))
		if err != nil {
			log.Println(err)
		}
		for _, r := range ps.Result {
			_, err = f.Write([]byte(fmt.Sprintf("%d %d %7.3f %d", r.Seq, r.Size, r.Delay, r.Ttl)))
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func tracertHandler(resp http.ResponseWriter, req *http.Request) {
	var trs TrStat
	if req.Method == "POST" {
		data, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Println(err)
		}
		err = json.Unmarshal(data, &trs)
		if err != nil {
			log.Println(err)
		}
		filename := fmt.Sprintf("tracert_%s-%d.log", trs.Userid, trs.Datetime)
		f, err := os.Create(filename)
		if err != nil {
			log.Println(err)
		}
		defer f.Close()
		_, err = f.Write([]byte(fmt.Sprintf("Userid: %s\nDatetime: %d\nOperator: %s\nNetwork: %s\nTarget: %s\n", trs.Userid, trs.Datetime, trs.Operator, trs.Network, trs.Target)))
		if err != nil {
			log.Println(err)
		}
		for _, r := range trs.Result {
			delays := ""
			for _, d := range r.Delay {
				delays += fmt.Sprintf("%7.3f ", d)
			}
			_, err = f.Write([]byte(fmt.Sprintf("%d %s%d", r.Hop, delays, r.Ip)))
			if err != nil {
				log.Println(err)
			}
		}
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
	r.HandleFunc("/ping", pingHandler)
	r.HandleFunc("/tracert", tracertHandler)
	log.Printf("Listening on %d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), r))
}
