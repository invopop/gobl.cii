package gtoc

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/invopop/gobl"
	"github.com/invopop/gobl/bill"
)

// LoadTestInvoice returns a GOBL Invoice from a file in the `test/data` folder
func LoadTestInvoice(name string) (*bill.Invoice, error) {
	env, err := LoadTestEnvelope(name)
	if err != nil {
		return nil, err
	}

	return env.Extract().(*bill.Invoice), nil
}

// LoadTestEnvelope returns a GOBL Envelope from a file in the `test/data` folder
func LoadTestEnvelope(name string) (*gobl.Envelope, error) {
	src, _ := os.Open(filepath.Join(GetTestDataPath(), name))
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

// NewDocumentFrom creates a cii Document from a GOBL file in the `test/data` folder
func NewDocumentFrom(name string) (*Document, error) {
	env, err := LoadTestEnvelope(name)
	if err != nil {
		return nil, err
	}
	c := &Conversor{}
	return c.ConvertToCII(env)
}

// GetTestDataPath returns the path to the `test/data/gtoc` folder
func GetTestDataPath() string {
	return filepath.Join(getRootFolder(), "test", "data", "gtoc")
}

// TODO: adapt to new folder structure
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
