package main

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

func mapHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		log.Errorf("Request with method not allowed: %s", r.Method)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Errorf("Error reading request body: %s", err)
		return
	}

	// preprocess content
	re := regexp.MustCompile(`[[:punct:]]`)
	content := re.ReplaceAllString(string(body), "")
	content = strings.ToLower(content)

	// compute word mappings
	words := strings.Fields(content)
	mappings := []map[string]int{}
	for _, word := range words {
		mapping := map[string]int{
			word: 1,
		}

		mappings = append(mappings, mapping)
	}

	// write response
	w.Header().Set("Content-Type", "application/json")
	wm_marshaled, err := json.Marshal(mappings)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Errorf("Error encoding mapping: %s", err)
		return
	}
	if _, err := w.Write(wm_marshaled); err != nil {
		log.Errorf("Error writing response: %s", err)
		return
	}

	log.Infof("Successfully mapped the words in: %.8s...", content)

}

func main() {
	http.HandleFunc("/", mapHandler)

	http.ListenAndServe(":80", nil)
}
