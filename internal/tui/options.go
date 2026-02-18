package tui

import (
	"log/slog"

	"github.com/daptify14/chezit/internal/chezmoi"
)

// EscBehavior defines what happens when the user presses the Escape key.
type EscBehavior int

const (
	// EscQuit exits the entire program.
	EscQuit EscBehavior = iota
	// EscBack returns control to the caller.
	EscBack
)

// Options configures the TUI model.
type Options struct {
	// Service is the chezmoi service that provides backend operations with policy enforcement.
	Service *chezmoi.Service

	// Breadcrumb defines the navigation breadcrumb trail.
	// Example: ["chezit", "Chezmoi"]
	Breadcrumb []string

	// EscBehavior determines what happens when Escape is pressed.
	// Use EscQuit to exit, EscBack to return control to the caller.
	EscBehavior EscBehavior

	// CommitPresets provides custom commit message templates.
	// These appear as quick-select options in the commit message input.
	CommitPresets []string

	// Editor overrides the $EDITOR environment variable for file editing.
	// Supports binary with arguments (e.g., "code --wait").
	// Resolution order: Editor > $EDITOR > "vi".
	Editor string

	// PanelMode controls default panel visibility: "auto" (default), "show", "hide".
	// "auto" shows when terminal >= 110 columns, "show" always shows, "hide" never shows.
	PanelMode string

	// InitialTab sets the active tab when the TUI starts.
	// Valid values: "Status", "Files", "Info", "Commands" (case-insensitive).
	// If set, the landing screen is skipped. If empty, default behavior is used.
	InitialTab string

	// IconMode controls which icon set to display next to filenames.
	// Valid values: IconModeNerdFont (default), IconModeUnicode, IconModeNone.
	IconMode IconMode

	// DebugLog, when non-nil, receives structured JSON logs of every tea.Msg
	// processed by Update(). Set via the CHEZIT_DEBUG environment variable.
	DebugLog *slog.Logger
}
