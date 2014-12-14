package http

import (
	"encoding/json"
	"net/http"
)

func respondJson(w http.ResponseWriter, obj interface{}) {
	w.Header().Set("Content-Type", "application/json; encoding=utf8")

	e := json.NewEncoder(w)
	if err := e.Encode(obj); err != nil {
		panic(err)
	}
}

func readJson(r *http.Request, obj interface{}) {
	d := json.NewDecoder(r.Body)
	if err := d.Decode(obj); err != nil {
		panic(err)
	}
}
