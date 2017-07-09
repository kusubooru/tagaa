package autocomplete

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

const (
	danbooruAutocompleteURL = "https://danbooru.donmai.us/tags/autocomplete.json?search[name_matches]="
	teianAutocompleteURL    = "https://kusubooru.com/suggest/autocomplete?q="
	danbooruTagCategory     = "danbooru"
	teianTagCategory        = "kusubooru"
	minAllowedQueryLength   = 3
)

type Tag struct {
	Category string `json:"category"`
	Name     string `json:"name"`
	Old      string `json:"old"`
	Count    int    `json:"count"`
}

type autocompleteFn func(query string) ([]*Tag, error)

func GetTags(q string) ([]*Tag, error) {
	q = strings.TrimSpace(q)
	if len(q) < minAllowedQueryLength {
		return []*Tag{}, nil
	}

	errch := make(chan error)
	defer close(errch)
	tagsch := make(chan []*Tag)
	defer close(tagsch)
	autocompletes := []autocompleteFn{getTeianAutocomplete, getDanbooruAutocomplete}
	for _, aufn := range autocompletes {
		go func(fn autocompleteFn, query string) {
			tags, err := fn(query)
			if err != nil {
				errch <- err
				return
			}
			tagsch <- tags
		}(aufn, q)
	}
	allTags := make([]*Tag, 0)
	for range autocompletes {
		select {
		case err := <-errch:
			log.Println("autocomplete failed:", err)
		case tags := <-tagsch:
			allTags = append(allTags, tags...)
		}
	}
	// Results come already sorted by count desc. Now we need to separate them
	// and make sure the teianTagCategory category appears first.
	teianTags := make([]*Tag, 0)
	danbooruTags := make([]*Tag, 0)
	for _, t := range allTags {
		if t.Category == teianTagCategory {
			teianTags = append(teianTags, t)
		} else {
			danbooruTags = append(danbooruTags, t)
		}
	}
	allTags = make([]*Tag, 0, len(teianTags)+len(danbooruTags))
	allTags = append(allTags, teianTags...)
	allTags = append(allTags, danbooruTags...)
	return allTags, nil
}

func getTeianAutocomplete(q string) ([]*Tag, error) {
	resp, err := http.Get(teianAutocompleteURL + q)
	if err != nil {
		return nil, err
	}

	var tags []*Tag
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, err
	}
	for _, t := range tags {
		t.Category = teianTagCategory
	}
	return tags, nil
}

func getDanbooruAutocomplete(query string) ([]*Tag, error) {
	resp, err := http.Get(danbooruAutocompleteURL + "*" + query + "*")
	if err != nil {
		return nil, err
	}

	// Decode result JSON.
	type danbooruAutocomplete struct {
		Name           string `json:"name"`
		PostCount      int    `json:"post_count"`
		Category       int    `json:"category"`
		AntecedentName string `json:"antecedent_name"`
	}
	var acTags []*danbooruAutocomplete
	if err := json.NewDecoder(resp.Body).Decode(&acTags); err != nil {
		return nil, err
	}

	// Convert databooruAutocomplete to Tag.
	tags := make([]*Tag, 0, len(acTags))
	for _, ac := range acTags {
		//fmt.Printf("%#v\n", ac)
		t := &Tag{
			Name:     ac.Name,
			Count:    ac.PostCount,
			Old:      ac.AntecedentName,
			Category: "danbooru",
		}
		tags = append(tags, t)
	}
	if len(tags) > 5 {
		return tags[:5], nil
	}
	return tags, nil
}
