package chezmoi

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	chezitconfig "github.com/daptify14/chezit/internal/config"
)

// Service wraps a Client with policy enforcement and use-case orchestration.
// It is the primary interface consumed by the TUI.
type Service struct {
	client *Client
	policy Policy
}

func NewService(client *Client, mode chezitconfig.Mode, targetPath string) *Service {
	return &Service{
		client: client,
		policy: NewPolicy(mode, targetPath),
	}
}

func (s *Service) Policy() Policy {
	return s.policy
}

func (s *Service) IsReadOnly() bool {
	return s.policy.IsReadOnly()
}

func (s *Service) TargetPath() string {
	return s.policy.TargetPath()
}

// --- Read operations (delegate to client, no policy checks) ---

func (s *Service) Status() ([]FileStatus, error)    { return s.client.Status() }
func (s *Service) StatusText() (string, error)      { return s.client.StatusText() }
func (s *Service) Diff(path string) (string, error) { return s.client.Diff(path) }
func (s *Service) DiffAll() (string, error)         { return s.client.DiffAll() }
func (s *Service) ManagedFiles() ([]string, error)  { return s.client.Managed() }
func (s *Service) ManagedFilesWithFilter(filter EntryFilter) ([]string, error) {
	return s.client.ManagedWithFilter(filter)
}
func (s *Service) IgnoredFiles() ([]string, error) { return s.client.Ignored() }
func (s *Service) IgnoredFilesWithFilter(filter EntryFilter) ([]string, error) {
	return s.client.IgnoredWithFilter(filter)
}

func (s *Service) UnmanagedFiles(filter ...EntryFilter) ([]string, error) {
	return s.client.Unmanaged(filter...)
}
func (s *Service) CatTarget(path string) (string, error) { return s.client.CatTarget(path) }
func (s *Service) CatConfig() (string, error)            { return s.client.CatConfig() }
func (s *Service) DumpConfig() (string, error)           { return s.client.DumpConfig() }
func (s *Service) DumpConfigJSON() (string, error)       { return s.client.DumpConfigJSON() }
func (s *Service) Data() (string, error)                 { return s.client.Data() }
func (s *Service) DataJSON() (string, error)             { return s.client.DataJSON() }
func (s *Service) Doctor() (string, error)               { return s.client.Doctor() }
func (s *Service) Verify() error                         { return s.client.Verify() }
func (s *Service) SourceDir() (string, error)            { return s.client.SourceDir() }
func (s *Service) GitBranchInfo() (GitInfo, error)       { return s.client.GitBranchInfo() }
func (s *Service) GitStatus() (staged, unstaged []GitFile, err error) {
	return s.client.GitStatusFiles()
}

func (s *Service) GitDiff(path string, staged bool) (string, error) {
	return s.client.GitDiff(path, staged)
}
func (s *Service) GitLog() (string, error)             { return s.client.GitLog() }
func (s *Service) GitLogUnpushed() (string, error)     { return s.client.GitLogUnpushed() }
func (s *Service) GitLogIncoming() (string, error)     { return s.client.GitLogIncoming() }
func (s *Service) GitShow(hash string) (string, error) { return s.client.GitShow(hash) }

// GitFetch is allowed in read-only mode — fetch only updates remote-tracking refs.
func (s *Service) GitFetch() error {
	return s.client.GitFetch()
}

func (s *Service) GitPull() error {
	if err := s.policy.CheckMutation(); err != nil {
		return err
	}
	return s.client.GitPull()
}

// --- Aggregated read operations ---

// LoadStatus combines chezmoi status with git status (skips git in read-only mode).
func (s *Service) LoadStatus() (StatusSnapshot, error) {
	files, err := s.client.Status()
	if err != nil {
		return StatusSnapshot{}, err
	}
	snap := StatusSnapshot{Files: files}

	if !s.policy.IsReadOnly() {
		staged, unstaged, gitErr := s.client.GitStatusFiles()
		if gitErr == nil {
			snap.Staged = staged
			snap.Unstaged = unstaged
			info, _ := s.client.GitBranchInfo()
			snap.GitInfo = info
		}
	}
	return snap, nil
}

func (s *Service) LoadInfo(req LoadInfoRequest) (InfoSnapshot, error) {
	var content string
	var err error
	switch req.View {
	case InfoViewConfig:
		content, err = s.client.CatConfig()
	case InfoViewFull:
		if req.Format == "json" {
			content, err = s.client.DumpConfigJSON()
		} else {
			content, err = s.client.DumpConfig()
		}
	case InfoViewData:
		if req.Format == "json" {
			content, err = s.client.DataJSON()
		} else {
			content, err = s.client.Data()
		}
	case InfoViewDoctor:
		content, err = s.client.Doctor()
	}
	if err != nil {
		return InfoSnapshot{}, err
	}
	return InfoSnapshot{View: req.View, Content: content}, nil
}

// --- Mutation operations (all check policy.CheckMutation before delegating) ---

func (s *Service) ReAdd(path string) error {
	if err := s.policy.CheckMutation(); err != nil {
		return err
	}
	return s.client.ReAdd(path)
}

func (s *Service) ReAddAll() (string, error) {
	if err := s.policy.CheckMutation(); err != nil {
		return "", err
	}
	return s.client.ReAddAll()
}

func (s *Service) Forget(path string) error {
	if err := s.policy.CheckMutation(); err != nil {
		return err
	}
	return s.client.Forget(path)
}

// Add also validates that path is within the target directory.
func (s *Service) Add(path string, opts AddOptions) error {
	if err := s.policy.CheckMutation(); err != nil {
		return err
	}
	if err := s.policy.ValidateTargetPath(path); err != nil {
		return err
	}
	return s.client.AddWithOptions(path, opts)
}

func (s *Service) GitAdd(path string) error {
	if err := s.policy.CheckMutation(); err != nil {
		return err
	}
	return s.client.GitAdd(path)
}

func (s *Service) GitAddAll() error {
	if err := s.policy.CheckMutation(); err != nil {
		return err
	}
	return s.client.GitAddAll()
}

func (s *Service) GitReset(path string) error {
	if err := s.policy.CheckMutation(); err != nil {
		return err
	}
	return s.client.GitReset(path)
}

func (s *Service) GitResetAll() error {
	if err := s.policy.CheckMutation(); err != nil {
		return err
	}
	return s.client.GitResetAll()
}

func (s *Service) GitCheckoutFile(path string) error {
	if err := s.policy.CheckMutation(); err != nil {
		return err
	}
	return s.client.GitCheckoutFile(path)
}

func (s *Service) GitSoftReset() error {
	if err := s.policy.CheckMutation(); err != nil {
		return err
	}
	return s.client.GitSoftReset()
}

func (s *Service) GitCommit(msg string) error {
	if err := s.policy.CheckMutation(); err != nil {
		return err
	}
	return s.client.Commit(msg)
}

func (s *Service) GitPush() error {
	if err := s.policy.CheckMutation(); err != nil {
		return err
	}
	return s.client.Push()
}

// --- Interactive commands (return *exec.Cmd for tea.ExecProcess, nil in read-only mode) ---

func (s *Service) ApplyCmd(path string) *exec.Cmd {
	if s.policy.IsReadOnly() {
		return nil
	}
	return s.client.ApplyCmd(path)
}

func (s *Service) ApplyAllCmd() *exec.Cmd {
	if s.policy.IsReadOnly() {
		return nil
	}
	return s.client.ApplyAllCmd()
}

func (s *Service) ApplyRefreshCmd() *exec.Cmd {
	if s.policy.IsReadOnly() {
		return nil
	}
	return s.client.ApplyRefreshCmd()
}

func (s *Service) ApplyDryRunCmd() *exec.Cmd {
	return s.client.ApplyDryRunCmd()
}

func (s *Service) ApplyRefreshDryRunCmd() *exec.Cmd {
	return s.client.ApplyRefreshDryRunCmd()
}

func (s *Service) UpdateCmd() *exec.Cmd {
	if s.policy.IsReadOnly() {
		return nil
	}
	return s.client.UpdateCmd()
}

func (s *Service) InitCmd() *exec.Cmd {
	if s.policy.IsReadOnly() {
		return nil
	}
	return s.client.InitCmd()
}

func (s *Service) EditCmd(path string) *exec.Cmd {
	if s.policy.IsReadOnly() {
		return nil
	}
	return s.client.EditCmd(path)
}

func (s *Service) EditSourceCmd() *exec.Cmd {
	if s.policy.IsReadOnly() {
		return nil
	}
	return s.client.EditSourceCmd()
}

// EditConfigCmd is not gated by read-only: read-only protects dotfiles, not chezmoi config.
func (s *Service) EditConfigCmd() *exec.Cmd {
	return s.client.EditConfigCmd()
}

// EditConfigTemplateCmd is not gated by read-only (same reason as EditConfigCmd).
func (s *Service) EditConfigTemplateCmd() *exec.Cmd {
	return s.client.EditConfigTemplateCmd()
}

// --- Archive operations ---

// Archive creates a timestamped tar.gz of the target state. Returns the output path.
// Not gated by read-only: archiving is a read operation.
func (s *Service) Archive() (string, error) {
	outputPath, err := s.archiveOutputPath()
	if err != nil {
		return "", err
	}
	if err := s.client.Archive(outputPath); err != nil {
		return "", err
	}
	return outputPath, nil
}

func (s *Service) archiveOutputPath() (string, error) {
	dir := s.ArchiveOutputDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create archive directory: %w", err)
	}
	filename := fmt.Sprintf("chezmoi-archive-%s.tar.gz", time.Now().Format("20060102-150405"))
	return filepath.Join(dir, filename), nil
}

func (s *Service) ArchiveOutputDir() string {
	dataDir, err := os.UserHomeDir()
	if err != nil {
		dataDir = os.TempDir()
	}
	return filepath.Join(dataDir, ".local", "share", "chezit", "archives")
}

func (s *Service) AvailableCommands() []CommandAvailability {
	return s.policy.AvailableCommands(
		s.client.EditSourceCmd() != nil,
		s.client.EditConfigCmd() != nil,
	)
}
