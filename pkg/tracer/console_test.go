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
	t.Chdir(t.TempDir())

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
