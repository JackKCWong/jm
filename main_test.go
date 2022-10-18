package main

import (
	"context"
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

	var app app
	app.Input = strings.NewReader(server.URL + "/json1\n" + server.URL + "/json2\n")
	app.Concurrency = 2

	err := app.Run(context.Background(), []string{"key1"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestJsonArrayFromRemote(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/json1", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"key1": "value1", "key2": true}, {"key1": "value2", "key2": false}]`))
	}))

	server := httptest.NewServer(mux)
	defer server.Close()

	var app app
	app.Input = strings.NewReader(server.URL + "/json1\n")
	app.Concurrency = 2

	err := app.Run(context.Background(), []string{"key2"})
	if err != nil {
		t.Fatal(err)
	}
}
