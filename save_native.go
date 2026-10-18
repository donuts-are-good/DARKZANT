//go:build !js

package main

import (
	"os"
	"path/filepath"
)

// canQuit is false in the browser, where there is nothing to quit to.
const canQuit = true

func savePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "darkzant", "save.json")
}

func writeSave(b []byte) error {
	if err := os.MkdirAll(filepath.Dir(savePath()), 0o755); err != nil {
		return err
	}
	return os.WriteFile(savePath(), b, 0o644)
}

func readSave() ([]byte, error) { return os.ReadFile(savePath()) }
