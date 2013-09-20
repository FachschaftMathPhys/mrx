package main

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"fmt"
	"git.merovius.de/MrX/log"
	"html/template"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"
)

var index = template.Must(template.ParseFiles("index.html"))


func Main(w http.ResponseWriter, req *http.Request) {
	index.Execute(w, nil)
}

func Static(w http.ResponseWriter, req *http.Request) {
	header := w.Header()
	if req.Method != "GET" {
		log.Warn("Invalid method for static request")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	filepath := path.Join("static", path.Base(req.URL.Path))
	f, err := os.Open(filepath)
	if err != nil {
		log.Warn("Could not open file: %s", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	header.Set("content-type", mime.TypeByExtension(path.Ext(filepath)))
	io.Copy(w, f)
}

func SetMrX(w http.ResponseWriter, req *http.Request) {
	var reg Registration
	rec := json.NewDecoder(req.Body)
	err := rec.Decode(&reg)
	if err != nil {
		log.Error("Could not decode: ", err)
	}
	NumMrX = reg.Number
	if strings.HasPrefix(NumMrX, "+") {
		NumMrX = NumMrX[1:]
	} else if strings.HasPrefix(NumMrX, "0") {
		NumMrX = "49" + NumMrX[1:]
	}
	log.Printf("Set Number of MrX to %s", NumMrX)
}

func main() {
	go InitSMS()
	http.HandleFunc("/", Main)
	http.HandleFunc("/static/", Static)
	http.HandleFunc("/setmrx", SetMrX)
	http.Handle("/log", websocket.Handler(log.LogServer))
	http.Handle("/sms", websocket.Handler(SmsServer))
	http.Handle("/reg", websocket.Handler(RegServer))
	err := http.ListenAndServe(fmt.Sprintf(":%d", 8000), nil)
	if err != nil {
		panic("ListenANdServe: " + err.Error())
	}
}
