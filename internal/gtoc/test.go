package gtoc

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl.cii/document"
)

// newDocumentFrom creates a cii Document from a GOBL file in the `test/data` folder
func newDocumentFrom(name string) (*document.Document, error) {
	env, err := loadTestEnvelope(name)
	if err != nil {
		return nil, err
	}
	c := &Converter{}
	return c.Convert(env)
}

// loadTestEnvelope returns a GOBL Envelope from a file in the `test/data` folder
func loadTestEnvelope(name string) (*gobl.Envelope, error) {
	src, _ := os.Open(filepath.Join(getTestDataPath(), name))
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(src); err != nil {
		return nil, err
	}
	env := new(gobl.Envelope)
	if err := json.Unmarshal(buf.Bytes(), env); err != nil {
		return nil, err
	}

	return env, nil
}

// getTestDataPath returns the path to the `test/data/gtoc` folder
func getTestDataPath() string {
	return filepath.Join(getRootFolder(), "test", "data", "gtoc")
}

func getRootFolder() string {
	cwd, _ := os.Getwd()

	for !isRootFolder(cwd) {
		cwd = removeLastEntry(cwd)
	}
	return cwd
}

func isRootFolder(dir string) bool {
	files, _ := os.ReadDir(dir)

	for _, file := range files {
		if file.Name() == "go.mod" {
			return true
		}
	}

	return false
}

func removeLastEntry(dir string) string {
	lastEntry := "/" + filepath.Base(dir)
	i := strings.LastIndex(dir, lastEntry)
	return dir[:i]
}
