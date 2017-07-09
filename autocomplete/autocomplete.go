package autocomplete

import (
	"encoding/json"
	"net/http"
	"strings"
)

const (
	danbooruAutocompleteURL = "https://danbooru.donmai.us/tags/autocomplete.json?search[name_matches]="
	teianAutocompleteURL    = "https://kusubooru.com/suggest/autocomplete?q="
	minAllowedQueryLength   = 3
)

type Tag struct {
	Category string `json:"category"`
	Name     string `json:"name"`
	Old      string `json:"old"`
	Count    int    `json:"count"`
}

func GetTags(q string) ([]*Tag, error) {
	q = strings.TrimSpace(q)
	if len(q) < minAllowedQueryLength {
		return []*Tag{}, nil
	}
	teianTags, err := getTeianAutocomplete(q)
	if err != nil {
		return nil, err
	}
	danbooruTags, err := getDanbooruAutocomplete(q)
	if err != nil {
		return nil, err
	}
	tags := make([]*Tag, 0, len(teianTags)+len(danbooruTags))
	tags = append(tags, teianTags...)
	tags = append(tags, danbooruTags...)
	return tags, nil
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
		t.Category = "kusubooru"
	}
	return tags, nil
}

func getDanbooruAutocomplete(query string) ([]*Tag, error) {
	// Get danbooru autocomplete tags.
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
