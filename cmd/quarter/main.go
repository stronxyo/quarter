package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"github.com/stronxyo/quarter/internal/config"
	"github.com/stronxyo/quarter/internal/library"
	"github.com/stronxyo/quarter/internal/ui"
)

// version is overridden at release time with -ldflags "-X main.version=<tag>".
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	list := flag.Bool("list", false, "print installed games as plain text and exit")
	steamPath := flag.String("steam-path", "", "Steam install folder (default: detect)")
	flag.Parse()

	var err error
	switch {
	case *showVersion:
		fmt.Printf("%s %s\n", config.AppName, version)
	case *list:
		err = listGames(os.Stdout, os.Stderr, *steamPath)
	default:
		err = runUI(*steamPath)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", config.AppName, err)
		os.Exit(1)
	}
}

func runUI(steamPath string) error {
	closeLog := setupLog()
	defer closeLog()
	_, err := tea.NewProgram(ui.New(steamPath)).Run()
	return err
}

// setupLog sends slog output to quarter.log in the data folder, since the UI
// owns the terminal. If the folder is unavailable, logs are dropped.
func setupLog() (closeLog func()) {
	dir, err := config.DataDir()
	if err != nil {
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
		return func() {}
	}
	f, err := os.Create(filepath.Join(dir, config.LogFile))
	if err != nil {
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
		return func() {}
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(f, nil)))
	return func() { f.Close() }
}

// listGames prints the library to out and any warnings to warn, so the list
// stays clean when redirected.
func listGames(out, warn io.Writer, steamPath string) error {
	res, err := library.Scan(steamPath)
	if err != nil {
		return err
	}
	for _, w := range res.Warnings {
		fmt.Fprintf(warn, "warning: %v\n", w)
	}

	summary := fmt.Sprintf("%d games", len(res.Games))
	width := utf8.RuneCountInString(summary)
	for _, g := range res.Games {
		width = max(width, utf8.RuneCountInString(g.Name))
	}

	var total time.Duration
	for _, g := range res.Games {
		total += g.Playtime
		fmt.Fprintf(out, "%-*s  %9s  %10s\n", width, g.Name, library.FormatHours(g.Playtime, res.PlaytimeKnown), library.FormatDate(g.LastPlayed))
	}
	fmt.Fprintf(out, "\n%-*s  %9s\n", width, summary, library.FormatHours(total, res.PlaytimeKnown))
	return nil
}
