package generate

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneratedGoVersions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the mock go command is a shell script")
	}

	tests := []struct {
		name       string
		version    string
		wantModule string
		wantImage  string
	}{
		{"missing toolchain", "", defaultGoModVersion, defaultImageGoModVersion},
		{"older toolchain", "go1.26.4", defaultGoModVersion, defaultImageGoModVersion},
		{"matching toolchain", "go1.27.1", "go 1.27.1", "golang:1.27.1-alpine"},
		{"newer patch", "go1.27.2", "go 1.27.2", "golang:1.27.2-alpine"},
		{"newer minor", "go1.28.0", "go 1.28.0", "golang:1.28.0-alpine"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.version != "" {
				script := "#!/bin/sh\necho 'go version " + tt.version + " linux/amd64'\n"
				require.NoError(t, os.WriteFile(filepath.Join(dir, "go"), []byte(script), 0755))
			}
			t.Setenv("PATH", dir)

			require.Equal(t, tt.wantModule, getLocalGoVersion())
			require.Equal(t, tt.wantImage, extractImageGoVersion())
		})
	}
}
