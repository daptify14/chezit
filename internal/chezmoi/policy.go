package chezmoi

import (
	"path/filepath"
	"strings"

	chezitconfig "github.com/daptify14/chezit/internal/config"
)

// Policy enforces mutation guards and target path validation.
type Policy struct {
	mode       chezitconfig.Mode
	targetPath string
}

func NewPolicy(mode chezitconfig.Mode, targetPath string) Policy {
	return Policy{mode: mode, targetPath: targetPath}
}

func (p Policy) IsReadOnly() bool {
	return p.mode == chezitconfig.ModeReadOnly
}

func (p Policy) CheckMutation() error {
	if p.IsReadOnly() {
		return ErrReadOnly
	}
	return nil
}

// ValidateTargetPath rejects empty, relative, or out-of-bounds paths.
func (p Policy) ValidateTargetPath(absPath string) error {
	if absPath == "" {
		return ErrPathEmpty
	}
	if !filepath.IsAbs(absPath) {
		return ErrPathNotAbs
	}
	if p.targetPath == "" {
		return ErrOutsideTarget
	}
	cleanAbs := filepath.Clean(absPath)
	cleanTarget := filepath.Clean(p.targetPath)
	if cleanAbs == cleanTarget {
		return nil
	}
	if strings.HasPrefix(cleanAbs, cleanTarget+string(filepath.Separator)) {
		return nil
	}
	return ErrOutsideTarget
}

func (p Policy) TargetPath() string {
	return p.targetPath
}

// AvailableCommands builds the Commands tab list, filtering by mode and editor availability.
func (p Policy) AvailableCommands(hasEditSource, hasEditConfig bool) []CommandAvailability {
	readOnly := p.IsReadOnly()
	cmds := make([]CommandAvailability, 0, 16)

	if !readOnly {
		cmds = append(cmds,
			CommandAvailability{
				Label: "Apply", Description: "Apply source state to destination",
				Command: "chezmoi apply", Category: "apply",
				Available: true, SupportsDryRun: true,
			},
			CommandAvailability{
				Label: "Update", Description: "Pull from remote and apply",
				Command: "chezmoi update", Category: "apply",
				Available: true,
			},
			CommandAvailability{
				Label: "Refresh Externals", Description: "Re-download external files and apply",
				Command: "chezmoi apply --refresh-externals", Category: "apply",
				Available: true, SupportsDryRun: true,
			},
			CommandAvailability{
				Label: "Re-Add All", Description: "Re-add all files from destination to source",
				Command: "chezmoi re-add", Category: "apply",
				Available: true,
			},
			CommandAvailability{
				Label: "Init", Description: "Interactive chezmoi init",
				Command: "chezmoi init", Category: "apply",
				Available: true,
			},
		)
	}

	cmds = append(cmds,
		CommandAvailability{
			Label: "Status", Description: "Show file status summary",
			Command: "chezmoi status", Category: "info",
			Available: true,
		},
		CommandAvailability{
			Label: "Diff All", Description: "Show combined diff for all files",
			Command: "chezmoi diff", Category: "info",
			Available: true,
		},
		CommandAvailability{
			Label: "Doctor", Description: "Run diagnostics and check configuration",
			Command: "chezmoi doctor", Category: "info",
			Available: true,
		},
		CommandAvailability{
			Label: "Verify", Description: "Check if destination matches source",
			Command: "chezmoi verify", Category: "info",
			Available: true,
		},
		CommandAvailability{
			Label: "Data", Description: "View template data (for debugging)",
			Command: "chezmoi data --format=yaml", Category: "info",
			Available: true,
		},
		CommandAvailability{
			Label: "Cat Config", Description: "Show resolved configuration",
			Command: "chezmoi cat-config", Category: "info",
			Available: true,
		},
		CommandAvailability{
			Label: "Git Log", Description: "View recent source repo commits",
			Command: "git log --oneline -20", Category: "info",
			Available: true,
		},
		CommandAvailability{
			Label: "Archive", Description: "Create backup archive of target state",
			Command: "chezmoi archive --output=<path>", Category: "info",
			Available: true,
		},
	)

	if !readOnly && hasEditSource {
		cmds = append(cmds, CommandAvailability{
			Label: "Edit Source", Description: "Open source directory in $EDITOR",
			Command: "chezmoi edit", Category: "edit",
			Available: true,
		})
	}
	if hasEditConfig {
		cmds = append(cmds, CommandAvailability{
			Label: "Edit Config", Description: "Edit local config (changes lost on init if template exists)",
			Command: "chezmoi edit-config", Category: "edit",
			Available: true,
		})
	}
	cmds = append(cmds, CommandAvailability{
		Label: "Edit Config Template", Description: "Edit config template (version-controlled)",
		Command: "chezmoi edit-config-template", Category: "edit",
		Available: true,
	})

	return cmds
}
