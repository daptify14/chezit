// Package chezmoi wraps the chezmoi CLI in a three-layer architecture:
// Client (exec), Service (orchestration + policy), and Policy (mutation guards).
package chezmoi

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Client wraps the chezmoi CLI binary.
type Client struct {
	Timeout    time.Duration
	BinaryPath string
	Editor     string
}

type Option func(*Client)

func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.Timeout = d
	}
}

func WithBinaryPath(path string) Option {
	return func(c *Client) {
		c.BinaryPath = strings.TrimSpace(path)
	}
}

// WithEditor overrides $EDITOR in the command environment for edit commands.
func WithEditor(editor string) Option {
	return func(c *Client) {
		c.Editor = strings.TrimSpace(editor)
	}
}

func New(opts ...Option) *Client {
	c := &Client{
		Timeout:    30 * time.Second,
		BinaryPath: "chezmoi",
	}
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
	if c.BinaryPath == "" {
		c.BinaryPath = "chezmoi"
	}
	return c
}

func (c *Client) binary() string {
	if strings.TrimSpace(c.BinaryPath) == "" {
		return "chezmoi"
	}
	return c.BinaryPath
}

// baseFlags returns flags injected into every non-interactive command.
// These ensure machine-parseable output with no TTY prompts, pager, color
// codes, progress bars, or external diff tool interference.
func (c *Client) baseFlags() []string {
	return []string{
		"--no-tty",
		"--color=false",
		"--no-pager",
		"--progress=false",
		"--use-builtin-diff",
	}
}

func (c *Client) cmd(args ...string) (*exec.Cmd, context.CancelFunc) {
	allArgs := append(c.baseFlags(), args...)
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	cmd := exec.CommandContext(ctx, c.binary(), allArgs...)
	cmd.Stdin = nil
	return cmd, cancel
}

func (c *Client) run(args ...string) ([]byte, error) {
	cmd, cancel := c.cmd(args...)
	defer cancel()
	return cmd.CombinedOutput()
}

func IsAvailable() bool {
	return IsAvailableAt("chezmoi")
}

func IsAvailableAt(binaryPath string) bool {
	binaryPath = strings.TrimSpace(binaryPath)
	if binaryPath == "" {
		binaryPath = "chezmoi"
	}
	_, err := exec.LookPath(binaryPath)
	return err == nil
}

func (c *Client) IsAvailable() bool {
	return IsAvailableAt(c.binary())
}

// IsTracked checks via `chezmoi source-path` whether filePath is managed.
func (c *Client) IsTracked(filePath string) bool {
	cmd, cancel := c.cmd("source-path", filePath)
	defer cancel()
	return cmd.Run() == nil
}

// Status runs `chezmoi status` and parses the output.
func (c *Client) Status() ([]FileStatus, error) {
	output, err := c.run("status", "--path-style=absolute")
	if err != nil {
		return nil, fmt.Errorf("chezmoi status: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return ParseStatus(string(output)), nil
}

// Diff runs `chezmoi diff` for a single file.
func (c *Client) Diff(filePath string) (string, error) {
	output, err := c.run("diff", filePath)
	if err != nil {
		out := string(output)
		if out != "" && !strings.HasPrefix(out, "error:") {
			return out, nil
		}
		return "", fmt.Errorf("chezmoi diff: %s: %w", strings.TrimSpace(out), err)
	}
	return string(output), nil
}

// AddOptions maps to `chezmoi add` flags.
type AddOptions struct {
	Encrypt      bool // --encrypt
	Template     bool // --template
	AutoTemplate bool // --autotemplate
	Exact        bool // --exact (directories only)
	NoRecursive  bool // --recursive=false (directories only)
}

// Validate rejects mutually exclusive flag combinations.
func (o AddOptions) Validate() error {
	exclusive := 0
	if o.Encrypt {
		exclusive++
	}
	if o.Template {
		exclusive++
	}
	if o.AutoTemplate {
		exclusive++
	}
	if exclusive > 1 {
		return errors.New("invalid add options: only one of --encrypt, --template, --autotemplate may be specified")
	}
	return nil
}

func (o AddOptions) args() []string {
	var flags []string
	if o.Encrypt {
		flags = append(flags, "--encrypt")
	}
	if o.Template {
		flags = append(flags, "--template")
	}
	if o.AutoTemplate {
		flags = append(flags, "--autotemplate")
	}
	if o.Exact {
		flags = append(flags, "--exact")
	}
	if o.NoRecursive {
		flags = append(flags, "--recursive=false")
	}
	return flags
}

// Add runs `chezmoi add --force`.
func (c *Client) Add(filePath string) error {
	output, err := c.run("add", "--force", filePath)
	if err != nil {
		return fmt.Errorf("chezmoi add: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// AddWithOptions runs `chezmoi add --force` with extra flags from opts.
func (c *Client) AddWithOptions(filePath string, opts AddOptions) error {
	if strings.TrimSpace(filePath) == "" {
		return errors.New("chezmoi add: path must not be empty")
	}
	if err := opts.Validate(); err != nil {
		return err
	}
	args := []string{"add", "--force"}
	args = append(args, opts.args()...)
	args = append(args, "--", filePath)
	output, err := c.run(args...)
	if err != nil {
		return fmt.Errorf("chezmoi add: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// DumpConfigJSON runs `chezmoi dump-config --format=json`.
func (c *Client) DumpConfigJSON() (string, error) {
	output, err := c.run("dump-config", "--format=json")
	if err != nil {
		return "", fmt.Errorf("chezmoi dump-config: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return string(output), nil
}

// ReAdd runs `chezmoi re-add --force`. Errors if file is not tracked.
func (c *Client) ReAdd(filePath string) error {
	if !c.IsTracked(filePath) {
		return fmt.Errorf("file not tracked by chezmoi: %s (run: chezmoi add %s)", filePath, filePath)
	}
	output, err := c.run("re-add", "--force", filePath)
	if err != nil {
		return fmt.Errorf("chezmoi re-add: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// Managed runs `chezmoi managed --path-style=absolute --exclude=dirs`.
func (c *Client) Managed() ([]string, error) {
	output, err := c.run("managed", "--path-style=absolute", "--exclude=dirs")
	if err != nil {
		return nil, fmt.Errorf("chezmoi managed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return parseLines(output), nil
}

// ManagedWithFilter is like Managed but applies include/exclude filters.
// Preserves --exclude=dirs unless dirs is explicitly included.
func (c *Client) ManagedWithFilter(filter EntryFilter) ([]string, error) {
	args := []string{"managed", "--path-style=absolute"}
	merged := filter
	if !slices.Contains(merged.Include, EntryDirs) && !slices.Contains(merged.Exclude, EntryDirs) {
		merged.Exclude = append(slices.Clone(filter.Exclude), EntryDirs)
	}
	args = append(args, entryFilterArgs(merged)...)
	output, err := c.run(args...)
	if err != nil {
		return nil, fmt.Errorf("chezmoi managed: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return parseLines(output), nil
}

// Ignored runs `chezmoi ignored` and resolves paths to absolute.
func (c *Client) Ignored() ([]string, error) {
	output, err := c.run("ignored")
	if err != nil {
		return nil, fmt.Errorf("chezmoi ignored: %s: %w", strings.TrimSpace(string(output)), err)
	}
	target, targetErr := c.TargetPath()
	if targetErr != nil {
		return nil, fmt.Errorf("chezmoi target-path: %w", targetErr)
	}
	return parseLinesWithHome(output, target), nil
}

// Unmanaged runs `chezmoi unmanaged --path-style=absolute`.
func (c *Client) Unmanaged(filter ...EntryFilter) ([]string, error) {
	args := []string{"unmanaged", "--path-style=absolute"}
	if len(filter) > 0 {
		args = append(args, entryFilterArgs(filter[0])...)
	}
	output, err := c.run(args...)
	if err != nil {
		return nil, fmt.Errorf("chezmoi unmanaged: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return parseLines(output), nil
}

func parseLines(output []byte) []string {
	var files []string
	for line := range strings.SplitSeq(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files
}

// parseLinesWithHome joins each line with the home directory to produce absolute paths.
func parseLinesWithHome(output []byte, home string) []string {
	var files []string
	for line := range strings.SplitSeq(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, filepath.Join(home, line))
		}
	}
	return files
}

// DumpConfig runs `chezmoi dump-config --format=yaml`.
func (c *Client) DumpConfig() (string, error) {
	output, err := c.run("dump-config", "--format=yaml")
	if err != nil {
		return "", fmt.Errorf("chezmoi dump-config: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return string(output), nil
}

// SourceDir runs `chezmoi source-path`.
func (c *Client) SourceDir() (string, error) {
	output, err := c.run("source-path")
	if err != nil {
		return "", fmt.Errorf("chezmoi source-path: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return strings.TrimSpace(string(output)), nil
}

// TargetPath runs `chezmoi target-path`.
func (c *Client) TargetPath() (string, error) {
	output, err := c.run("target-path")
	if err != nil {
		return "", fmt.Errorf("chezmoi target-path: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GitRoot runs `chezmoi git rev-parse --show-toplevel`.
func (c *Client) GitRoot() (string, error) {
	output, err := c.run("git", "--", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("chezmoi git rev-parse: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (c *Client) ApplyRefreshCmd() *exec.Cmd {
	return exec.Command(c.binary(), "apply", "--refresh-externals")
}

func (c *Client) ApplyCmd(filePath string) *exec.Cmd {
	return exec.Command(c.binary(), "apply", filePath)
}

func (c *Client) ApplyAllCmd() *exec.Cmd {
	return exec.Command(c.binary(), "apply")
}

func (c *Client) ApplyDryRunCmd() *exec.Cmd {
	return exec.Command(c.binary(), "apply", "--dry-run", "-v")
}

func (c *Client) ApplyRefreshDryRunCmd() *exec.Cmd {
	return exec.Command(c.binary(), "apply", "--refresh-externals", "--dry-run", "-v")
}

func (c *Client) UpdateCmd() *exec.Cmd {
	return exec.Command(c.binary(), "update")
}

// Forget runs `chezmoi forget --force`.
func (c *Client) Forget(filePath string) error {
	output, err := c.run("forget", "--force", filePath)
	if err != nil {
		return fmt.Errorf("chezmoi forget: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// CatTarget runs `chezmoi cat` for the target-state content of a file.
func (c *Client) CatTarget(filePath string) (string, error) {
	output, err := c.run("cat", filePath)
	if err != nil {
		return "", fmt.Errorf("chezmoi cat: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return string(output), nil
}

func (c *Client) applyEditorEnv(cmd *exec.Cmd) {
	if c.Editor != "" {
		cmd.Env = append(os.Environ(), "EDITOR="+c.Editor)
	}
}

func (c *Client) EditCmd(filePath string) *exec.Cmd {
	cmd := exec.Command(c.binary(), "edit", filePath)
	c.applyEditorEnv(cmd)
	return cmd
}

// Push runs `chezmoi git push`.
func (c *Client) Push() error {
	output, err := c.run("git", "--", "push")
	if err != nil {
		return fmt.Errorf("chezmoi git push: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// Commit runs `chezmoi git commit`. Silently succeeds if nothing to commit.
func (c *Client) Commit(message string) error {
	commitOutput, err := c.run("git", "--", "commit", "-m", message)
	if err != nil {
		if strings.Contains(string(commitOutput), "nothing to commit") {
			return nil
		}
		return fmt.Errorf("chezmoi git commit: %s: %w", strings.TrimSpace(string(commitOutput)), err)
	}
	return nil
}

// GitStatusFiles runs `chezmoi git status --porcelain -u`.
func (c *Client) GitStatusFiles() (staged, unstaged []GitFile, err error) {
	output, err := c.run("git", "--", "status", "--porcelain", "-u")
	if err != nil {
		return nil, nil, fmt.Errorf("chezmoi git status: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return ParseGitPorcelain(string(output))
}

func (c *Client) GitAdd(path string) error {
	output, err := c.run("git", "--", "add", "--", path)
	if err != nil {
		return fmt.Errorf("chezmoi git add: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (c *Client) GitAddAll() error {
	output, err := c.run("git", "--", "add", "-A")
	if err != nil {
		return fmt.Errorf("chezmoi git add -A: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (c *Client) GitReset(path string) error {
	output, err := c.run("git", "--", "reset", "HEAD", "--", path)
	if err != nil {
		return fmt.Errorf("chezmoi git reset: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (c *Client) GitResetAll() error {
	output, err := c.run("git", "--", "reset", "HEAD")
	if err != nil {
		return fmt.Errorf("chezmoi git reset: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (c *Client) GitCheckoutFile(path string) error {
	output, err := c.run("git", "--", "checkout", "--", path)
	if err != nil {
		return fmt.Errorf("chezmoi git checkout: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (c *Client) GitSoftReset() error {
	output, err := c.run("git", "--", "reset", "--soft", "HEAD~1")
	if err != nil {
		return fmt.Errorf("chezmoi git reset --soft: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (c *Client) GitBranchInfo() (GitInfo, error) {
	var info GitInfo

	out, err := c.run("git", "--", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return info, fmt.Errorf("chezmoi git branch: %w", err)
	}
	info.Branch = strings.TrimSpace(string(out))

	out, _ = c.run("git", "--", "remote")
	info.Remote = strings.TrimSpace(strings.Split(string(out), "\n")[0])

	out, err = c.run("git", "--", "rev-list", "--left-right", "--count", "@{upstream}...HEAD")
	if err == nil {
		parts := strings.Fields(strings.TrimSpace(string(out)))
		if len(parts) == 2 {
			info.Behind, _ = strconv.Atoi(parts[0])
			info.Ahead, _ = strconv.Atoi(parts[1])
		}
	}

	return info, nil
}

func (c *Client) EditSourceCmd() *exec.Cmd {
	cmd := exec.Command(c.binary(), "edit")
	c.applyEditorEnv(cmd)
	return cmd
}

// Doctor runs `chezmoi doctor`. Returns output even on non-zero exit (doctor reports issues that way).
func (c *Client) Doctor() (string, error) {
	output, err := c.run("doctor")
	if err != nil {
		out := string(output)
		if out != "" {
			return out, nil
		}
		return "", fmt.Errorf("chezmoi doctor: %s: %w", strings.TrimSpace(out), err)
	}
	return string(output), nil
}

func (c *Client) EditConfigCmd() *exec.Cmd {
	cmd := exec.Command(c.binary(), "edit-config")
	c.applyEditorEnv(cmd)
	return cmd
}

func (c *Client) GitLog() (string, error) {
	output, err := c.run("git", "--", "log", "--oneline", "-20")
	if err != nil {
		return "", fmt.Errorf("chezmoi git log: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return string(output), nil
}

// GitLogUnpushed returns commits ahead of upstream. Returns "" if no upstream.
func (c *Client) GitLogUnpushed() (string, error) {
	output, err := c.run("git", "--", "log", "@{upstream}..HEAD", "--oneline")
	if err != nil {
		out := strings.TrimSpace(string(output))
		if strings.Contains(out, "no upstream") || strings.Contains(out, "unknown revision") {
			return "", nil
		}
		return "", fmt.Errorf("chezmoi git log unpushed: %s: %w", out, err)
	}
	return string(output), nil
}

// GitLogIncoming returns commits behind upstream. Returns "" if no upstream.
func (c *Client) GitLogIncoming() (string, error) {
	output, err := c.run("git", "--", "log", "HEAD..@{upstream}", "--oneline")
	if err != nil {
		out := strings.TrimSpace(string(output))
		if strings.Contains(out, "no upstream") || strings.Contains(out, "unknown revision") {
			return "", nil
		}
		return "", fmt.Errorf("chezmoi git log incoming: %s: %w", out, err)
	}
	return string(output), nil
}

func (c *Client) GitFetch() error {
	output, err := c.run("git", "--", "fetch")
	if err != nil {
		return fmt.Errorf("chezmoi git fetch: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// isValidGitHash checks that s is a plausible abbreviated or full git
// commit hash (hex-only, 4-64 chars). Rejects flag-shaped strings like
// "--help" since '-' is not a hex character.
func isValidGitHash(s string) bool {
	if len(s) < 4 || len(s) > 64 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

// GitShow runs `chezmoi git show` for a commit. Validates hash format first.
func (c *Client) GitShow(hash string) (string, error) {
	if !isValidGitHash(hash) {
		return "", fmt.Errorf("%w: %q", ErrInvalidHash, hash)
	}
	output, err := c.run("git", "--", "show", "--format=fuller", hash)
	if err != nil {
		out := string(output)
		if out != "" && !strings.HasPrefix(out, "error:") {
			return out, nil
		}
		return "", fmt.Errorf("chezmoi git show: %s: %w", strings.TrimSpace(out), err)
	}
	return string(output), nil
}

func (c *Client) GitPull() error {
	output, err := c.run("git", "--", "pull")
	if err != nil {
		return fmt.Errorf("chezmoi git pull: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

func (c *Client) Data() (string, error) {
	output, err := c.run("data", "--format=yaml")
	if err != nil {
		return "", fmt.Errorf("chezmoi data: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return string(output), nil
}

func (c *Client) DataJSON() (string, error) {
	output, err := c.run("data", "--format=json")
	if err != nil {
		return "", fmt.Errorf("chezmoi data --format=json: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return string(output), nil
}

// Verify runs `chezmoi verify`. Errors if target is out of date.
func (c *Client) Verify() error {
	output, err := c.run("verify")
	if err != nil {
		return fmt.Errorf("chezmoi verify: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return nil
}

// GitDiff runs `chezmoi git diff` for a file. Uses --cached when staged is true.
func (c *Client) GitDiff(path string, staged bool) (string, error) {
	args := []string{"git", "--", "diff"}
	if staged {
		args = append(args, "--cached")
	}
	args = append(args, "--", path)
	output, err := c.run(args...)
	if err != nil {
		out := string(output)
		if out != "" && !strings.HasPrefix(out, "error:") {
			return out, nil
		}
		return "", fmt.Errorf("chezmoi git diff: %s: %w", strings.TrimSpace(out), err)
	}
	return string(output), nil
}

// ReAddAll runs `chezmoi re-add --force` for all managed files.
func (c *Client) ReAddAll() (string, error) {
	output, err := c.run("re-add", "--force")
	if err != nil {
		return "", fmt.Errorf("chezmoi re-add: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return string(output), nil
}

// StatusText runs `chezmoi status` and returns raw output.
func (c *Client) StatusText() (string, error) {
	output, err := c.run("status")
	if err != nil {
		return "", fmt.Errorf("chezmoi status: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return string(output), nil
}

// DiffAll runs `chezmoi diff` with no file argument.
func (c *Client) DiffAll() (string, error) {
	output, err := c.run("diff")
	if err != nil {
		out := string(output)
		if out != "" {
			return out, nil
		}
		return "", fmt.Errorf("chezmoi diff: %s: %w", strings.TrimSpace(out), err)
	}
	return string(output), nil
}

// CatConfig runs `chezmoi cat-config`.
func (c *Client) CatConfig() (string, error) {
	output, err := c.run("cat-config")
	if err != nil {
		return "", fmt.Errorf("chezmoi cat-config: %s: %w", strings.TrimSpace(string(output)), err)
	}
	return string(output), nil
}

func (c *Client) InitCmd() *exec.Cmd {
	return exec.Command(c.binary(), "init")
}

// EditConfigTemplateCmd opens `chezmoi edit-config-template`.
// If no template exists, chezmoi creates one from the current config.
func (c *Client) EditConfigTemplateCmd() *exec.Cmd {
	cmd := exec.Command(c.binary(), "edit-config-template")
	c.applyEditorEnv(cmd)
	return cmd
}

// Archive runs `chezmoi archive --output=<path>`. Format is auto-detected from extension.
func (c *Client) Archive(outputPath string) error {
	_, err := c.run("archive", "--output="+outputPath)
	if err != nil {
		return fmt.Errorf("chezmoi archive: %w", err)
	}
	return nil
}
