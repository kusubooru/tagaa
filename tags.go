package main

import (
	"encoding/json"
	"net/http"

	"github.com/kusubooru/tagaa/autocomplete"
)

func tagsHandler(w http.ResponseWriter, r *http.Request) {
	q := r.FormValue("q")
	tags, err := autocomplete.GetTags(q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(tags); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
