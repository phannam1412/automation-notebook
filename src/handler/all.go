package handler

import (
	"io/ioutil"
	"net/http"
	"strings"
)

func All(w http.ResponseWriter, r *http.Request) {
	if r.RequestURI != "/" {
		w.WriteHeader(404)
		return
	}
	b, err := ioutil.ReadFile("index.html")
	if err != nil {
		w.WriteHeader(500)
		_,_ = w.Write([]byte(err.Error()))
		return
	}
	html := string(b)
	b, err = ioutil.ReadFile("history.txt")
	if err == nil {
		j, err := getHistory()
		if err == nil {
			html = strings.ReplaceAll(html, "{{history}}", string(j))
		} else {
			html = strings.ReplaceAll(html, "{{history}}", "[]")
		}
	}
	b, err = ioutil.ReadFile("public/main.js")
	if err != nil {
		html = strings.ReplaceAll(html, "{{react}}", "")
	} else {
		html = strings.ReplaceAll(html, "{{react}}", string(b))
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	_,_ = w.Write([]byte(html))
}
