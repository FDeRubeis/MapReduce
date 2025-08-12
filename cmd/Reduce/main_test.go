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

func Test_reduceHandler(t *testing.T) {

	type args struct {
		r *http.Request
		w *httptest.ResponseRecorder
	}
	tests := []struct {
		name            string
		args            args
		wantStatus      int
		wantHeader      http.Header
		wantBodySuccess map[string]int
		wantBodyFailure string
	}{
		{
			name: "test reduce handler",
			args: args{
				w: httptest.NewRecorder(),
				r: &http.Request{
					Method: http.MethodPost,
					Body:   io.NopCloser(strings.NewReader("{\"lorem\": [2, 1], \"ipsum\": [1, 1], \"sit\": [1]}")),
				},
			},
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
			name: "test reduce handler wrong request method",
			args: args{
				w: httptest.NewRecorder(),
				r: &http.Request{
					Method: http.MethodGet,
					Body:   io.NopCloser(strings.NewReader("{\"lorem\": [2, 1], \"ipsum\": [1, 1], \"sit\": [1]}")),
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
			name: "test map handler bad input formatting",
			args: args{
				w: httptest.NewRecorder(),
				r: &http.Request{
					Method: http.MethodPost,
					Body:   io.NopCloser(strings.NewReader("rwbcs\"lorem\": [2cssc, 1], \"ipsum\": [1, 1]]], \"sit\": [1]}")),
				},
			},
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

			reduceHandler(tt.args.w, tt.args.r)

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
