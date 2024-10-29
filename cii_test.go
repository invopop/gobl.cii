package cii

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/invopop/gobl"
	ctog "github.com/invopop/gobl.cii/ctog"
	gtoc "github.com/invopop/gobl.cii/gtoc"

	"github.com/invopop/gobl/bill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	xmlPattern  = "*.xml"
	jsonPattern = "*.json"
)

var update = flag.Bool("update", false, "Update out directory")

func TestNewDocument(t *testing.T) {
	// schema, err := LoadSchema("schema.xsd")
	// require.NoError(t, err)

	examples, err := getDataGlob("*.json")
	require.NoError(t, err)

	for _, example := range examples {
		inName := filepath.Base(example)
		outName := strings.Replace(inName, ".json", ".xml", 1)

		t.Run(inName, func(t *testing.T) {
			doc, err := NewDocumentFrom(inName)
			require.NoError(t, err)

			data, err := doc.Bytes()
			require.NoError(t, err)

			// err = test.ValidateXML(schema, data)
			// require.NoError(t, err)

			output, err := loadOutputFile(outName)
			assert.NoError(t, err)

			if *update {
				err = saveOutputFile(outName, data)
				require.NoError(t, err)
			} else {
				assert.Equal(t, output, data, "Output should match the expected XML. Update with --update flag.")
			}
		})
	}
}

func TestNewDocumentGOBL(t *testing.T) {
	examples, err := getDataGlob("*.xml")
	require.NoError(t, err)

	for _, example := range examples {
		inName := filepath.Base(example)
		outName := strings.Replace(inName, ".xml", ".json", 1)

		t.Run(inName, func(t *testing.T) {
			// Load XML data
			xmlData, err := os.ReadFile(example)
			require.NoError(t, err)

			// Create a new conversor
			conversor := ctog.NewConverter()

			// Convert CII XML to GOBL
			goblEnv, err := conversor.ConvertToGOBL(xmlData)
			require.NoError(t, err)

			// Extract the invoice from the envelope
			invoice, ok := goblEnv.Extract().(*bill.Invoice)
			require.True(t, ok, "Document should be an invoice")

			// Remove UUID from the invoice
			invoice.UUID = ""

			// Marshal only the invoice
			data, err := json.MarshalIndent(invoice, "", "  ")
			require.NoError(t, err)

			// Load the expected output
			output, err := loadOutputFile(outName)
			assert.NoError(t, err)

			// Parse the expected output to extract the invoice
			var expectedEnv gobl.Envelope
			err = json.Unmarshal(output, &expectedEnv)
			require.NoError(t, err)

			expectedInvoice, ok := expectedEnv.Extract().(*bill.Invoice)
			require.True(t, ok, "Expected document should be an invoice")

			// Remove UUID from the expected invoice
			expectedInvoice.UUID = ""

			// Marshal the expected invoice
			expectedData, err := json.MarshalIndent(expectedInvoice, "", "  ")
			require.NoError(t, err)

			if *update {
				err = saveOutputFile(outName, data)
				require.NoError(t, err)
			} else {
				assert.JSONEq(t, string(expectedData), string(data), "Invoice should match the expected JSON. Update with --update flag.")
			}
		})
	}
}

// NewDocumentFrom creates a cii Document from a GOBL file in the `test/data` folder
func NewDocumentFrom(name string) (*gtoc.Document, error) {
	env, err := LoadTestEnvelope(name)
	if err != nil {
		return nil, err
	}
	c := &gtoc.Converter{}
	return c.ConvertToCII(env)
}

// LoadTestXMLDoc returns a CII XMLDoc from a file in the test data folder
func LoadTestXMLDoc(name string) (*ctog.Document, error) {
	src, err := os.Open(filepath.Join(getConversionTypePath(xmlPattern), name))
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
	doc := new(ctog.Document)
	if err := xml.Unmarshal(inData, doc); err != nil {
		return nil, err
	}

	return doc, err
}

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
	src, _ := os.Open(filepath.Join(getConversionTypePath(jsonPattern), name))
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

func loadOutputFile(name string) ([]byte, error) {
	var pattern string
	if strings.HasSuffix(name, ".json") {
		pattern = xmlPattern
	} else {
		pattern = jsonPattern
	}
	src, _ := os.Open(filepath.Join(getOutPath(pattern), name))
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(src); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func saveOutputFile(name string, data []byte) error {
	var pattern string
	if strings.HasSuffix(name, jsonPattern) {
		pattern = xmlPattern
	} else {
		pattern = jsonPattern
	}
	return os.WriteFile(filepath.Join(getOutPath(pattern), name), data, 0644)
}

func getDataGlob(pattern string) ([]string, error) {
	return filepath.Glob(filepath.Join(getConversionTypePath(pattern), pattern))
}

// func getSchemaPath(pattern string) string {
// 	return filepath.Join(getConversionTypePath(pattern), "schema")
// }

func getOutPath(pattern string) string {
	return filepath.Join(getConversionTypePath(pattern), "out")
}

func getDataPath() string {
	return filepath.Join(getTestPath(), "data")
}

func getConversionTypePath(pattern string) string {
	if pattern == xmlPattern {
		return filepath.Join(getDataPath(), "ctog")
	}
	return filepath.Join(getDataPath(), "gtoc")
}

func getTestPath() string {
	return filepath.Join(getRootFolder(), "test")
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
