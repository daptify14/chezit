//go:build unix

package chezmoi

import (
	"os/exec"
	"syscall"
	"time"
)

// detachWaitDelay bounds the pipe wait after the process group is killed;
// a backstop only, the group kill normally closes the pipe immediately.
const detachWaitDelay = time.Second

// detachProcess runs cmd as the leader of a new session. Without a
// controlling terminal, git and ssh cannot open /dev/tty to prompt for
// credentials or host-key confirmation, so they fail fast with a classifiable
// message instead of writing over the TUI. Because the new session is also a
// new process group, cancellation kills every descendant, not just chezmoi,
// which would otherwise leave git or ssh holding the output pipe open past
// the timeout.
func detachProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = detachWaitDelay
}
