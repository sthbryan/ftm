package service

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func serveOn(t *testing.T, body string) int {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.Listener.Addr().(*net.TCPAddr).Port
}

func TestFoundryOnTellsFoundryFromOtherServers(t *testing.T) {
	foundry := serveOn(t, "<!DOCTYPE html><html><head><title>Foundry Virtual Tabletop</title></head></html>")
	other := serveOn(t, "<html><title>Something else</title></html>")

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	closed := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()

	got := (&Service{}).FoundryOn(context.Background(), []int{other, closed, foundry})
	if len(got) != 1 || got[0].Port != foundry || got[0].Version != "" {
		t.Fatalf("FoundryOn = %+v, want only %d, found by its title", got, foundry)
	}
}

func TestFoundryOnReadsTheStatusEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/status" {
			_, _ = w.Write([]byte(`{"active":true,"version":"12.331","world":"curse-of-strahd","system":"dnd5e","systemVersion":"4.1.2","users":3,"uptime":4270000}`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	port := srv.Listener.Addr().(*net.TCPAddr).Port

	got := (&Service{}).FoundryOn(context.Background(), []int{port})
	want := FoundryStatus{Port: port, Active: true, Version: "12.331", World: "curse-of-strahd", System: "dnd5e", SystemVersion: "4.1.2", Users: 3, UptimeMs: 4270000}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("FoundryOn = %+v, want %+v", got, want)
	}
}

func TestFoundryOnTheSetupScreenHasNoWorld(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"active":false,"version":"12.331"}`))
	}))
	t.Cleanup(srv.Close)
	port := srv.Listener.Addr().(*net.TCPAddr).Port

	got := (&Service{}).FoundryOn(context.Background(), []int{port})
	if len(got) != 1 || got[0].Active || got[0].World != "" {
		t.Fatalf("FoundryOn = %+v, want Foundry with no world", got)
	}
}

// A real answer from Foundry 14.368 with a dnd4e world launched.
func TestFoundryOnReadsARealFoundry14Answer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"active":true,"version":"14.368","world":"test","system":"dnd4e","systemVersion":"0.9.3","users":1,"uptime":6742}`))
	}))
	t.Cleanup(srv.Close)
	port := srv.Listener.Addr().(*net.TCPAddr).Port

	got := (&Service{}).FoundryOn(context.Background(), []int{port})
	want := FoundryStatus{Port: port, Active: true, Version: "14.368", World: "test", System: "dnd4e", SystemVersion: "0.9.3", Users: 1, UptimeMs: 6742}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("FoundryOn = %+v, want %+v", got, want)
	}
}
