package autocomplete

import (
	"encoding/json"
	"net/http"
	"strings"
)

const (
	danbooruTagsURL       = "https://danbooru.donmai.us/tags.json?search[name_matches]="
	danbooruTagAliasesURL = "https://danbooru.donmai.us/tag_aliases.json?search[name_matches]="
)

func GetTags(query string) ([]*DanbooruTagAlias, error) {
	query = strings.TrimSpace(query)
	if len(query) < 2 {
		return []*DanbooruTagAlias{}, nil
	}
	if query == "" {
		return []*DanbooruTagAlias{}, nil
	}
	resp, err := http.Get(danbooruTagAliasesURL + "*" + query + "*")
	if err != nil {
		return nil, err
	}

	var tags []*DanbooruTagAlias
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, err
	}
	return tags, nil
}

type DanbooruTagAlias struct {
	ID             int    `json:"id"`
	AntecedentName string `json:"antecedent_name"`
	Reason         string `json:"reason"`
	CreatorID      int    `json:"creator_id"`
	ConsequentName string `json:"consequent_name"`
	Status         string `json:"status"`
	ForumTopicID   int    `json:"forum_topic_id"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
	PostCount      int    `json:"post_count"`
}
