package chezmoi

import "strings"

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
	if len(f.Include) > 0 {
		args = append(args, "--include="+joinEntryTypes(f.Include))
	}
	if len(f.Exclude) > 0 {
		args = append(args, "--exclude="+joinEntryTypes(f.Exclude))
	}
	return args
}

func joinEntryTypes(types []EntryType) string {
	parts := make([]string, len(types))
	for i, t := range types {
		parts[i] = string(t)
	}
	return strings.Join(parts, ",")
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
	if f.IsScript() && (f.SourceStatus == 'R' || f.DestStatus == 'R') {
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

// GitSyncState describes HEAD relative to its upstream. The zero value is
// GitSyncUnknown: a failed comparison must never render as "synced".
type GitSyncState int

const (
	GitSyncUnknown GitSyncState = iota
	GitSyncNoUpstream
	GitSyncSynced
	GitSyncAhead
	GitSyncBehind
	GitSyncDiverged
)

// String is for debug logs and test failure messages.
func (s GitSyncState) String() string {
	switch s {
	case GitSyncUnknown:
		return "unknown"
	case GitSyncNoUpstream:
		return "no-upstream"
	case GitSyncSynced:
		return "synced"
	case GitSyncAhead:
		return "ahead"
	case GitSyncBehind:
		return "behind"
	case GitSyncDiverged:
		return "diverged"
	}
	return "unknown"
}

// classifySync maps ahead/behind counts to a sync state. Argument order
// matches the GitInfo field order.
func classifySync(ahead, behind int) GitSyncState {
	switch {
	case ahead > 0 && behind > 0:
		return GitSyncDiverged
	case ahead > 0:
		return GitSyncAhead
	case behind > 0:
		return GitSyncBehind
	}
	return GitSyncSynced
}

type GitInfo struct {
	Branch   string
	Upstream string // e.g. "origin/main"; empty unless Sync is Synced/Ahead/Behind/Diverged
	Ahead    int
	Behind   int
	Remote   string
	Sync     GitSyncState
}

// GitCommit is a parsed line from `git log --oneline`.
type GitCommit struct {
	Hash    string // abbreviated commit hash
	Message string // first line of commit message
}

type CommandAvailability struct {
	Label          string
	Description    string
	Command        string
	Category       string
	Available      bool
	SupportsDryRun bool
}
