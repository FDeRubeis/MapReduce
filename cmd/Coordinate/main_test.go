package main

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/foxcpp/go-mockdns"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var mapServer *httptest.Server
var shuffleServer *httptest.Server
var reduceServer *httptest.Server
var shuffleServerARecord = "shuffle."

func Test_partitionContent(t *testing.T) {
	type args struct {
		content string
		n       int
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "test partition content",
			args: args{
				content: "lorem ipsum\ndolor sit\namet consectetur",
				n:       3,
			},
			want: []string{
				"lorem ipsum",
				"dolor sit",
				"amet consectetur",
			},
		},
		{
			name: "test uneven number of lines",
			args: args{
				content: "lorem ipsum\ndolor sit\namet consectetur\nadipiscing elit",
				n:       3,
			},
			want: []string{
				"lorem ipsum\ndolor sit",
				"amet consectetur",
				"adipiscing elit",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := partitionContent(tt.args.content, tt.args.n); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("partitionContent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mapContent(t *testing.T) {

	server_address := mapServer.Listener.Addr().(*net.TCPAddr)
	t.Setenv("MAP_SVC_NAME", server_address.IP.String())
	t.Setenv("MAP_SVC_PORT", strconv.Itoa(server_address.Port))

	type args struct {
		content          string
		http_workers_num int
	}
	tests := []struct {
		name    string
		args    args
		want    []map[string]int
		wantErr string
	}{
		{
			name: "test map content",
			args: args{
				content:          "lorem lorem\nlorem ipsum\nipsum sit",
				http_workers_num: 3,
			},
			want: []map[string]int{
				{"lorem": 1},
				{"lorem": 1},
				{"lorem": 1},
				{"ipsum": 1},
				{"ipsum": 1},
				{"sit": 1},
			},
		},
		{
			name: "test gibberish response",
			args: args{
				content:          "send me gibberish",
				http_workers_num: 3,
			},
			wantErr: "invalid character 'b' looking for beginning of value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mapContent(tt.args.content, tt.args.http_workers_num)
			if err != nil {
				assert.EqualErrorf(t, err, tt.wantErr, "map_content() =  %q, want %q", err.Error(), tt.wantErr)
			}

			if !slicesDeepEqual(got, tt.want) {
				t.Errorf("map_content() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getShuffler(t *testing.T) {
	type args struct {
		mapping   map[string]int
		shufflers int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "test get shuffler",
			args: args{
				mapping: map[string]int{
					"lorem": 1,
				},
				shufflers: 3,
			},
			want: 1,
		},
		{
			name: "test get shuffler 2",
			args: args{
				mapping: map[string]int{
					"dolor": 1,
				},
				shufflers: 6,
			},
			want: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getShuffler(tt.args.mapping, tt.args.shufflers); got != tt.want {
				t.Errorf("getShuffler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_shuffle(t *testing.T) {

	server_address := shuffleServer.Listener.Addr().(*net.TCPAddr)
	t.Setenv("SHUFFLE_SVC_NAME", shuffleServerARecord)
	t.Setenv("SHUFFLE_SVC_PORT", strconv.Itoa(server_address.Port))

	type args struct {
		mappings []map[string]int
	}
	tests := []struct {
		name    string
		args    args
		want    map[string][]int
		wantErr string
	}{
		{
			name: "test shuffle",
			args: args{
				mappings: []map[string]int{
					{"lorem": 1},
					{"lorem": 1},
					{"lorem": 1},
					{"ipsum": 1},
					{"ipsum": 1},
					{"sit": 1},
				},
			},
			want: map[string][]int{
				"lorem": {1, 1, 1},
				"ipsum": {1, 1},
				"sit":   {1},
			},
		},
		{
			name: "test gibberish response",
			args: args{
				mappings: []map[string]int{
					{"gibberish": 1},
				},
			},
			wantErr: "invalid character 'b' looking for beginning of value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := shuffle(tt.args.mappings)
			if err != nil {
				assert.EqualErrorf(t, err, tt.wantErr, "shuffle() =  %q, want %q", err.Error(), tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("shuffle() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_partitionShuffle(t *testing.T) {
	type args struct {
		shuffle map[string][]int
		n       int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test partition shuffle",
			args: args{
				shuffle: map[string][]int{
					"lorem":       {1, 1, 1},
					"ipsum":       {1, 1},
					"dolor":       {1, 1},
					"sit":         {1},
					"amet":        {1},
					"consectetur": {1, 1, 1},
				},
				n: 3,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := partitionShuffle(tt.args.shuffle, tt.args.n); len(got) != tt.args.n {
				t.Errorf("partitionShuffle() = unexpected result: %v", got)
			}
		})
	}
}

func Test_reduce(t *testing.T) {

	server_address := reduceServer.Listener.Addr().(*net.TCPAddr)
	t.Setenv("REDUCE_SVC_NAME", server_address.IP.String())
	t.Setenv("REDUCE_SVC_PORT", strconv.Itoa(server_address.Port))

	type args struct {
		shuffle          map[string][]int
		http_workers_num int
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]int
		wantErr string
	}{
		{
			name: "test reduce",
			args: args{
				shuffle: map[string][]int{
					"lorem": {1, 1, 1},
					"ipsum": {1, 1},
					"sit":   {1},
				},
				http_workers_num: 3,
			},
			want: map[string]int{
				"lorem": 3,
				"ipsum": 2,
				"sit":   1,
			},
		},
		{
			name: "test gibberish response",
			args: args{
				shuffle: map[string][]int{
					"gibberish": {1, 1, 1},
				},
				http_workers_num: 3,
			},
			wantErr: "invalid character 'b' looking for beginning of value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := reduce(tt.args.shuffle, tt.args.http_workers_num)
			if err != nil {
				assert.EqualErrorf(t, err, tt.wantErr, "reduce() =  %q, want %q", err.Error(), tt.wantErr)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("reduce() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_coordinatorHandler(t *testing.T) {

	mapServerAddress := mapServer.Listener.Addr().(*net.TCPAddr)
	t.Setenv("MAP_SVC_NAME", mapServerAddress.IP.String())
	t.Setenv("MAP_SVC_PORT", strconv.Itoa(mapServerAddress.Port))

	server_address := shuffleServer.Listener.Addr().(*net.TCPAddr)
	t.Setenv("SHUFFLE_SVC_NAME", shuffleServerARecord)
	t.Setenv("SHUFFLE_SVC_PORT", strconv.Itoa(server_address.Port))

	redServerAddress := reduceServer.Listener.Addr().(*net.TCPAddr)
	t.Setenv("REDUCE_SVC_NAME", redServerAddress.IP.String())
	t.Setenv("REDUCE_SVC_PORT", strconv.Itoa(redServerAddress.Port))

	type args struct {
		r *http.Request
		w *httptest.ResponseRecorder
	}
	tests := []struct {
		name            string
		args            args
		numWorkers      string
		wantStatus      int
		wantHeader      http.Header
		wantBodySuccess map[string]int
		wantBodyFailure string
	}{
		{
			name: "test coordinator handler",
			args: args{
				w: httptest.NewRecorder(),
				r: &http.Request{
					Method: http.MethodPost,
					Body:   io.NopCloser(strings.NewReader("lorem lorem\nlorem ipsum\nipsum sit")),
				},
			},
			numWorkers: "3",
			wantStatus: http.StatusOK,
			wantHeader: http.Header{
				"Content-Type": []string{"application/json"},
			},
			wantBodySuccess: map[string]int{
				"lorem": 3,
				"ipsum": 2,
				"sit":   1,
			},
		},
		{
			name: "test coordinator handler wrong request method",
			args: args{
				w: httptest.NewRecorder(),
				r: &http.Request{
					Method: http.MethodGet,
					Body:   io.NopCloser(strings.NewReader("lorem lorem\nlorem ipsum\nipsum sit")),
				},
			},
			numWorkers: "3",
			wantStatus: http.StatusMethodNotAllowed,
			wantHeader: http.Header{
				"Content-Type":           []string{"text/plain; charset=utf-8"},
				"X-Content-Type-Options": []string{"nosniff"},
			},
			wantBodyFailure: "Method Not Allowed\n",
		},
		{
			name: "test coordinator handler wrong number of workers",
			args: args{
				w: httptest.NewRecorder(),
				r: &http.Request{
					Method: http.MethodPost,
					Body:   io.NopCloser(strings.NewReader("lorem lorem\nlorem ipsum\nipsum sit")),
				},
			},
			numWorkers: "3a",
			wantStatus: http.StatusInternalServerError,
			wantHeader: http.Header{
				"Content-Type":           []string{"text/plain; charset=utf-8"},
				"X-Content-Type-Options": []string{"nosniff"},
			},
			wantBodyFailure: "Internal Server Error\n",
		},
		{
			name: "test coordinator handler map fail",
			args: args{
				w: httptest.NewRecorder(),
				r: &http.Request{
					Method: http.MethodPost,
					Body:   io.NopCloser(strings.NewReader("lorem lorem\nlorem ipsum\nipsum sit pacet")),
				},
			},
			numWorkers: "3",
			wantStatus: http.StatusInternalServerError,
			wantHeader: http.Header{
				"Content-Type":           []string{"text/plain; charset=utf-8"},
				"X-Content-Type-Options": []string{"nosniff"},
			},
			wantBodyFailure: "Internal Server Error\n",
		},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("HTTP_WORKERS_NUM", tt.numWorkers)
			coordinatorHandler(tt.args.w, tt.args.r)

			assert.Equalf(t, tt.wantStatus, tt.args.w.Code, "coordinatorHandler() = %d, expected status code: %d", tt.args.w.Code, tt.wantStatus)

			if !reflect.DeepEqual(tt.args.w.Header(), tt.wantHeader) {
				t.Errorf("coordinatorHandler() = %v, want %v", tt.args.w.Header(), tt.wantHeader)
			}

			if tt.args.w.Code == http.StatusOK {
				response := map[string]int{}
				json.Unmarshal(tt.args.w.Body.Bytes(), &response)
				if !reflect.DeepEqual(response, tt.wantBodySuccess) {
					t.Errorf("coordinatorHandler() = %v, want %v", response, tt.wantBodySuccess)
				}
			} else {
				response := tt.args.w.Body.String()
				if !reflect.DeepEqual(response, tt.wantBodyFailure) {
					t.Errorf("coordinatorHandler() = %v, want %v", response, tt.wantBodyFailure)
				}
			}

		})
	}
}

func slicesDeepEqual(a, b []map[string]int) bool {
	if len(a) != len(b) {
		return false
	}

	visited := make([]bool, len(b))
	for _, mapA := range a {
		found := false
		for i := 0; i < len(b); i++ {
			if visited[i] {
				continue
			}
			if reflect.DeepEqual(mapA, b[i]) {
				visited[i] = true
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func mapServerHandler(w http.ResponseWriter, r *http.Request) {

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Errorf("Error reading request body: %s", err)
		return
	}

	// successful mapping
	switch string(body) {
	case "lorem lorem":
		w.Write([]byte(`[{"lorem":1},{"lorem":1}]`))
		return
	case "lorem ipsum":
		w.Write([]byte(`[{"lorem":1},{"ipsum":1}]`))
		return
	case "ipsum sit":
		w.Write([]byte(`[{"ipsum":1},{"sit":1}]`))
		return
	}

	// otherwise send non-JSON gibberish
	w.Write([]byte(`blah blah`))
}

func shuffleServerHandler(w http.ResponseWriter, r *http.Request) {

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Errorf("Error reading request body: %s", err)
		return
	}

	// successful shuffling
	switch string(body) {
	case "[{\"lorem\":1},{\"lorem\":1},{\"lorem\":1}]":
		w.Write([]byte(`{"lorem":[1,1,1]}`))
		return
	case "[{\"ipsum\":1},{\"ipsum\":1}]":
		w.Write([]byte(`{"ipsum":[1,1]}`))
		return
	case "[{\"sit\":1}]":
		w.Write([]byte(`{"sit":[1]}`))
		return
	}

	// otherwise send non-JSON gibberish
	w.Write([]byte(`blah blah`))
}

func reduceServerHandler(w http.ResponseWriter, r *http.Request) {

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		log.Errorf("Error reading request body: %s", err)
		return
	}

	// successful reduce
	switch string(body) {
	case `{"lorem":[1,1,1]}`:
		w.Write([]byte(`{"lorem":3}`))
		return
	case `{"sit":[1]}`:
		w.Write([]byte(`{"sit":1}`))
		return
	case `{"ipsum":[1,1]}`:
		w.Write([]byte(`{"ipsum":2}`))
		return
	}

	// otherwise send non-JSON gibberish
	w.Write([]byte(`blah blah`))
}

func TestMain(m *testing.M) {

	mapServer = httptest.NewServer(http.HandlerFunc(mapServerHandler))
	defer mapServer.Close()

	shuffleServer = httptest.NewServer(http.HandlerFunc(shuffleServerHandler))
	defer shuffleServer.Close()

	reduceServer = httptest.NewServer(http.HandlerFunc(reduceServerHandler))
	defer reduceServer.Close()

	// add shuffle server in DNS lookup
	shufflers_nr := 6
	shufflers := make([]string, shufflers_nr)
	for i := 0; i < shufflers_nr; i++ {
		shufflers[i] = shuffleServer.Listener.Addr().(*net.TCPAddr).IP.String()
	}
	srv, _ := mockdns.NewServer(map[string]mockdns.Zone{
		shuffleServerARecord: {
			A: shufflers,
		},
	}, false)
	defer srv.Close()

	srv.PatchNet(net.DefaultResolver)
	defer mockdns.UnpatchNet(net.DefaultResolver)

	os.Exit(m.Run())
}
