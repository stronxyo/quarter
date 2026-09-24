package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"
	"unicode/utf8"

	"github.com/stronxyo/quarter/internal/config"
	"github.com/stronxyo/quarter/internal/library"
	"github.com/stronxyo/quarter/internal/steam"
)

// version is overridden at release time with -ldflags "-X main.version=<tag>".
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	list := flag.Bool("list", false, "print installed games as plain text and exit")
	steamPath := flag.String("steam-path", "", "Steam install folder (default: detect)")
	flag.Parse()

	switch {
	case *showVersion:
		fmt.Printf("%s %s\n", config.AppName, version)
	case *list:
		if err := listGames(os.Stdout, os.Stderr, *steamPath); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", config.AppName, err)
			os.Exit(1)
		}
	default:
		fmt.Printf("%s: nothing to show yet. Try --list or --version.\n", config.AppName)
	}
}

// listGames prints the library to out and any warnings to warn, so the list
// stays clean when redirected.
func listGames(out, warn io.Writer, steamPathFlag string) error {
	steamPath, err := steam.Locate(steamPathFlag)
	if err != nil {
		return err
	}

	installed, warnings := steam.InstalledGames(steamPath)

	var stats map[uint32]steam.PlayStats
	acct, err := steam.ActiveAccount(steamPath)
	if err == nil {
		stats, err = steam.ReadPlayStats(steamPath, acct)
	}
	if err != nil {
		warnings = append(warnings, fmt.Errorf("playtime unavailable: %w", err))
	}

	for _, w := range warnings {
		fmt.Fprintf(warn, "warning: %v\n", w)
	}

	games := library.FromSteam(installed, stats)
	summary := fmt.Sprintf("%d games", len(games))
	width := utf8.RuneCountInString(summary)
	for _, g := range games {
		width = max(width, utf8.RuneCountInString(g.Name))
	}

	var total time.Duration
	for _, g := range games {
		total += g.Playtime
		fmt.Fprintf(out, "%-*s  %9s  %10s\n", width, g.Name, hours(g.Playtime, stats != nil), date(g.LastPlayed))
	}
	fmt.Fprintf(out, "\n%-*s  %9s\n", width, summary, hours(total, stats != nil))
	return nil
}

func hours(d time.Duration, known bool) string {
	if !known {
		return "?"
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

func date(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02")
}
