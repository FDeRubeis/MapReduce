package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_mapHandler(t *testing.T) {

	type args struct {
		r *http.Request
		w *httptest.ResponseRecorder
	}
	tests := []struct {
		name            string
		args            args
		wantStatus      int
		wantHeader      http.Header
		wantBodySuccess []map[string]int
		wantBodyFailure string
	}{
		{
			name: "test map handler",
			args: args{
				w: httptest.NewRecorder(),
				r: &http.Request{
					Method: http.MethodPost,
					Body:   io.NopCloser(strings.NewReader("lorem lorem\nlorem ipsum\nipsum sit")),
				},
			},
			wantStatus: http.StatusOK,
			wantHeader: http.Header{
				"Content-Type": []string{"application/json"},
			},
			wantBodySuccess: []map[string]int{
				{"lorem": 1},
				{"lorem": 1},
				{"lorem": 1},
				{"ipsum": 1},
				{"ipsum": 1},
				{"sit": 1},
			},
		},
		{
			name: "test map handler wrong request method",
			args: args{
				w: httptest.NewRecorder(),
				r: &http.Request{
					Method: http.MethodGet,
					Body:   io.NopCloser(strings.NewReader("lorem lorem\nlorem ipsum\nipsum sit")),
				},
			},
			wantStatus: http.StatusMethodNotAllowed,
			wantHeader: http.Header{
				"Content-Type":           []string{"text/plain; charset=utf-8"},
				"X-Content-Type-Options": []string{"nosniff"},
			},
			wantBodyFailure: "Method Not Allowed\n",
		},
		{
			name: "test map handler preprocessing",
			args: args{
				w: httptest.NewRecorder(),
				r: &http.Request{
					Method: http.MethodPost,
					Body:   io.NopCloser(strings.NewReader("lorem Lorem\n{}{}}}lorEm ipsUm\nipsum!!!    sit%#$^")),
				},
			},
			wantStatus: http.StatusOK,
			wantHeader: http.Header{
				"Content-Type": []string{"application/json"},
			},
			wantBodySuccess: []map[string]int{
				{"lorem": 1},
				{"lorem": 1},
				{"lorem": 1},
				{"ipsum": 1},
				{"ipsum": 1},
				{"sit": 1},
			},
		},
	}
	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			mapHandler(tt.args.w, tt.args.r)

			assert.Equalf(t, tt.wantStatus, tt.args.w.Code, "coordinatorHandler() = %d, expected status code: %d", tt.args.w.Code, tt.wantStatus)

			if !reflect.DeepEqual(tt.args.w.Header(), tt.wantHeader) {
				t.Errorf("coordinatorHandler() = %v, want %v", tt.args.w.Header(), tt.wantHeader)
			}

			if tt.args.w.Code == http.StatusOK {
				response := []map[string]int{}
				json.Unmarshal(tt.args.w.Body.Bytes(), &response)
				if !slicesDeepEqual(response, tt.wantBodySuccess) {
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
