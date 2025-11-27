//go:build windows

package server

import (
	"os"
	"path/filepath"
)

func init() {
	wd, err := os.Getwd()
	if err != nil {
		return
	}

	base := filepath.Join(wd, "_tmp_tests")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return
	}

	_ = os.Setenv("TMP", base)
	_ = os.Setenv("TEMP", base)
}
