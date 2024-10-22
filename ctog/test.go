package ctog

import (
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	xmlPattern = "*.xml"
)

// LoadTestXMLDoc returns a CII XMLDoc from a file in the test data folder
func LoadTestXMLDoc(name string) (*Document, error) {
	src, err := os.Open(filepath.Join(GetConversionTypePath(xmlPattern), name))
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
func GetDataGlob(pattern string) ([]string, error) {
	return filepath.Glob(filepath.Join(GetConversionTypePath(pattern), pattern))
}

// GetSchemaPath returns the path to the `test/data/schema` folder
func GetSchemaPath(pattern string) string {
	return filepath.Join(GetConversionTypePath(pattern), "schema")
}

// GetOutPath returns the path to the `test/data/out` folder
func GetOutPath(pattern string) string {
	return filepath.Join(GetConversionTypePath(pattern), "out")
}

// GetDataPath returns the path to the `test/data` folder
func GetDataPath() string {
	return filepath.Join(GetTestPath(), "data")
}

// GetConversionTypePath returns the path to the `test/data/ctog` or `test/data/gtoc` folder
func GetConversionTypePath(pattern string) string {
	if pattern == xmlPattern {
		return filepath.Join(GetDataPath(), "ctog")
	}
	return filepath.Join(GetDataPath(), "gtoc")
}

// GetTestPath returns the path to the `test` folder
func GetTestPath() string {
	return filepath.Join(getRootFolder(), "test")
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
