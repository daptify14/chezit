// Command chezit is a terminal UI for chezmoi dotfile management.
package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/daptify14/chezit/internal/chezmoi"
	chezitconfig "github.com/daptify14/chezit/internal/config"
	"github.com/daptify14/chezit/internal/tui"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "chezit",
		Short: "Terminal UI for chezmoi dotfile management",
		Long:  "chezit is an interactive TUI for managing dotfiles with chezmoi. Browse changes, stage files, commit, and run chezmoi commands — all from a single interface.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI("")
		},
	}
	rootCmd.Version = version + " (commit " + commit + ", built " + date + ")"

	tabCommands := []struct {
		use   string
		short string
		tab   string
	}{
		{"status", "Open directly to the Status tab", "Status"},
		{"files", "Open directly to the Files tab", "Files"},
		{"info", "Open directly to the Info tab", "Info"},
		{"commands", "Open directly to the Commands tab", "Commands"},
	}

	for _, tc := range tabCommands {
		tab := tc.tab
		rootCmd.AddCommand(&cobra.Command{
			Use:   tc.use,
			Short: tc.short,
			RunE: func(cmd *cobra.Command, args []string) error {
				return runTUI(tab)
			},
		})
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runTUI(initialTab string) error {
	cfg, err := chezitconfig.Load()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	client := chezmoi.New(chezmoi.WithBinaryPath(cfg.BinaryPath))
	tp, err := client.TargetPath()
	if err != nil {
		return fmt.Errorf("could not determine chezmoi target path: %w", err)
	}
	svc := chezmoi.NewService(client, cfg.Mode, tp)

	iconMode, err := tui.ParseIconMode(cfg.Icons)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	var debugLog *slog.Logger
	if debugPath := os.Getenv("CHEZIT_DEBUG"); debugPath != "" {
		cleanPath := filepath.Clean(debugPath)
		f, err := os.OpenFile(cleanPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600) //#nosec G703 -- developer-controlled debug log path
		if err != nil {
			return fmt.Errorf("debug log: %w", err)
		}
		defer func() { _ = f.Close() }()
		debugLog = slog.New(slog.NewJSONHandler(f, nil))
	}

	opts := tui.Options{
		Service:       svc,
		EscBehavior:   tui.EscQuit,
		CommitPresets: cfg.CommitPresets,
		PanelMode:     cfg.Panel,
		IconMode:      iconMode,
		InitialTab:    initialTab,
		DebugLog:      debugLog,
	}

	model := tui.NewModel(opts)
	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error: %w", err)
	}
	return nil
}
