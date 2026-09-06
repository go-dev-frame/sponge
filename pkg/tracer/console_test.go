package tracer

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConsoleExporter(t *testing.T) {
	exporter, err := NewConsoleExporter()
	assert.NoError(t, err)
	assert.NotNil(t, exporter)
}

func TestNewFileExporter(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatal(err)
		}
	}()

	exporter, file, err := NewFileExporter("demo")
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, exporter)
	_ = file.Close()

	exporter, file, err = NewFileExporter("")
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, exporter)
	_ = file.Close()

	assert.Panics(t, func() {
		_, _, _ = NewFileExporter(t.TempDir())
	})
}

func Test_newExporter(t *testing.T) {
	exporter, err := newExporter(os.Stdout)
	assert.NoError(t, err)
	assert.NotNil(t, exporter)
}
