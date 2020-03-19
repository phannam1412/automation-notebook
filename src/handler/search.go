package handler

import (
	"common"
	"encoding/json"
	"net/http"
)

func Search(fuzzySearch *common.FuzzySearch) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		input := query.Get("query")
		output := fuzzySearch.Find(input, 20)
		res := NewSearchResultFromStringArray(output)
		j, err := json.Marshal(res)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
			return
		}
		w.WriteHeader(200)
		w.Write(j)
	}
}
