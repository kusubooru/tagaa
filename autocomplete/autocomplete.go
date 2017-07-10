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
	danbooruBoard           = "danbooru"
	teianBoard              = "kusubooru"
	minAllowedQueryLength   = 3
)

type Category int

const (
	Unknown Category = iota
	Normal
	Artist
	Character
	Series
	Tk
)

func (c Category) String() string {
	switch c {
	case Normal:
		return "normal"
	case Artist:
		return "artist"
	case Character:
		return "character"
	case Series:
		return "series"
	case Tk:
		return "tk"
	default:
		return "unknown"
	}
}

func (c *Category) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	switch strings.ToLower(s) {
	case "normal":
		*c = Normal
	case "artist":
		*c = Artist
	case "character":
		*c = Character
	case "series":
		*c = Series
	case "tk":
		*c = Tk
	default:
		*c = Unknown
	}
	return nil
}

func (c Category) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String())
}

type danbooruCategory int

const (
	normal    danbooruCategory = 0
	artist                     = 1
	series                     = 3
	character                  = 4
)

func (c danbooruCategory) Category() Category {
	switch c {
	case normal:
		return Normal
	case artist:
		return Artist
	case series:
		return Series
	case character:
		return Character
	default:
		return Normal
	}
}

type Tag struct {
	Board    string   `json:"board"`
	Name     string   `json:"name"`
	Old      string   `json:"old"`
	Count    int      `json:"count"`
	Category Category `json:"category"`
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
		if t.Board == teianBoard {
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
		t.Board = teianBoard
		switch {
		case strings.HasPrefix(t.Name, "artist:"):
			t.Category = Artist
		case strings.HasPrefix(t.Name, "character:"):
			t.Category = Character
		case strings.HasPrefix(t.Name, "series:"):
			t.Category = Series
		case strings.HasPrefix(t.Name, "tk:"):
			t.Category = Tk
		default:
			t.Category = Normal
		}
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
			Board:    "danbooru",
			Category: danbooruCategory(ac.Category).Category(),
		}
		tags = append(tags, t)
	}
	if len(tags) > 5 {
		return tags[:5], nil
	}
	return tags, nil
}
