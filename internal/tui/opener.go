package tui

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
)

type capability struct {
	Available bool
	Reason    string
}

var (
	cachedFileManagerCap capability
	fileManagerCapOnce   sync.Once
)

func fileManagerCapability() capability {
	fileManagerCapOnce.Do(func() {
		cachedFileManagerCap = computeFileManagerCapability()
	})
	return cachedFileManagerCap
}

func computeFileManagerCapability() capability {
	switch runtime.GOOS {
	case "darwin":
		if _, err := exec.LookPath("open"); err != nil {
			return capability{Available: false, Reason: "open command not found"}
		}
		return capability{Available: true}
	case "linux":
		if _, err := exec.LookPath("xdg-open"); err != nil {
			return capability{Available: false, Reason: "xdg-open command not found"}
		}
		if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
			return capability{Available: false, Reason: "no GUI session detected"}
		}
		return capability{Available: true}
	default:
		return capability{Available: false, Reason: "unsupported platform"}
	}
}

func openPath(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path).Start()
	case "linux":
		return exec.Command("xdg-open", path).Start()
	default:
		return fmt.Errorf("open path not supported on platform %s", runtime.GOOS)
	}
}
