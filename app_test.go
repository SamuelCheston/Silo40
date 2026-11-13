package main

import (
	"embed"
	"testing"
)

func TestHasActiveGameBeforeStartup(t *testing.T) {
	app := &App{}
	active, err := app.HasActiveGame()
	if err != nil {
		t.Fatalf("pre-startup query returned an error: %v", err)
	}
	if active {
		t.Fatal("pre-startup query must report no active game")
	}
}

func TestHasActiveGameWaitsForStartupReady(t *testing.T) {
	app := NewApp(embed.FS{})
	result := make(chan error, 1)
	go func() {
		_, err := app.HasActiveGame()
		result <- err
	}()
	close(app.ready)
	if err := <-result; err != nil {
		t.Fatalf("ready startup query returned an error: %v", err)
	}
}
