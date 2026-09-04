//go:build unix

package chezmoi

import (
	"os/exec"
	"testing"
)

func TestDetachProcessSetsNewSession(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("true")
	detachProcess(cmd)

	if cmd.SysProcAttr == nil || !cmd.SysProcAttr.Setsid {
		t.Fatal("expected Setsid so children cannot open /dev/tty")
	}
	if cmd.Cancel == nil {
		t.Fatal("expected a Cancel func that kills the process group")
	}
	if cmd.WaitDelay != detachWaitDelay {
		t.Fatalf("WaitDelay = %s, want %s", cmd.WaitDelay, detachWaitDelay)
	}
}
