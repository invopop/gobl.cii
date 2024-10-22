package ctog

import (
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LoadTestXMLDoc returns a CII XMLDoc from a file in the test data folder
func LoadTestXMLDoc(name string) (*Document, error) {
	src, err := os.Open(filepath.Join(GetTestDataPath(), name))
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := src.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	inData, err := io.ReadAll(src)
	if err != nil {
		return nil, err
	}
	doc := new(Document)
	if err := xml.Unmarshal(inData, doc); err != nil {
		return nil, err
	}

	return doc, err
}

// GetDataGlob returns a list of files in the `test/data` folder that match the pattern
func GetDataGlob() ([]string, error) {
	return filepath.Glob(filepath.Join(GetTestDataPath(), "*.xml"))
}

// GetSchemaPath returns the path to the `test/data/schema` folder
func GetSchemaPath() string {
	return filepath.Join(GetTestDataPath(), "schema")
}

// GetOutPath returns the path to the `test/data/out` folder
func GetOutPath() string {
	return filepath.Join(GetTestDataPath(), "out")
}

// GetTestDataPath returns the path to the `test/data/ctog` folder
func GetTestDataPath() string {
	return filepath.Join(getRootFolder(), "test", "data", "ctog")
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
