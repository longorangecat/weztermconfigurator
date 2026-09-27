package ui

import (
	"weztermconfigurator/internal/state"
)

// osWriteFileAtomic writes content to path via state.WriteAtomic.
func osWriteFileAtomic(path, content string) error {
	return state.WriteAtomic(path, []byte(content))
}
