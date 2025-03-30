package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func reduceHandler(w http.ResponseWriter, r *http.Request) {

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

	shuffle := map[string][]int{}
	if err = json.Unmarshal(body, &shuffle); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Errorf("Error decoding JSON: %s", err)
		return
	}

	// compute word count
	wc := map[string]int{}
	for word, mappings := range shuffle {
		wc[word] = 0
		for _, mapping := range mappings {
			wc[word] += mapping
		}
	}

	// write response
	w.Header().Set("Content-Type", "application/json")
	wc_marshaled, err := json.Marshal(wc)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Errorf("Error encoding word count: %s", err)
		return
	}
	if _, err = w.Write(wc_marshaled); err != nil {
		log.Errorf("Error writing response: %s", err)
	}

	log.Infof("Successfully reduced the occurrences in %.16s...", fmt.Sprintf("%v", shuffle))

}
func main() {
	http.HandleFunc("/", reduceHandler)

	http.ListenAndServe(":80", nil)
}
