package request_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dylanlyu/pusher-go/internal/request"
)

func TestDo(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		wantErr      bool
		wantBody     string
	}{
		{
			name:         "200 returns body",
			statusCode:   http.StatusOK,
			responseBody: `{"id":"abc"}`,
			wantBody:     `{"id":"abc"}`,
		},
		{
			name:         "201 returns body",
			statusCode:   http.StatusCreated,
			responseBody: `{"id":"xyz"}`,
			wantBody:     `{"id":"xyz"}`,
		},
		{
			name:         "400 returns error",
			statusCode:   http.StatusBadRequest,
			responseBody: `{"error":"bad request"}`,
			wantErr:      true,
		},
		{
			name:         "500 returns error",
			statusCode:   http.StatusInternalServerError,
			responseBody: `internal server error`,
			wantErr:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				io.WriteString(w, tc.responseBody)
			}))
			defer srv.Close()

			body, err := request.Do(context.Background(), srv.Client(), http.MethodGet, srv.URL, nil, nil)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(body) != tc.wantBody {
				t.Errorf("body = %q, want %q", string(body), tc.wantBody)
			}
		})
	}
}

func TestDo_PostWithBody(t *testing.T) {
	payload := []byte(`{"event":"test"}`)
	var received []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	_, err := request.Do(context.Background(), srv.Client(), http.MethodPost, srv.URL, payload, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(received, payload) {
		t.Errorf("server received %q, want %q", received, payload)
	}
}

func TestDo_ExtraHeadersSent(t *testing.T) {
	var gotHeader string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Pusher-Library")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	headers := map[string]string{"X-Pusher-Library": "pusher-go/0.1"}
	_, err := request.Do(context.Background(), srv.Client(), http.MethodGet, srv.URL, nil, headers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotHeader != "pusher-go/0.1" {
		t.Errorf("header = %q, want %q", gotHeader, "pusher-go/0.1")
	}
}

func TestDo_ContextCancelled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := request.Do(ctx, srv.Client(), http.MethodGet, srv.URL, nil, nil)
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}
