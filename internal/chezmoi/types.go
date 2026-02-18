package chezmoi

import (
	"os/exec"
	"strings"
)

// EntryType values map to chezmoi --include/--exclude flags.
type EntryType string

const (
	EntryDirs      EntryType = "dirs"
	EntryFiles     EntryType = "files"
	EntryTemplates EntryType = "templates"
	EntryEncrypted EntryType = "encrypted"
	EntryExternals EntryType = "externals"
	EntryScripts   EntryType = "scripts"
	EntrySymlinks  EntryType = "symlinks"
	EntryAlways    EntryType = "always"
)

func AllEntryTypes() []EntryType {
	return []EntryType{
		EntryDirs, EntryFiles, EntryTemplates, EntryEncrypted,
		EntryExternals, EntryScripts, EntrySymlinks, EntryAlways,
	}
}

type EntryFilter struct {
	Include []EntryType // chezmoi --include flags; empty means include all
	Exclude []EntryType // chezmoi --exclude flags
}

func (f EntryFilter) IsZero() bool {
	return len(f.Include) == 0 && len(f.Exclude) == 0
}

func entryFilterArgs(f EntryFilter) []string {
	var args []string
	for _, t := range f.Include {
		args = append(args, "--include="+string(t))
	}
	for _, t := range f.Exclude {
		args = append(args, "--exclude="+string(t))
	}
	return args
}

// FileStatus is a parsed line from `chezmoi status`.
type FileStatus struct {
	Path         string
	SourceStatus rune
	DestStatus   rune
	IsTemplate   bool // true if source file is a .tmpl template
}

// SideLabel returns a drift subtype label: "pending apply", "target changed",
// "diverged", or "pending script run".
func (f FileStatus) SideLabel() string {
	src := f.SourceStatus != ' '
	dest := f.DestStatus != ' '
	if f.IsScript() && f.SourceStatus == 'R' && !dest {
		return "pending script run"
	}
	switch {
	case src && dest:
		return "diverged"
	case src:
		return "pending apply"
	case dest:
		return "target changed"
	default:
		return ""
	}
}

func (f FileStatus) IsModified() bool {
	return f.SourceStatus != ' ' || f.DestStatus != ' '
}

func (f FileStatus) IsScript() bool {
	if f.Path == "" {
		return false
	}
	normalized := strings.ReplaceAll(f.Path, "\\", "/")
	return strings.Contains(normalized, "/.chezmoiscripts/") || strings.HasPrefix(normalized, ".chezmoiscripts/")
}

// GitFile is a parsed entry from `git status --porcelain`.
type GitFile struct {
	Path       string
	StatusCode string
}

type GitInfo struct {
	Branch string
	Ahead  int
	Behind int
	Remote string
}

// GitCommit is a parsed line from `git log --oneline`.
type GitCommit struct {
	Hash    string // abbreviated commit hash
	Message string // first line of commit message
}

// InteractiveCmd wraps commands that require TTY (edit, apply, update).
type InteractiveCmd struct {
	Cmd *exec.Cmd
}

// --- Service-level types ---

type StatusSnapshot struct {
	Files    []FileStatus
	Staged   []GitFile
	Unstaged []GitFile
	GitInfo  GitInfo
}

type FileKind int

const (
	FileKindManaged FileKind = iota
	FileKindIgnored
	FileKindUnmanaged
)

type LoadFilesRequest struct {
	Kind        FileKind
	EntryFilter EntryFilter
}

type FilesSnapshot struct {
	Kind  FileKind
	Files []string
}

type InfoView int

const (
	InfoViewConfig InfoView = iota // cat-config
	InfoViewFull                   // dump-config
	InfoViewData                   // template data
	InfoViewDoctor                 // health check
)

type LoadInfoRequest struct {
	View   InfoView
	Format string // "yaml" or "json"
}

type InfoSnapshot struct {
	View    InfoView
	Content string
}

type ActionKind int

const (
	ActionReAdd ActionKind = iota
	ActionReAddAll
	ActionForget
	ActionAdd
	ActionGitAdd
	ActionGitAddAll
	ActionGitReset
	ActionGitResetAll
	ActionGitCommit
	ActionGitPush
)

type ActionRequest struct {
	Kind       ActionKind
	Path       string
	AddOptions AddOptions
	CommitMsg  string
}

type ActionResult struct {
	Kind    ActionKind
	Message string
}

type CommandAvailability struct {
	Label          string
	Description    string
	Command        string
	Category       string
	Available      bool
	SupportsDryRun bool
}
