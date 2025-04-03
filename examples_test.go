package cii_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/invopop/gobl"
	cii "github.com/invopop/gobl.cii"
	"github.com/invopop/gobl/bill"
	"github.com/invopop/gobl/uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lestrrat-go/libxml2"
	"github.com/lestrrat-go/libxml2/xsd"
)

const (
	pathPatternXML  = "*.xml"
	pathPatternJSON = "*.json"
	pathConvert     = "convert"
	pathParse       = "parse"
	pathOut         = "out"

	staticUUID uuid.UUID = "0195ce71-dc9c-72c8-bf2c-9890a4a9f0a2"
)

// updateOut is a flag that can be set to update example files
var updateOut = flag.Bool("update", false, "Update the example files in test/data")

func TestConvertInvoice(t *testing.T) {
	schema, err := loadSchema("schema.xsd")
	require.NoError(t, err)

	examples := findSourceFiles(t, pathConvert, pathPatternJSON)
	for _, example := range examples {
		inName := filepath.Base(example)
		outName := strings.Replace(inName, ".json", ".xml", 1)

		t.Run(inName, func(t *testing.T) {
			out, err := newInvoiceFrom(t, inName)
			require.NoError(t, err)

			data, err := out.Bytes()
			require.NoError(t, err)

			if *updateOut {
				err = os.WriteFile(outputFilepath(pathConvert, outName), data, 0644)
				require.NoError(t, err)
				err = validateXML(schema, data)
				require.NoError(t, err)
			}

			output := loadOutputFile(t, pathConvert, outName)

			assert.Equal(t, string(output), string(data), "Output should match the expected XML. Update with --update flag.")
		})
	}
}

func TestParseInvoice(t *testing.T) {
	examples := findSourceFiles(t, pathParse, pathPatternXML)

	for _, example := range examples {
		inName := filepath.Base(example)
		outName := strings.Replace(inName, ".xml", ".json", 1)

		t.Run(inName, func(t *testing.T) {
			// Load XML data
			xmlData, err := os.ReadFile(example)
			require.NoError(t, err)

			// Convert CII XML to GOBL
			env, err := cii.ParseInvoice(xmlData)
			require.NoError(t, err)

			env.Head.UUID = staticUUID
			if inv, ok := env.Extract().(*bill.Invoice); ok {
				inv.UUID = staticUUID
			}
			require.NoError(t, env.Calculate())

			writeEnvelope(dataPath(pathParse, pathOut, outName), env)

			// Extract the invoice from the envelope
			inv, ok := env.Extract().(*bill.Invoice)
			require.True(t, ok, "Document should be an invoice")

			// Marshal only the invoice
			data, err := json.MarshalIndent(inv, "", "\t")
			require.NoError(t, err)

			// Load the expected output
			output := loadOutputFile(t, pathParse, outName)

			// Parse the expected output to extract the invoice
			var expectedEnv gobl.Envelope
			err = json.Unmarshal(output, &expectedEnv)
			require.NoError(t, err)

			expectedInvoice, ok := expectedEnv.Extract().(*bill.Invoice)
			require.True(t, ok, "Expected document should be an invoice")

			// Marshal the expected invoice
			expectedData, err := json.MarshalIndent(expectedInvoice, "", "  ")
			require.NoError(t, err)

			assert.JSONEq(t, string(expectedData), string(data), "Invoice should match the expected JSON. Update with --update flag.")
		})
	}
}

// newInvoiceFrom creates a cii Document from a GOBL file in the `test/data` folder
func newInvoiceFrom(t *testing.T, name string) (*cii.Invoice, error) {
	t.Helper()
	env := loadEnvelope(t, name)
	return cii.ConvertInvoice(env)
}

func parseInvoiceFrom(t *testing.T, name string) (*gobl.Envelope, error) {
	t.Helper()
	path := dataPath(pathParse, name)
	src, err := os.Open(path)
	require.NoError(t, err)
	defer func() {
		if cerr := src.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	data, err := io.ReadAll(src)
	if err != nil {
		require.NoError(t, err)
	}
	return cii.ParseInvoice(data)
}

// loadEnvelope returns a GOBL Envelope from a file in the `test/data/convert` folder
func loadEnvelope(t *testing.T, name string) *gobl.Envelope {
	t.Helper()
	path := dataPath(pathConvert, name)

	src, _ := os.Open(path)
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(src)
	require.NoError(t, err)

	env := new(gobl.Envelope)
	require.NoError(t, json.Unmarshal(buf.Bytes(), env))

	// Clear the IDs
	env.Head.UUID = staticUUID
	if inv, ok := env.Extract().(*bill.Invoice); ok {
		inv.UUID = staticUUID
	}
	require.NoError(t, env.Calculate())
	require.NoError(t, env.Validate())

	writeEnvelope(path, env)

	return env
}

func writeEnvelope(path string, env *gobl.Envelope) {
	if !*updateOut {
		return
	}
	data, err := json.MarshalIndent(env, "", "\t")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		panic(err)
	}
}

func loadSchema(name string) (*xsd.Schema, error) {
	return xsd.ParseFromFile(filepath.Join(schemaPath(), name))
}

// validateXML validates a XML document against a XSD Schema
func validateXML(schema *xsd.Schema, data []byte) error {
	xmlDoc, err := libxml2.Parse(data)
	if err != nil {
		return err
	}

	err = schema.Validate(xmlDoc)
	if err != nil {
		// Collect all errors into a single error message
		errors := err.(xsd.SchemaValidationError).Errors()
		var errorMessages []string
		for _, e := range errors {
			errorMessages = append(errorMessages, e.Error())
		}
		return fmt.Errorf("validation errors: %s", strings.Join(errorMessages, ",\n ")) // Return all errors as a single error
	}

	return nil
}

func outputFilepath(path, name string) string {
	return filepath.Join(dataPath(path, pathOut, name))
}

func loadOutputFile(t *testing.T, path, name string) []byte {
	t.Helper()
	src, err := os.Open(outputFilepath(path, name))
	require.NoError(t, err)
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(src); err != nil {
		require.NoError(t, err)
	}
	return buf.Bytes()
}

func findSourceFiles(t *testing.T, path, pattern string) []string {
	path = filepath.Join(dataPath(), path, pattern)
	files, err := filepath.Glob(path)
	require.NoError(t, err)
	return files
}

func schemaPath() string {
	return filepath.Join(dataPath(), "schema")
}

func dataPath(files ...string) string {
	files = append([]string{rootFolder(), "test", "data"}, files...)
	return filepath.Join(files...)
}

func rootFolder() string {
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
