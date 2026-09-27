//go:build !windows

package wezcli

import (
	"os/exec"
)

func prepareCommand(cmd *exec.Cmd) {
	// No-op on non-Windows platforms
}
