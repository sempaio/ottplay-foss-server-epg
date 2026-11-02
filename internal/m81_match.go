package internal

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	xxhash32 "github.com/OneOfOne/xxhash"
)

const maxM81MatchBody = 8 << 20
const maxM81MatchChannels = 20000

type m81Channel struct {
	Key   string `json:"key"`
	TvgID string `json:"tvg_id"`
	Name  string `json:"name"`
}

type m81Request struct {
	Channels []m81Channel `json:"channels"`
}

type m81Suggestion struct {
	Key      string `json:"key"`
	SourceID string `json:"source_id"`
	XMLTVID  string `json:"xmltv_id"`
}

type m81Response struct {
	Suggestions []m81Suggestion `json:"suggestions"`
}

// M81Match is an optional, authenticated suggestion API. It never approves a mapping.
func M81Match(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.Body == nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxM81MatchBody))
	var request m81Request
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF || len(request.Channels) > maxM81MatchChannels {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	for _, channel := range request.Channels {
		if len(channel.Key) == 0 || len(channel.Key) > 200 || len(channel.TvgID) > 200 || len(channel.Name) > 200 {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
	}
	ChDbLock.RLock()
	defer ChDbLock.RUnlock()
	response := m81Response{Suggestions: make([]m81Suggestion, 0, len(request.Channels))}
	now := time.Now().Unix()
	for _, channel := range request.Channels {
		for _, provider := range PO.Epg {
			var found = provider.ById[xxhash32.ChecksumString32(channel.TvgID)]
			if found != nil && found.Id != channel.TvgID {
				found = nil
			}
			if found == nil && channel.Name != "" {
				name := strings.ToLower(channel.Name)
				candidate := provider.ByName[xxhash32.ChecksumString32(name)]
				if candidate != nil {
					for _, known := range candidate.Names {
						if known == name {
							found = candidate
							break
						}
					}
				}
			}
			if found != nil && found.Id != "" && found.LastEpg > now {
				response.Suggestions = append(response.Suggestions, m81Suggestion{
					Key: channel.Key, SourceID: provider.Prov.Id, XMLTVID: found.Id,
				})
				break
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(response)
}
