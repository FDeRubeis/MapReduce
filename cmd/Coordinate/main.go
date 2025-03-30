package main

import (
	"bytes"
	"encoding/json"
	"hash/fnv"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

type mapReturn struct {
	mappings []map[string]int
	err      error
}

type shuffleReturn struct {
	shuffles map[string][]int
	err      error
}

type redReturn struct {
	count map[string]int
	err   error
}

func partitionContent(content string, n int) []string {

	// Compute size (in lines) of each partition
	lines := strings.Split(content, "\n")
	totalLines := len(lines)
	partSize := totalLines / n
	remainder := totalLines % n

	parts := make([]string, n)
	index := 0

	for i := 0; i < n; i++ {

		// Compute size of current partition
		currentPartSize := partSize
		if i < remainder {
			currentPartSize++
		}

		// Assign lines to partition
		part := strings.Join(lines[index:index+currentPartSize], "\n")
		parts[i] = part

		index += currentPartSize
	}

	return parts
}

func mapContent(content string, http_workers_num int) ([]map[string]int, error) {

	mapTasks := partitionContent(content, http_workers_num)
	retCh := make(chan mapReturn, http_workers_num)

	for _, task := range mapTasks {

		go func() {

			// send a task to the map service
			url := "http://" + os.Getenv("MAP_SVC_NAME") + ":" + os.Getenv("MAP_SVC_PORT")
			resp, err := http.Post(url, "text/plain", strings.NewReader(task))
			if err != nil {
				retCh <- mapReturn{nil, err}
				return
			}
			defer resp.Body.Close()

			// read response
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				retCh <- mapReturn{nil, err}
				return
			}

			// get mappings from worker
			mappings := []map[string]int{}
			if err := json.Unmarshal(body, &mappings); err != nil {
				retCh <- mapReturn{nil, err}
				return
			}

			retCh <- mapReturn{mappings, nil}

		}()
	}

	// get all mappings
	mappings := []map[string]int{}
	for i := 0; i < http_workers_num; i++ {

		ret := <-retCh
		if ret.err != nil {
			return nil, ret.err
		}

		mappings = append(mappings, ret.mappings...)

	}

	return mappings, nil
}

func getShuffler(mapping map[string]int, shufflers int) int {

	// Extract the first (and only) key from the mapping
	var key string
	for k := range mapping {
		key = k
		break
	}

	// assign to shuffler
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32()) % shufflers
}

func shuffle(mappings []map[string]int) (map[string][]int, error) {

	// nslookup shuffle hosts
	ips, err := net.LookupIP(os.Getenv("SHUFFLE_SVC_NAME"))
	if err != nil {
		return nil, err
	}
	shufflers := len(ips)

	shuffleTasks := make([][]map[string]int, shufflers)
	for _, mapping := range mappings {
		shfl := getShuffler(mapping, shufflers)
		shuffleTasks[shfl] = append(shuffleTasks[shfl], mapping)
	}

	retCh := make(chan shuffleReturn, shufflers)

	for i, task := range shuffleTasks {

		go func() {

			// skip empty tasks
			if len(task) == 0 {
				retCh <- shuffleReturn{nil, nil}
				return
			}

			marshaled_task, err := json.Marshal(task)
			if err != nil {
				retCh <- shuffleReturn{nil, err}
				return
			}

			// send a task to the shuffle service
			url := "http://" + ips[i].String() + ":" + os.Getenv("SHUFFLE_SVC_PORT")
			resp, err := http.Post(url, "application/json", bytes.NewReader(marshaled_task))
			if err != nil {
				retCh <- shuffleReturn{nil, err}
				return
			}
			defer resp.Body.Close()

			// read response
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				retCh <- shuffleReturn{nil, err}
				return
			}

			// get shuffles from worker
			shuffles := map[string][]int{}
			if err := json.Unmarshal(body, &shuffles); err != nil {
				retCh <- shuffleReturn{nil, err}
				return
			}

			retCh <- shuffleReturn{shuffles, nil}

		}()
	}

	// get all shuffles
	shuffles := map[string][]int{}
	for i := 0; i < shufflers; i++ {

		ret := <-retCh
		if ret.err != nil {
			return nil, ret.err
		}

		for shuffle := range ret.shuffles {
			shuffles[shuffle] = ret.shuffles[shuffle]
		}

	}

	return shuffles, nil
}

func partitionShuffle(shuffle map[string][]int, n int) []map[string][]int {

	// split shuffle in n parts
	parts := make([]map[string][]int, n)
	for i := range parts {
		parts[i] = map[string][]int{}
	}

	idx := 0
	for word, mappings := range shuffle {

		parts[idx%n][word] = mappings
		idx++
	}

	return parts
}

func reduce(shuffle map[string][]int, http_workers_num int) (map[string]int, error) {

	reduceTasks := partitionShuffle(shuffle, http_workers_num)
	retCh := make(chan redReturn, http_workers_num)

	for _, task := range reduceTasks {
		go func() {

			marshaled_task, err := json.Marshal(task)
			if err != nil {
				retCh <- redReturn{nil, err}
				return
			}

			// send a task to the reduce service
			url := "http://" + os.Getenv("REDUCE_SVC_NAME") + ":" + os.Getenv("REDUCE_SVC_PORT")
			resp, err := http.Post(url, "application/json", bytes.NewReader(marshaled_task))
			if err != nil {
				retCh <- redReturn{nil, err}
				return
			}
			defer resp.Body.Close()

			// read response
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				retCh <- redReturn{nil, err}
				return
			}

			// get word count
			word_count := map[string]int{}
			if err := json.Unmarshal(body, &word_count); err != nil {
				retCh <- redReturn{nil, err}
				return
			}

			retCh <- redReturn{word_count, nil}

		}()
	}

	// get word counts
	word_count := map[string]int{}
	for i := 0; i < http_workers_num; i++ {
		ret := <-retCh
		if ret.err != nil {
			return nil, ret.err
		}
		for word, count := range ret.count {
			word_count[word] = count
		}
	}

	return word_count, nil
}

func coordinatorHandler(w http.ResponseWriter, r *http.Request) {

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

	http_workers_num, err := strconv.Atoi(os.Getenv("HTTP_WORKERS_NUM"))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Errorf("Invalid HTTP_WORKERS_NUM: %s", os.Getenv("HTTP_WORKERS_NUM"))
		return
	}

	// map
	content := string(body)
	mappings, err := mapContent(content, http_workers_num)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Errorf("Map request failed: %s", err)
		return
	}

	// shuffle
	shuffles, err := shuffle(mappings)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Errorf("Shuffle request failed: %s", err)
		return
	}

	// reduce
	word_count, err := reduce(shuffles, http_workers_num)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Errorf("Reduce request failed: %s", err)
		return
	}

	// write response
	w.Header().Set("Content-Type", "application/json")
	wc_marshaled, err := json.Marshal(word_count)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Errorf("Error encoding word count: %s", err)
		return
	}

	if _, err := w.Write(wc_marshaled); err != nil {
		log.Errorf("Error writing answer: %s", err)
		return
	}

	log.Infof("Successfully counted the words in: %.8s...", content)
}

func main() {
	http.HandleFunc("/", coordinatorHandler)

	http.ListenAndServe(":80", nil)
}
