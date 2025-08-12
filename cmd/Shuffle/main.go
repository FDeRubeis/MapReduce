package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func shuffleHandler(w http.ResponseWriter, r *http.Request) {

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

	mappings := []map[string]int{}
	if err = json.Unmarshal(body, &mappings); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Errorf("Error decoding JSON: %s", err)
		return
	}

	// compute shuffles
	shuffles := map[string][]int{}
	for _, mapping := range mappings {

		// get first (and only) key from the mapping
		var key string
		for k := range mapping {
			key = k
			break
		}

		if _, ok := shuffles[key]; !ok {
			shuffles[key] = []int{}
		}
		shuffles[key] = append(shuffles[key], mapping[key])
	}

	// write response
	w.Header().Set("Content-Type", "application/json")
	shfl_marshaled, err := json.Marshal(shuffles)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Errorf("Error encoding shuffles: %s", err)
		return
	}
	if _, err := w.Write(shfl_marshaled); err != nil {
		log.Errorf("Error writing response: %s", err)
		return
	}

	log.Infof("Successfully shuffled the mappings in %.16s...", fmt.Sprintf("%v", mappings))

}

func main() {
	http.HandleFunc("/", shuffleHandler)

	http.ListenAndServe(":80", nil)
}
