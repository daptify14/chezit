//go:build !unix

package chezmoi

import (
	"os/exec"
	"time"
)

const detachWaitDelay = time.Second

// detachProcess on non-Unix platforms can only bound the pipe wait; there is
// no session or process group to detach from.
func detachProcess(cmd *exec.Cmd) {
	cmd.WaitDelay = detachWaitDelay
}
