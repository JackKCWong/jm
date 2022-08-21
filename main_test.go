package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJsonFromRemote(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/json1", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"key1": "value1", "key2": true}`))
	}))
	mux.Handle("/json2", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"key1": "value2", "key2": false}`))
	}))

	server := httptest.NewServer(mux)
	defer server.Close()

	// TODO: refactor
	err := runMain(strings.NewReader(server.URL + "/json1\n" + server.URL + "/json2\n"), "key1")
	if err != nil {
		t.Fatal(err)
	}
}