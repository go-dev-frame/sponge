// Package configs used to locate config file.
package configs

import (
	"path/filepath"
	"runtime"
)

var basePath string

func init() {
	_, currentFile, _, _ := runtime.Caller(0) //nolint
	basePath = filepath.Dir(currentFile)
}

// Location return absolute path of the configs yml file
func Location(rel string) string {
	if filepath.IsAbs(rel) {
		return rel
	}

	return filepath.Join(basePath, rel)
}
